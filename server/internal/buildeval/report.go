package buildeval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"time"

	"github.com/chimii-ai/chimii/server/internal/build"
)

type CaseResult struct {
	Case
	InputHash       string                  `json:"input_hash"`
	OutputHash      string                  `json:"output_hash"`
	ErrorCode       string                  `json:"error_code,omitempty"`
	Validation      *build.ValidationReport `json:"validation,omitempty"`
	Plan            *build.BuildPlan        `json:"plan,omitempty"`
	PlanningMS      *float64                `json:"planning_ms"`
	CompileMS       float64                 `json:"compile_including_validation_ms"`
	ValidationMS    float64                 `json:"validation_recheck_ms"`
	InvariantErrors []string                `json:"invariant_errors"`
	ContractPassed  bool                    `json:"contract_passed"`
	Solver          *SolverDiagnostics      `json:"solver,omitempty"`
}

// Keep operational statistics out of the product's BuildDocument contract.
type SolverDiagnostics struct {
	build.SolverReport
	RepairNodes    int    `json:"repair_nodes"`
	RepairAttempts int    `json:"repair_attempts"`
	Repairs        int    `json:"repairs"`
	Attempts       int    `json:"attempts"`
	Pruned         int    `json:"pruned"`
	StopReason     string `json:"stop_reason"`
}

type Report struct {
	Version        int               `json:"version"`
	CorpusVersion  string            `json:"corpus_version"`
	CorpusHash     string            `json:"corpus_hash"`
	CatalogVersion string            `json:"catalog_version"`
	CatalogHash    string            `json:"catalog_hash"`
	SourceHash     string            `json:"source_hash,omitempty"`
	CatalogAssets  []CatalogAsset    `json:"catalog_assets,omitempty"`
	Validator      string            `json:"validator_version"`
	PhysicsProfile string            `json:"physics_profile"`
	Runtime        string            `json:"runtime"`
	GeneratedAt    time.Time         `json:"generated_at"`
	Scope          string            `json:"scope"`
	Catalog        build.PartCatalog `json:"catalog"`
	Cases          []CaseResult      `json:"cases"`
	Summary        Summary           `json:"summary"`
}

type Summary struct {
	Total          int            `json:"total"`
	Accepted       int            `json:"accepted"`
	Rejected       int            `json:"rejected"`
	Observed       int            `json:"observed"`
	Asserted       int            `json:"asserted"`
	ContractFailed int            `json:"contract_failed"`
	Errors         map[string]int `json:"errors"`
	CompileP50MS   float64        `json:"compile_p50_ms"`
	CompileP95MS   float64        `json:"compile_p95_ms"`
}

