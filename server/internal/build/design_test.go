package build

import (
	"context"
	"testing"
	"time"
)

func designRecipe(shapes ...ShapeNode) AssemblyRecipe {
	return AssemblyRecipe{Version: 3, Subject: "custom", Archetype: "custom", Title: "Custom", Metadata: map[string]string{},
		Constraints: RecipeConstraints{ExactColors: true}, Design: &DesignSpec{Version: 1, Mode: "static", Shapes: shapes}}
}

func shape(id, kind string, x, y, z, sx, sy, sz, color int) ShapeNode {
	return ShapeNode{ID: id, Label: id, Kind: kind, Operation: "add", Position: DesignVector{x, y, z}, Size: DesignVector{sx, sy, sz}, Color: color}
}

func clockRecipe() AssemblyRecipe {
	return designRecipe(
		shape("dial", "ellipse", -5, 0, -5, 10, 6, 10, 15),
		shape("minute", "box", -1, 6, -3, 1, 3, 4, 1),
		shape("hour", "box", 0, 6, 0, 3, 3, 1, 4),
		shape("north", "box", 0, 6, -4, 1, 3, 1, 14),
		shape("south", "box", 0, 6, 3, 1, 3, 1, 14),
		shape("west", "box", -4, 6, 0, 1, 3, 1, 14),
		shape("east", "box", 3, 6, 0, 1, 3, 1, 14),
	)
}

func TestShapeDesignsCompileRealPartsAndPreserveTargets(t *testing.T) {
	polygon := shape("outline", "polygon", 0, 0, 0, 6, 6, 6, 2)
	polygon.Points = []DesignPoint{{0, 0}, {6, 0}, {6, 2}, {2, 2}, {2, 6}, {0, 6}}
	ring := shape("ring", "ellipse", -5, 0, -5, 10, 6, 10, 1)
	hole := shape("hole", "ellipse", -2, 0, -2, 4, 6, 4, 1)
	hole.Operation = "subtract"
	tower := shape("stack", "box", 0, 0, 0, 2, 3, 2, 4)
	tower.Repeat = &ShapeRepeat{Count: 3, Offset: DesignVector{Y: 3}}
	for name, recipe := range map[string]AssemblyRecipe{
		"clock": clockRecipe(), "polygon": designRecipe(polygon), "ring": designRecipe(ring, hole),
		"slab":   designRecipe(shape("surface", "box", 0, 0, 0, 8, 6, 6, 14)),
		"repeat": designRecipe(tower),
	} {
		t.Run(name, func(t *testing.T) {
			result, err := Compile(recipe, UnlimitedInventory(), time.Unix(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			if !result.Plan.Validation.Buildable || result.Plan.Document.Solver.MatchedCells != result.Plan.Document.Solver.TargetCells {
				t.Fatal("invalid or incomplete design")
			}
			if result.Plan.Document.Solver.Visited > maxShapeSearchNodes {
				t.Fatal("search exceeded the global budget")
			}
			target, _ := RasterizeDesign(*recipe.Design)
			actual := designTarget{}
			for _, p := range result.Plan.Placements {
				spec := result.Plan.Parts[p.PartID]
				size := orientedSizeWith(p, result.Plan.Parts)
				for x := 0; x < size.x; x++ {
					for z := 0; z < size.z; z++ {
						for y := 0; y < spec.PlatesY; y++ {
							q := DesignVector{p.X + x, p.Y + y, p.Z + z}
							if _, ok := actual[q]; ok {
								t.Fatal("collision")
							}
							actual[q] = targetCell{p.Module, p.Color}
						}
					}
				}
			}
			if len(actual) != len(target) {
				t.Fatal("shape size changed")
			}
			for p, v := range target {
				if actual[p] != v {
					t.Fatalf("feature lost at %#v", p)
				}
			}
			second, err := Compile(recipe, UnlimitedInventory(), time.Unix(123, 0))
			if err != nil || second.Plan.ContentHash != result.Plan.ContentHash || second.MPD != result.MPD {
				t.Fatal("nondeterministic compilation")
			}
			t.Logf("%d parts, %d nodes", len(result.Plan.Placements), result.Plan.Document.Solver.Visited)
		})
	}
}

func TestShapeSolverHonorsExactCountAndResizing(t *testing.T) {
	r := designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1))
	r.Constraints.PartCount = 2
	first, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil || len(first.Plan.Placements) != 2 {
		t.Fatalf("exact count: %v", err)
	}
	r.Design.Shapes[0].Size.Y = 9
	r.Constraints.PartCount = 3
	resized, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil || len(resized.Plan.Placements) != 3 || resized.Plan.Document.DesignHash == first.Plan.Document.DesignHash || resized.Plan.ContentHash == first.Plan.ContentHash {
		t.Fatalf("resizing: %v", err)
	}
	r.Constraints.PartCount = 1
	if _, err = Compile(r, UnlimitedInventory(), time.Unix(0, 0)); err == nil {
		t.Fatal("silently ignored exact count")
	}
	// A deliberately tiny global budget must stop both tiling and seam repair.
	target, _ := RasterizeDesign(*clockRecipe().Design)
	s := &shapeSearch{budget: &searchBudget{ctx: context.Background(), limit: 1}, target: target, cells: sortedTargetCells(target), catalog: StarterCatalog,
		inventory: UnlimitedInventory(), remaining: map[inventoryKey]int{}, occupied: map[DesignVector]int{}, limit: 1}
	for _, p := range StarterCatalog {
		if p.GeometryProfile == "stud_tube_rect" && partIsMechanicallyCertified(p) {
			s.parts = append(s.parts, p)
		}
	}
	if s.walk(0) || !s.limited || s.budget.used != 1 {
		t.Fatal("search did not obey its budget")
	}
}

