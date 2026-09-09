package buildeval

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chimii-ai/chimii/server/internal/build"
)

// CandidateBatch is an offline boundary, not a new production recipe format.
type CandidateBatch struct {
	Version    int         `json:"version"`
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	ID        string                  `json:"id"`
	Prompt    string                  `json:"prompt"`
	Bricks    string                  `json:"bricks"`
	Color     int                     `json:"color"`
	Inventory build.InventorySnapshot `json:"inventory"`
	Source    CandidateSource         `json:"source"`
}

type CandidateSource struct {
	Kind        string `json:"kind"`
	Revision    string `json:"revision"`
	ObjectID    string `json:"object_id,omitempty"`
	StructureID string `json:"structure_id,omitempty"`
	Split       string `json:"split,omitempty"`
	Seed        *int   `json:"seed,omitempty"`
	Model       string `json:"model,omitempty"`
	ModelHash   string `json:"model_hash,omitempty"`
}

var brickLine = regexp.MustCompile(`^(\d+)x(\d+) \((\d+),(\d+),(\d+)\)$`)
var upstreamDimensions = map[[2]int]bool{{1, 1}: true, {1, 2}: true, {1, 4}: true, {1, 6}: true, {1, 8}: true, {2, 2}: true, {2, 4}: true, {2, 6}: true}

func ImportCandidates(batch CandidateBatch, catalog build.PartCatalog) ([]Case, error) {
	if batch.Version != 1 || len(batch.Candidates) == 0 || len(batch.Candidates) > 1000 {
		return nil, fmt.Errorf("invalid candidate batch version or size")
	}
	cases := make([]Case, 0, len(batch.Candidates))
	ids := map[string]bool{}
	for _, candidate := range batch.Candidates {
		if candidate.ID == "" || ids[candidate.ID] {
			return nil, fmt.Errorf("empty or duplicate candidate id")
		}
		ids[candidate.ID] = true
		if candidate.Source.Revision == "" || (candidate.Source.Kind != "brickgpt" && candidate.Source.Kind != "stabletext2brick" && candidate.Source.Kind != "fixture") {
			return nil, fmt.Errorf("candidate source needs kind and revision")
		}
		if candidate.Source.Kind == "stabletext2brick" && (candidate.Source.ObjectID == "" || candidate.Source.StructureID == "" || candidate.Source.Split == "") {
			return nil, fmt.Errorf("dataset candidate needs object, structure and split provenance")
		}
		if candidate.Source.Kind == "brickgpt" && (candidate.Source.Seed == nil || candidate.Source.Model == "" || len(candidate.Source.ModelHash) != 64) {
			return nil, fmt.Errorf("generated candidate needs model, weight hash and seed provenance")
		}
		p, code := importBrickText(candidate.Bricks, candidate.Color, catalog)
		c := Case{ID: candidate.ID, Family: candidate.Source.Kind, Prompt: candidate.Prompt, Expectation: "observe", Placements: p, Inventory: candidate.Inventory, SourceCandidate: &candidate, PreflightError: code}
		cases = append(cases, c)
	}
	return cases, nil
}

func importBrickText(text string, color int, catalog build.PartCatalog) ([]build.Placement, string) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(text) > 32768 || len(lines) == 0 || len(lines) > 200 {
		return nil, "candidate_size_unsupported"
	}
	keys := make([]string, 0, len(catalog))
	for id := range catalog {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	placements := []build.Placement{}
	minX, minZ, maxX, maxZ := 20, 20, 0, 0
	for i, line := range lines {
		match := brickLine.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			return nil, "candidate_invalid_format"
		}
		values := [5]int{}
		for j := range values {
			n, err := strconv.Atoi(match[j+1])
			if err != nil || n < 0 || n > 20 {
				return nil, "candidate_out_of_bounds"
			}
			values[j] = n
		}
		sx, sz, x, z, layer := values[0], values[1], values[2], values[3], values[4]
		a, b := sx, sz
		if a > b {
			a, b = b, a
		}
		if !upstreamDimensions[[2]int{a, b}] {
			return nil, "candidate_unknown_dimension"
		}
		if x+sx > 20 || z+sz > 20 || layer >= 20 {
			return nil, "candidate_out_of_bounds"
		}
		// Chimii target designs currently end at 48 plates. Do not scale a
		// taller learned candidate or shift its lowest layer onto the ground.
		if (layer+1)*3 > 48 {
			return nil, "candidate_height_unsupported"
		}
		id, rotation := "", 0
		for _, key := range keys {
			part := catalog[key]
			if !build.IsShapePartEligible(part) || part.PlatesY != 3 {
				continue
			}
			if part.StudsX == sx && part.StudsZ == sz {
				id = key
				break
			}
			if part.StudsX == sz && part.StudsZ == sx {
				id, rotation = key, 90
				break
			}
		}
		if id == "" {
			return nil, "candidate_part_unavailable"
		}
		p := placed(fmt.Sprintf("brickgpt-%03d", i+1), id, x, layer*3, z, i+1)
		p.Rotation, p.Color, p.Module = rotation, color, "brickgpt-candidate"
		placements = append(placements, p)
		minX, minZ, maxX, maxZ = min(minX, x), min(minZ, z), max(maxX, x+sx), max(maxZ, z+sz)
	}
	for i := range placements {
		placements[i].X -= minX + (maxX-minX)/2
		placements[i].Z -= minZ + (maxZ-minZ)/2
	}
	return placements, ""
}
