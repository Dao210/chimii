package build

import (
	"context"
	"encoding/json"
	"math"
	"testing"
)

func TestStaticLoadRespectsClutchAndFreeGround(t *testing.T) {
	cat := sceneCatalog()
	plan := BuildPlan{Parts: cat, Placements: []Placement{
		{ID: "base", PartID: "brick-2x4", Color: 4, Step: 1},
		{ID: "neck", PartID: "brick-1x2", Color: 4, X: 2, Y: 3, Step: 2},
		{ID: "ballast", PartID: "brick-1x2", Color: 4, Y: 3, Step: 2},
		{ID: "beam", PartID: "long-8", Color: 4, X: 2, Y: 6, Step: 3},
	}}
	if !ValidateWithCatalog(plan.Placements, UnlimitedInventory(), cat).Buildable {
		t.Fatal("load fixture must pass geometry")
	}
	strong := EstimateStaticLoad(context.Background(), plan, 0.98)
	weak := EstimateStaticLoad(context.Background(), plan, 0)
	if strong.Status != "feasible_in_model" || weak.Status != "infeasible_in_model" {
		t.Fatalf("clutch not modeled: strong=%+v weak=%+v", strong, weak)
	}
	if strong.Physical != "not_tested" || strong.MaximumResidual > 1e-7 {
		t.Fatal("estimate lost its evidence boundary")
	}
	plan.Placements = plan.Placements[1:]
	if r := EstimateStaticLoad(context.Background(), plan, 0.98); r.Status != "infeasible_in_model" {
		t.Fatalf("hidden ground anchor supported a floating structure: %+v", r)
	}
}

func TestStaticLoadUnsupportedAndCancelledAreNotSuccess(t *testing.T) {
	plan := BuildPlan{Parts: StarterCatalog, Placements: []Placement{{ID: "plate", PartID: "plate-2x4"}}}
	if r := EstimateStaticLoad(context.Background(), plan, 0.98); r.Status != "not_evaluated" {
		t.Fatalf("unknown mass silently guessed: %+v", r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r := EstimateStaticLoad(ctx, plan, 0.98); r.Status != "not_evaluated" {
		t.Fatal("cancellation accepted")
	}
	plan.Placements = []Placement{{ID: "brick", PartID: "brick-2x4"}, {ID: "upper", PartID: "brick-2x4", Y: 3}}
	if r := EstimateStaticLoad(context.Background(), plan, 0); r.Status != "feasible_in_model" {
		t.Fatalf("compressed tower rejected: %+v", r)
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), -1, 11} {
		r := EstimateStaticLoad(context.Background(), plan, value)
		if _, err := json.Marshal(r); err != nil || r.Status != "not_evaluated" {
			t.Fatalf("invalid parameter corrupted evidence: %+v %v", r, err)
		}
	}
}