func TestShapeSolverRespectsReusablePartCountsAndCancellation(t *testing.T) {
	r := designRecipe(shape("solid", "box", 0, 0, 0, 4, 6, 2, 1))
	inv := NewInventorySnapshot(true, 1, []InventoryItem{{PartID: "brick-2x4", Color: 1, Quantity: 2}})
	for i := 0; i < 2; i++ {
		if _, err := Compile(r, inv, time.Unix(0, 0)); err != nil {
			t.Fatal(err)
		}
	}
	inv.Items[0].Quantity = 1
	if _, err := Compile(r, inv, time.Unix(0, 0)); err == nil {
		t.Fatal("exceeded per-model parts")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, report, err := SolveDesign(ctx, r, UnlimitedInventory(), StarterCatalog)
	if code, _ := BuildErrorCode(err); code != BuildErrorSearchLimit || report.Status != "search_limit" {
		t.Fatal("cancellation misreported as impossible")
	}
}

func TestDesignRejectsInvalidOrErasedFeatures(t *testing.T) {
	for name, mutate := range map[string]func(*AssemblyRecipe){
		"mode":      func(r *AssemblyRecipe) { r.Design.Mode = "working-clock" },
		"unknown":   func(r *AssemblyRecipe) { r.Design.Shapes[0].Kind = "clock" },
		"oversize":  func(r *AssemblyRecipe) { r.Design.Shapes[0].Size.X = 1000000 },
		"duplicate": func(r *AssemblyRecipe) { r.Design.Shapes = append(r.Design.Shapes, r.Design.Shapes[0]) },
		"repeat_overflow": func(r *AssemblyRecipe) {
			r.Design.Shapes[0].Repeat = &ShapeRepeat{Count: 2, Offset: DesignVector{X: -int(^uint(0)>>1) - 1}}
		},
		"erased": func(r *AssemblyRecipe) {
			p := r.Design.Shapes[0]
			p.ID = "cut"
			p.Operation = "subtract"
			r.Design.Shapes = append(r.Design.Shapes, p)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := designRecipe(shape("base", "box", 0, 0, 0, 4, 6, 4, 1))
			mutate(&r)
			if _, err := RasterizeDesign(*r.Design); err == nil {
				t.Fatal("accepted invalid design")
			}
		})
	}
}
