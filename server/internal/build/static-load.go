package build

import (
	"context"
	"math"
	"strings"
)

const StaticLoadModelVersion = "reference-vertical-equilibrium-v1"

type StaticLoadEstimate struct {
	Status           string  `json:"status"`
	Model            string  `json:"model"`
	Reason           string  `json:"reason,omitempty"`
	MassSource       string  `json:"mass_source"`
	ClutchPerStudN   float64 `json:"clutch_per_stud_n"`
	MaximumResidual  float64 `json:"maximum_residual,omitempty"`
	MaximumPullRatio float64 `json:"maximum_pull_ratio,omitempty"`
	Physical         string  `json:"physical"`
}

// Reference masses are measured standard-brick values from StableLego Table I.
// They do not calibrate the user's kit or certify this simplified contact model.
var referenceBrickMass = map[string]struct {
	x, z  int
	grams float64
}{
	"3005.dat": {1, 1, 0.43}, "3004.dat": {2, 1, 0.81}, "3010.dat": {4, 1, 1.57}, "3009.dat": {6, 1, 2.28}, "3008.dat": {8, 1, 3.03}, "3003.dat": {2, 2, 1.15}, "3001.dat": {4, 2, 2.16}, "2456.dat": {6, 2, 3.23},
}

type loadContact struct {
	upper, lower int
	x, z, pull   float64
}

// EstimateStaticLoad tests gravity-only vertical force and overturning moments.
// Shear, torsion, deformation, wear, insertion and dynamic loads are outside its
// domain. Unsupported inputs never become a positive stability conclusion.
func EstimateStaticLoad(ctx context.Context, plan BuildPlan, clutchPerStudN float64) StaticLoadEstimate {
	r := StaticLoadEstimate{Status: "not_evaluated", Model: StaticLoadModelVersion, MassSource: "https://arxiv.org/html/2402.10711v2#S4.T1", ClutchPerStudN: clutchPerStudN, Physical: "not_tested"}
	unknown := func(reason string) StaticLoadEstimate { r.Reason = reason; return r }
	if math.IsNaN(clutchPerStudN) || math.IsInf(clutchPerStudN, 0) || clutchPerStudN < 0 || clutchPerStudN > 10 {
		r.ClutchPerStudN = 0 // Unknown evidence must remain valid JSON.
		return unknown("invalid clutch parameter")
	}
	if len(plan.Placements) == 0 || len(plan.Placements) > 32 {
		return unknown("model supports 1..32 standard bricks")
	}
	ids := map[string]int{}
	mass := make([]float64, len(plan.Placements))
	centers := make([]supportPoint, len(mass))
	contacts := []loadContact{}
	for i, p := range plan.Placements {
		if err := ctx.Err(); err != nil {
			return unknown(err.Error())
		}
		spec, ok := plan.Parts[p.PartID]
		entry, known := referenceBrickMass[strings.ToLower(spec.LDrawID)]
		if !ok || !known || !partIsMechanicallyCertified(spec) || spec.GeometryProfile != "stud_tube_rect" || spec.StudsX != entry.x || spec.StudsZ != entry.z || spec.PlatesY != 3 || p.Rotation%90 != 0 || p.Y < 0 {
			return unknown("part mass or mechanical profile is outside the reference model: " + p.PartID)
		}
		if _, duplicate := ids[p.ID]; duplicate || p.ID == "" {
			return unknown("invalid placement identity")
		}
		ids[p.ID] = i
		mass[i] = entry.grams * 0.00981
		size := orientedSizeWith(p, plan.Parts)
		centers[i] = supportPoint{float64(p.X) + float64(size.x)/2, float64(p.Z) + float64(size.z)/2}
		if p.Y == 0 {
			for _, q := range []supportPoint{{float64(p.X), float64(p.Z)}, {float64(p.X + size.x), float64(p.Z)}, {float64(p.X + size.x), float64(p.Z + size.z)}, {float64(p.X), float64(p.Z + size.z)}} {
				contacts = append(contacts, loadContact{i, -1, q.x, q.z, 0})
			}
		}
	}
	// Re-derive contacts from immutable catalog connectors, not caller annotations.
	for _, connection := range deriveExactConnections(plan.Placements, plan.Parts) {
		if connection.Kind != "stud" {
			return unknown("non-stud connection is outside the reference model")
		}
		a, b := ids[connection.APlacementID], ids[connection.BPlacementID]
		pa, pb := plan.Placements[a], plan.Placements[b]
		upper, lower := a, b
		if pa.Y < pb.Y {
			upper, lower = b, a
		}
		if topYWith(plan.Placements[lower], plan.Parts) != plan.Placements[upper].Y {
			return unknown("non-vertical stud connection")
		}
		for _, id := range connection.AConnectorIDs {
			var point connectorPosition
			found := false
			for _, c := range plan.Parts[pa.PartID].Connectors {
				if c.ID == id {
					point, _ = transformConnector(pa, plan.Parts[pa.PartID], c)
					found = true
					break
				}
			}
			if !found {
				return unknown("missing contact geometry")
			}
			// Four bounded samples approximate a stud's vertical force envelope.
			// The radius and aggregate clutch parameter remain model assumptions.
			for _, delta := range []supportPoint{{-0.15, -0.15}, {-0.15, 0.15}, {0.15, -0.15}, {0.15, 0.15}} {
				contacts = append(contacts, loadContact{upper, lower, float64(point.x)/20 + delta.x, float64(point.z)/20 + delta.z, clutchPerStudN / 4})
			}
		}
	}
	if len(contacts) > 256 {
		return unknown("contact budget exceeded")
	}
	rows := len(mass) * 3
	matrix := make([][]float64, rows)
	rhs := make([]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, len(contacts))
	}
	for i, m := range mass {
		rhs[i*3] = m
	}
	for j, c := range contacts {
		for _, side := range []struct {
			i    int
			sign float64
		}{{c.upper, 1}, {c.lower, -1}} {
			if side.i < 0 {
				continue
			}
			i, s := side.i, side.sign
			matrix[i*3][j] = s
			matrix[i*3+1][j] = s * (c.x - centers[i].x)
			matrix[i*3+2][j] = s * (c.z - centers[i].z)
		}
	}
	// Each signed contact force is x - pull_limit, with x >= 0. Ground
	// contacts have no tensile reaction and therefore no hidden fixed baseplate.
	a := make([][]float64, 0, rows*2)
	b := make([]float64, 0, rows*2)
	for i, row := range matrix {
		value := rhs[i]
		for j, c := range contacts {
			value += row[j] * c.pull
		}
		positive := append([]float64{}, row...)
		negative := make([]float64, len(row))
		for j, v := range row {
			negative[j] = -v
		}
		a = append(a, positive, negative)
		b = append(b, value, -value)
	}
	x, status := linearFeasibility(ctx, a, b, 20000)
	if status == "infeasible" {
		r.Status = "infeasible_in_model"
		return r
	}
	if status != "feasible" {
		return unknown(status)
	}
	for i, row := range matrix {
		residual := -rhs[i]
		for j, c := range contacts {
			residual += row[j] * (x[j] - c.pull)
		}
		r.MaximumResidual = math.Max(r.MaximumResidual, math.Abs(residual))
	}
	for j, c := range contacts {
		if c.pull > 0 {
			r.MaximumPullRatio = math.Max(r.MaximumPullRatio, math.Max(0, c.pull-x[j])/c.pull)
		}
	}
	if r.MaximumResidual > 1e-7 || r.MaximumPullRatio > 1+1e-7 {
		return unknown("numerical residual exceeds model tolerance")
	}
	r.Status = "feasible_in_model"
	return r
}
