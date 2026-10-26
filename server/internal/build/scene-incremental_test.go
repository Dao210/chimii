package build

import (
	"context"
	"testing"
	"time"
)

func TestSceneIncrementalCacheReusesOnlyUnchangedGeometry(t *testing.T) {
	r := AssemblyRecipe{Version: 4, Title: "Tower", Subject: "tower", Summary: "Layered tower", Scene: &SceneSpec{Version: 1, Brief: "A layered tower", Objects: []SceneObject{{ID: "main", Label: "Tower", Family: "architecture", Modules: []Construction{
		sceneModule("lower", "solid", 0, 0, 0, 4, 24, 4, 4), sceneModule("upper", "solid", 0, 24, 0, 4, 12, 4, 15),
	}}}}}
	cache := NewSceneCompileCache()
	compile := func() CompileResult {
		result, err := CompileSceneWithCache(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0), cache)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	first := compile()
	if first.Solver.CompileStages < 3 {
		t.Fatal("tower was not compiled incrementally")
	}
	second := compile()
	if second.Solver.ReusedStages != first.Solver.CompileStages || second.Plan.ContentHash != first.Plan.ContentHash {
		t.Fatal("identical prefix was not reused deterministically")
	}
	r.Scene.Objects[0].Modules[1].Color = 1
	third := compile()
	if third.Solver.ReusedStages != 2 || third.Plan.ContentHash == first.Plan.ContentHash {
		t.Fatalf("upper edit invalidated valid lower geometry or reused stale colors: %+v", third.Solver)
	}
	r.Scene.Objects[0].Modules[0].Color = 2
	fourth := compile()
	if fourth.Solver.ReusedStages != 0 {
		t.Fatal("changed lower geometry reused stale placements")
	}
	changed := CatalogCopy(StarterCatalog)
	p := changed["brick-2x4"]
	p.Quantity++
	changed[p.ID] = p
	result, err := CompileSceneWithCache(context.Background(), r, UnlimitedInventory(), CatalogVersion, changed, time.Unix(0, 0), cache)
	if err != nil || result.Solver.ReusedStages != 0 {
		t.Fatalf("changed catalog reused stale compilation: %v", err)
	}
	if result.Solver.Visited > maxSceneSearchNodes {
		t.Fatal("stages reset the shared node budget")
	}
}

func TestSceneIncrementalCancellationCannotReturnCachedSuccess(t *testing.T) {
	cache := NewSceneCompileCache()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CompileSceneWithCache(ctx, sceneFixture("figure"), UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0), cache); err == nil {
		t.Fatal("cancelled compilation returned success")
	}
}