func Hash(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // All call sites hash finite, JSON-owned fixture values.
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func Run(ctx context.Context, cases []Case, catalog build.PartCatalog) Report {
	r := Report{Version: 1, CorpusVersion: CorpusVersion, CorpusHash: Hash(cases), CatalogVersion: build.CatalogVersion, Validator: build.ValidatorVersion, PhysicsProfile: build.PhysicsProfileVersion, Runtime: runtime.Version() + "/" + runtime.GOOS + "/" + runtime.GOARCH, GeneratedAt: time.Now().UTC(), Scope: "offline compiler and validator; no LLM, production DB, or physical verification", Catalog: catalog, CatalogHash: Hash(catalog), Cases: []CaseResult{}, Summary: Summary{Errors: map[string]int{}}}
	var durations []float64
	for _, c := range cases {
		x := CaseResult{Case: c, InputHash: Hash(c), InvariantErrors: []string{}}
		if err := ctx.Err(); err != nil {
			x.ErrorCode = "evaluation_cancelled"
		} else if c.PreflightError != "" {
			x.ErrorCode = c.PreflightError
		} else if c.Recipe != nil {
			start := time.Now()
			result, err := build.CompileDesign(ctx, *c.Recipe, c.Inventory, build.CatalogVersion, catalog, time.Unix(0, 0))
			if s := result.Solver; s != nil {
				x.Solver = &SolverDiagnostics{SolverReport: *s, RepairNodes: s.RepairNodes, RepairAttempts: s.RepairAttempts, Repairs: s.Repairs, Attempts: s.Attempts, Pruned: s.Pruned, StopReason: s.StopReason}
			}
			x.CompileMS = float64(time.Since(start)) / float64(time.Millisecond)
			durations = append(durations, x.CompileMS)
			if err != nil {
				x.ErrorCode, _ = build.BuildErrorCode(err)
				if x.ErrorCode == "" {
					x.ErrorCode = "compile_error"
				}
			}
			if len(result.Plan.Placements) > 0 {
				x.Plan = &result.Plan
				x.Placements = result.Plan.Placements
				x.Validation = &result.Plan.Validation
				start = time.Now()
				checked := build.ValidateWithCatalog(x.Placements, c.Inventory, catalog)
				x.ValidationMS = float64(time.Since(start)) / float64(time.Millisecond)
				if !reflect.DeepEqual(checked, result.Plan.Validation) {
					x.InvariantErrors = append(x.InvariantErrors, "validation_recheck_changed")
				}
			}
			if err == nil {
				x.InvariantErrors = append(x.InvariantErrors, checkDesign(*c.Recipe, result.Plan)...)
			}
		} else {
			start := time.Now()
			v := build.ValidateWithCatalog(c.Placements, c.Inventory, catalog)
			x.Validation, x.ValidationMS = &v, float64(time.Since(start))/float64(time.Millisecond)
			if !v.Buildable {
				x.ErrorCode = build.BuildErrorStructureInvalid
				for _, issue := range v.Issues {
					if issue.Code == "inventory_exceeded" {
						x.ErrorCode = build.BuildErrorInsufficientInventory
						break
					}
				}
			}
		}
		if c.Source != nil && c.Recipe != nil && c.Recipe.Design != nil {
			for _, id := range c.PreserveIDs {
				before, after := findShape(c.Source, id), findShape(c.Recipe.Design, id)
				if before == nil || !reflect.DeepEqual(before, after) {
					x.InvariantErrors = append(x.InvariantErrors, "edited_preserved_shape:"+id)
				}
			}
		}
		if Hash(c) != x.InputHash {
			x.InvariantErrors = append(x.InvariantErrors, "input_mutated")
		}
		accepted := x.ErrorCode == "" && x.Validation != nil && x.Validation.Buildable
		x.ContractPassed = len(x.InvariantErrors) == 0
		switch c.Expectation {
		case "accept":
			x.ContractPassed = x.ContractPassed && accepted
		case "reject":
			x.ContractPassed = x.ContractPassed && !accepted && x.ErrorCode != "evaluation_cancelled"
		case "observe":
			r.Summary.Observed++
		default:
			x.ContractPassed = x.ContractPassed && x.ErrorCode == c.Expectation
		}
		if x.ErrorCode == "evaluation_cancelled" {
			x.ContractPassed = false
		}
		x.OutputHash = Hash(struct {
			Code       string
			Placements []build.Placement
			Validation *build.ValidationReport
			Invariants []string
		}{x.ErrorCode, x.Placements, x.Validation, x.InvariantErrors})
		r.Cases = append(r.Cases, x)
		r.Summary.Total++
		if c.Expectation != "observe" {
			r.Summary.Asserted++
		}
		if accepted {
			r.Summary.Accepted++
		} else {
			r.Summary.Rejected++
		}
		if !x.ContractPassed {
			r.Summary.ContractFailed++
		}
		if x.ErrorCode != "" {
			r.Summary.Errors[x.ErrorCode]++
		}
	}
	sort.Float64s(durations)
	if len(durations) > 0 {
		r.Summary.CompileP50MS = durations[(len(durations)-1)*50/100]
		r.Summary.CompileP95MS = durations[(len(durations)-1)*95/100]
	}
	return r
}

func findShape(d *build.DesignSpec, id string) *build.ShapeNode {
	for _, s := range d.Shapes {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

// Independently compare sampled target cells with the actual placed parts,
// rather than trusting the solver's matched-cell counter or success flag.
func checkDesign(recipe build.AssemblyRecipe, plan build.BuildPlan) []string {
	errors := []string{}
	target, err := build.RasterizeDesign(*recipe.Design)
	if err != nil {
		return []string{"invalid_target"}
	}
	type placedCell struct {
		color int
		node  string
	}
	actual := map[build.DesignVector]placedCell{}
	for _, p := range plan.Placements {
		part := plan.Parts[p.PartID]
		sx, sz := part.StudsX, part.StudsZ
		if p.Rotation%180 != 0 {
			sx, sz = sz, sx
		}
		for x := 0; x < sx; x++ {
			for y := 0; y < part.PlatesY; y++ {
				for z := 0; z < sz; z++ {
					q := build.DesignVector{X: p.X + x, Y: p.Y + y, Z: p.Z + z}
					if _, exists := actual[q]; exists {
						errors = append(errors, "overlapping_cell")
					}
					actual[q] = placedCell{p.Color, p.Module}
				}
			}
		}
	}
	if len(target) != len(actual) {
		errors = append(errors, "target_cell_count_changed")
	}
	for q, cell := range target {
		placed, ok := actual[q]
		if !ok || placed.node != cell.Node || (recipe.Constraints.ExactColors && placed.color != cell.Color) {
			errors = append(errors, fmt.Sprintf("target_cell_changed:%v", q))
			break
		}
	}
	if recipe.Constraints.PartCount > 0 && len(plan.Placements) != recipe.Constraints.PartCount {
		errors = append(errors, "exact_count_changed")
	}
	return errors
}
