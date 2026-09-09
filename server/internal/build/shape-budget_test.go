package build

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestCompileDesignRetainsFailureDiagnostics(t *testing.T) {
	recipe := designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1))
	for _, reason := range []string{"cancelled", "deadline", "inventory_insufficient"} {
		t.Run(reason, func(t *testing.T) {
			ctx := context.Background()
			inventory := UnlimitedInventory()
			switch reason {
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "deadline":
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Unix(0, 0))
				defer cancel()
			case "inventory_insufficient":
				inventory = NewInventorySnapshot(true, 1, nil)
			}
			result, err := CompileDesign(ctx, recipe, inventory, CatalogVersion, StarterCatalog, time.Unix(0, 0))
			if err == nil || result.Solver == nil || result.Solver.StopReason != reason || result.Solver.TargetCells != 48 || result.Solver.Visited != 0 {
				t.Fatalf("missing failure diagnostics: %#v, %v", result.Solver, err)
			}
			if len(result.Plan.Placements) != 0 {
				t.Fatal("failed search returned a partial product")
			}
		})
	}
}

func TestSolverDiagnosticsDoNotChangeProductJSONOrHash(t *testing.T) {
	r, err := CompileDesign(context.Background(), designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1)), UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil || r.Solver == nil || r.Solver.StopReason != "feasible" {
		t.Fatalf("missing success diagnostics: %v", err)
	}
	before, _ := json.Marshal(r)
	r.Solver.RepairNodes = 999
	r.Plan.Document.Solver.StopReason = "diagnostic-only"
	r.Plan.Document.Solver.RepairAttempts = 123
	after, _ := json.Marshal(r)
	if !reflect.DeepEqual(before, after) || physicalContentHash(r.Plan) != r.Plan.ContentHash {
		t.Fatal("operational diagnostics changed product JSON or physical hash")
	}
}

func TestRepairConsumesSharedBudgetExactlyOnce(t *testing.T) {
	f, s := loadSeamRepairFixture(t)
	s.budget.used, s.budget.limit = 13, 14
	if _, ok := s.repairSeams(f.Placements); ok {
		t.Fatal("one remaining node cannot repair this fixture")
	}
	if s.budget.used != 14 || s.budget.repairNodes != 1 || s.budget.repairAttempts != 1 || s.budget.repairs != 0 || !s.limited {
		t.Fatalf("repair acquired a separate budget or counted twice: %+v", s.budget)
	}
}
