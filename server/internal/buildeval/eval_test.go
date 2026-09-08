package buildeval

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/chimii-ai/chimii/server/internal/build"
)

func TestBaselineCorpusContractsAndReproducibility(t *testing.T) {
	cases := BaselineCases()
	if len(cases) != 100 {
		t.Fatalf("want 100 cases, got %d", len(cases))
	}
	families, ids := map[string]int{}, map[string]bool{}
	for _, c := range cases {
		if ids[c.ID] {
			t.Fatal("duplicate case", c.ID)
		}
		ids[c.ID], families[c.Family] = true, families[c.Family]+1
	}
	if len(families) != 10 {
		t.Fatal("missing case families")
	}
	for family, n := range families {
		if n != 10 {
			t.Fatal(family, n)
		}
	}
	before := Hash(cases)
	first, second := Run(context.Background(), cases), Run(context.Background(), BaselineCases())
	for i, a := range first.Cases {
		if !a.ContractPassed {
			t.Errorf("%s: %s %v", a.ID, a.ErrorCode, a.InvariantErrors)
		}
		b := second.Cases[i]
		if a.InputHash != b.InputHash || a.OutputHash != b.OutputHash {
			t.Errorf("%s was not reproducible", a.ID)
		}
	}
	if before != Hash(cases) {
		t.Fatal("inventory, edit source or input was mutated")
	}
	if first.Summary.Observed != 40 {
		t.Fatal("observations were hidden among expected cases")
	}
}

func TestCandidateConversionPreservesDimensionsHeightAndOrder(t *testing.T) {
	p, code := importBrickText("2x4 (7,8,0)\n2x2 (7,8,1)", 4, build.StarterCatalog)
	if code != "" {
		t.Fatal(code)
	}
	if len(p) != 2 || p[0].Rotation != 90 || p[0].X != -1 || p[0].Z != -2 || p[1].Y != 3 || p[1].Step != 2 || p[0].Color != 4 {
		t.Fatalf("wrong axis, origin, plate-height or ordering: %#v", p)
	}
	if v := build.Validate(p, build.UnlimitedInventory()); !v.Buildable {
		t.Fatal(v.Issues)
	}
	// Never silently ground a floating source by subtracting its first Z value.
	floating, code := importBrickText("2x2 (0,0,2)", 1, build.StarterCatalog)
	if code != "" || floating[0].Y != 6 || build.Validate(floating, build.UnlimitedInventory()).Buildable {
		t.Fatal("source height was silently repaired")
	}
}

func TestCandidateImportRejectsUnsupportedInputsWithoutDroppingParts(t *testing.T) {
	for text, want := range map[string]string{
		"": "candidate_invalid_format", "2x4 (-1,0,0)": "candidate_invalid_format",
		"2x4 (999999999999999999999,0,0)": "candidate_out_of_bounds",
		"2x4 (19,0,0)":                    "candidate_out_of_bounds", "3x3 (0,0,0)": "candidate_unknown_dimension",
		"1x8 (0,0,0)": "candidate_part_unavailable", "2x2 (0,0,16)": "candidate_height_unsupported",
		strings.Repeat("2x2 (0,0,0)\n", 201): "candidate_size_unsupported",
	} {
		p, code := importBrickText(text, 1, build.StarterCatalog)
		if code != want || len(p) != 0 {
			t.Errorf("%q: %s %#v", text, code, p)
		}
	}
}

func TestCandidateInventoryAndSourceBoundaries(t *testing.T) {
	inv := build.NewInventorySnapshot(true, 1, []build.InventoryItem{{PartID: "brick-2x2", Color: 1, Quantity: 1}})
	candidate := Candidate{ID: "sample", Bricks: "2x2 (0,0,0)\n2x2 (0,0,1)", Color: 1, Inventory: inv, Source: CandidateSource{Kind: "fixture", Revision: "v1"}}
	batch := CandidateBatch{Version: 1, Candidates: []Candidate{candidate}}
	cases, err := ImportCandidates(batch)
	if err != nil {
		t.Fatal(err)
	}
	r := Run(context.Background(), cases)
	if r.Summary.Accepted != 0 || r.Cases[0].Validation.Buildable {
		t.Fatal("candidate bypassed inventory validation")
	}
	if !reflect.DeepEqual(candidate.Inventory, inv) {
		t.Fatal("inventory mutated")
	}
	batch.Candidates = append(batch.Candidates, candidate)
	if _, err := ImportCandidates(batch); err == nil {
		t.Fatal("duplicate identity accepted")
	}
	batch.Candidates = batch.Candidates[:1]
	batch.Candidates[0].Source.Kind = "stabletext2brick"
	if _, err := ImportCandidates(batch); err == nil {
		t.Fatal("dataset without object provenance accepted")
	}
	batch.Candidates[0].Source = CandidateSource{Kind: "fixture", Revision: "v1"}
	batch.Candidates[0].Bricks = "1x8 (0,0,0)"
	cases, _ = ImportCandidates(batch)
	r = Run(context.Background(), cases)
	if len(r.Cases) != 1 || r.Cases[0].ErrorCode != "candidate_part_unavailable" {
		t.Fatal("failed candidate disappeared from denominator")
	}
}

func TestCancellationAndPreservedEditViolationsAreVisible(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := Run(ctx, BaselineCases()[:1])
	if r.Summary.ContractFailed != 1 || r.Cases[0].ErrorCode != "evaluation_cancelled" {
		t.Fatal("cancelled evaluation reported success")
	}
	for _, c := range BaselineCases() {
		if c.Family != "edit-opening" {
			continue
		}
		c.Recipe.Design.Shapes[1].Size.X++
		r = Run(context.Background(), []Case{c})
		if r.Summary.ContractFailed != 1 {
			t.Fatal("edit preservation violation was hidden")
		}
		if _, err := json.Marshal(r); err != nil {
			t.Fatal(err)
		}
		break
	}
}
