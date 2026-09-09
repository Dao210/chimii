package build

import (
	"context"
	"testing"
)

func TestShapeCapacityBoundsUseAvailableEligibleParts(t *testing.T) {
	r := designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1))
	target, _ := RasterizeDesign(*r.Design)
	s := &shapeSearch{target: target, catalog: StarterCatalog, inventory: NewInventorySnapshot(true, 1, nil),
		parts: []PartSpec{StarterCatalog["brick-2x4"], StarterCatalog["brick-1x1"]}, occupied: map[DesignVector]int{}, remaining: map[inventoryKey]int{}, exactCount: 2}
	s.remaining[inventoryKey{"brick-1x1", 1}] = 16
	if s.canCoverRemaining() {
		t.Fatal("unowned large bricks hid an impossible exact count")
	}
	s.remaining[inventoryKey{"brick-2x4", 1}] = 2
	if !s.canCoverRemaining() {
		t.Fatal("two owned bricks can cover this model")
	}
	s.exactCount = 19
	if s.canCoverRemaining() {
		t.Fatal("accepted fewer available pieces than the exact count")
	}
	s.exactCount = 0
	s.remaining[inventoryKey{"brick-2x4", 1}], s.remaining[inventoryKey{"brick-1x1", 1}] = 1, 7
	if s.canCoverRemaining() {
		t.Fatal("insufficient remaining volume was not pruned")
	}
}

func TestImpossibleExactCountStopsBeforeEnumeratingLayouts(t *testing.T) {
	r := designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1))
	r.Constraints.PartCount = 1
	_, report, err := SolveDesign(context.Background(), r, UnlimitedInventory(), StarterCatalog)
	if err == nil || report.Visited != 1 || report.RepairAttempts != 0 {
		t.Fatalf("necessary count bound did not stop early: %+v, %v", report, err)
	}
}

// Trigger cancellation after search has started, without wall-clock sleeps or
// reading a search's mutable counters concurrently from the test.
type cancelDuringSearch struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *cancelDuringSearch) Err() error {
	c.checks++
	if c.checks == 30 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestShapeSearchCooperativelyCancelsDuringEnumeration(t *testing.T) {
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelDuringSearch{Context: base, cancel: cancel}
	_, r, err := SolveDesign(ctx, clockRecipe(), UnlimitedInventory(), StarterCatalog)
	if err == nil || r.StopReason != "cancelled" || r.Visited == 0 || r.Visited >= maxShapeSearchNodes {
		t.Fatalf("mid-search cancellation was lost: %+v, %v", r, err)
	}
}
