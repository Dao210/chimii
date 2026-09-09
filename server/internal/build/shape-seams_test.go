package build

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestShapeCompilerResolvesInventorySeams(t *testing.T) {
	// These three complete recipes exhausted the v1 compiler's 24,000-node
	// budget. They need different seams, not more stock or a smaller target.
	for _, tc := range []struct {
		name  string
		w, d  int
		stock [4]int
	}{
		{"square", 5, 5, [4]int{12, 9, 5, 0}},
		{"wide", 7, 4, [4]int{14, 7, 7, 0}},
		{"narrow", 8, 3, [4]int{14, 5, 4, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := designRecipe(shape("solid", "box", 0, 0, 0, tc.w, 6, tc.d, 1))
			items := []InventoryItem{}
			for i, id := range []string{"brick-1x1", "brick-1x2", "brick-2x2", "brick-2x4"} {
				if tc.stock[i] > 0 {
					items = append(items, InventoryItem{PartID: id, Color: 1, Quantity: tc.stock[i]})
					r.Constraints.PartCount += tc.stock[i]
				}
			}
			inv := NewInventorySnapshot(true, 1, items)
			before, _ := json.Marshal([]any{r, inv})
			result, err := CompileDesign(context.Background(), r, inv, CatalogVersion, StarterCatalog, time.Unix(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			if result.Plan.GeneratorVersion != ShapeGeneratorVersion || result.Plan.Document.Solver.Visited > maxShapeSearchNodes || len(result.Plan.Placements) != r.Constraints.PartCount {
				t.Fatal("lost generator identity, budget or exact part count")
			}
			assertSeamTarget(t, r, result.Plan.Placements)
			if v := ValidateWithCatalog(result.Plan.Placements, inv, StarterCatalog); !v.Buildable {
				t.Fatal(v.Issues)
			}
			again, err := CompileDesign(context.Background(), r, inv, CatalogVersion, StarterCatalog, time.Unix(1, 0))
			if err != nil || again.Plan.ContentHash != result.Plan.ContentHash {
				t.Fatal("nondeterministic repair", err)
			}
			after, _ := json.Marshal([]any{r, inv})
			if string(before) != string(after) {
				t.Fatal("mutated recipe or reusable inventory")
			}
		})
	}
}

func TestShapeCompilerKeepsUnconstrainedBridgeSearch(t *testing.T) {
	// The eight-stud span needs the last original orientation seed. New seam
	// priorities must not replace all of the unrestricted search directions.
	r := designRecipe(shape("left", "box", 0, 0, 0, 2, 3, 2, 1),
		shape("right", "box", 6, 0, 0, 2, 3, 2, 1), shape("beam", "box", 0, 3, 0, 8, 6, 2, 1))
	result, err := CompileDesign(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil || !result.Plan.Validation.Buildable {
		t.Fatal("lost unconstrained bridge layout", err)
	}
	assertSeamTarget(t, r, result.Plan.Placements)
}

type seamRepairFixture struct {
	Recipe     AssemblyRecipe    `json:"recipe"`
	Placements []Placement       `json:"placements"`
	Inventory  InventorySnapshot `json:"inventory"`
}

func loadSeamRepairFixture(t *testing.T) (seamRepairFixture, *shapeSearch) {
	t.Helper()
	raw, err := os.ReadFile("testdata/brickgpt-seam.json")
	if err != nil {
		t.Fatal(err)
	}
	var f seamRepairFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	target, err := RasterizeDesign(*f.Recipe.Design)
	if err != nil {
		t.Fatal(err)
	}
	s := &shapeSearch{budget: &searchBudget{ctx: context.Background(), limit: maxShapeSearchNodes}, target: target, cells: sortedTargetCells(target), catalog: StarterCatalog,
		inventory: f.Inventory, remaining: f.Inventory.quantities(), occupied: map[DesignVector]int{}, exactColors: true,
		exactCount: f.Recipe.Constraints.PartCount, limit: maxShapeSearchNodes, seamPriority: true}
	for _, p := range StarterCatalog {
		if p.GeometryProfile == "stud_tube_rect" && partIsMechanicallyCertified(p) {
			s.parts = append(s.parts, p)
		}
	}
	sort.Slice(s.parts, func(i, j int) bool { return s.parts[i].ID < s.parts[j].ID })
	return f, s
}

func TestShapeRepairExpandsCriticalNeighborhood(t *testing.T) {
	f, s := loadSeamRepairFixture(t)
	if _, count := layoutComponents(f.Placements, StarterCatalog); count <= 1 {
		t.Fatal("fixture must start disconnected")
	}
	before, _ := json.Marshal(f)
	output, ok := s.repairSeams(f.Placements)
	if !ok || len(output) != len(f.Placements) || s.budget.used > maxShapeSearchNodes {
		t.Fatal("repair failed or exceeded count/budget")
	}
	if v := ValidateWithCatalog(output, f.Inventory, StarterCatalog); !v.Buildable {
		t.Fatal(v.Issues)
	}
	assertSeamTarget(t, f.Recipe, output)
	after, _ := json.Marshal(f)
	if string(before) != string(after) {
		t.Fatal("successful repair mutated the input")
	}
}

func TestShapeRepairFailurePreservesInputAndBudget(t *testing.T) {
	for _, reason := range []string{"budget", "cancelled", "missing-stock", "exact-count"} {
		t.Run(reason, func(t *testing.T) {
			f, s := loadSeamRepairFixture(t)
			s.budget.limit = 200
			switch reason {
			case "budget":
				s.budget.limit = 1
			case "cancelled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				s.budget.ctx = ctx
			case "missing-stock":
				s.inventory.Items = []InventoryItem{{PartID: "brick-2x4", Color: 1, Quantity: 1}}
			case "exact-count":
				s.exactCount = 1
			}
			before, _ := json.Marshal([]any{f, s.inventory, s.placements})
			remaining := maps.Clone(s.remaining)
			if _, ok := s.repairSeams(f.Placements); ok {
				t.Fatal("accepted invalid repair")
			}
			if s.budget.used > s.budget.limit {
				t.Fatal("exceeded global node budget")
			}
			after, _ := json.Marshal([]any{f, s.inventory, s.placements})
			if string(before) != string(after) || !reflect.DeepEqual(remaining, s.remaining) {
				t.Fatal("failed repair mutated input or inventory")
			}
		})
	}
}

func TestSeamDepthStopsAtSolidCrossingAndAir(t *testing.T) {
	s := &shapeSearch{occupied: map[DesignVector]int{}}
	for y := 0; y < 6; y++ {
		s.occupied[DesignVector{0, y, 0}], s.occupied[DesignVector{1, y, 0}] = 2*y, 2*y+1
	}
	if s.seamDepth(DesignVector{Y: 6}, 2, 1) != 6 {
		t.Fatal("lost continuous vertical seam")
	}
	s.occupied[DesignVector{1, 2, 0}] = s.occupied[DesignVector{0, 2, 0}]
	if s.seamDepth(DesignVector{Y: 6}, 2, 1) != 3 {
		t.Fatal("did not stop at a crossing brick")
	}
	delete(s.occupied, DesignVector{1, 5, 0})
	if s.seamDepth(DesignVector{Y: 6}, 2, 1) != 0 {
		t.Fatal("treated an opening as a brick seam")
	}
}

func assertSeamTarget(t *testing.T, recipe AssemblyRecipe, placements []Placement) {
	t.Helper()
	target, err := RasterizeDesign(*recipe.Design)
	if err != nil {
		t.Fatal(err)
	}
	actual := designTarget{}
	for _, p := range placements {
		size := orientedSizeWith(p, StarterCatalog)
		for x := 0; x < size.x; x++ {
			for y := 0; y < StarterCatalog[p.PartID].PlatesY; y++ {
				for z := 0; z < size.z; z++ {
					cell := DesignVector{p.X + x, p.Y + y, p.Z + z}
					if _, exists := actual[cell]; exists {
						t.Fatal("overlapping parts")
					}
					actual[cell] = targetCell{p.Module, p.Color}
				}
			}
		}
	}
	if !reflect.DeepEqual(actual, target) {
		t.Fatal("changed target geometry, feature identity or color")
	}
}
