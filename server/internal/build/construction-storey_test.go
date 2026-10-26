package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func storeyFixture() AssemblyRecipe {
	r := AssemblyRecipe{Version: 4, Title: "两层小屋", Subject: "architecture", Summary: "带窗洞、通层楼梯和柱网楼板", Requirements: []string{"门窗", "楼梯", "二层平台", "屋顶"}, Scene: &SceneSpec{Version: 1, Brief: "两层白色小屋，真实通层楼梯洞、横梁楼板和阶梯红屋顶", Objects: []SceneObject{{ID: "house", Label: "两层小屋", Family: "architecture"}}}}
	base := sceneModule("base", "solid", 0, 0, 0, 14, 6, 18, 71)
	base.Attach = &ConstructionAttachment{Anchor: "ground"}
	lower := sceneModule("lower", "storey", 0, 0, 0, 14, 18, 18, 15)
	lower.Attach = &ConstructionAttachment{Parent: "base", Anchor: "top"}
	lower.Openings = []WallOpening{{Face: "front", X: 3, Y: 0, Width: 2, Height: 9}, {Face: "left", X: 4, Y: 3, Width: 2, Height: 6}, {Face: "right", X: 10, Y: 3, Width: 2, Height: 6}}
	lower.CeilingOpenings = []DeckOpening{{X: 2, Z: 3, Width: 2, Depth: 12}}
	lower.Requirements = []int{0, 2}
	stairs := sceneModule("stairs", "stairs", 0, 0, 0, 2, 18, 12, 19)
	stairs.Attach = &ConstructionAttachment{Parent: "base", Anchor: "top", Offset: DesignVector{X: -4}}
	stairs.Requirements = []int{1}
	upper := lower
	upper.ID, upper.Label, upper.CeilingOpenings = "upper", "upper", nil
	upper.Attach = &ConstructionAttachment{Parent: "lower", Anchor: "top"}
	roof := sceneModule("roof", "roof", 0, 0, 0, 14, 9, 18, 4)
	roof.Attach = &ConstructionAttachment{Parent: "upper", Anchor: "top"}
	roof.Requirements = []int{3}
	r.Scene.Objects[0].Modules = []Construction{base, lower, stairs, upper, roof}
	return r
}

func TestStoreyStairAccessKeepsOpeningsAndCompiles(t *testing.T) {
	r := storeyFixture()
	lower := r.Scene.Objects[0].Modules[1]
	cells, _, err := constructionCells(lower)
	if err != nil {
		t.Fatal(err)
	}
	for y := lower.Size.Y - 6; y < lower.Size.Y; y++ {
		if cells[DesignVector{2, y, 3}] || cells[DesignVector{3, y, 14}] {
			t.Fatal("stair/access opening was filled by a ceiling layer")
		}
	}
	if cells[DesignVector{3, 0, 0}] || cells[DesignVector{0, 3, 4}] || !cells[DesignVector{6, 6, 6}] {
		t.Fatal("storey filled a wall opening or lost interior support")
	}
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Plan.Validation.Buildable {
		t.Fatal(result.Plan.Validation.Issues)
	}
	t.Logf("storey assembly: %d parts, %d steps", len(result.Plan.Placements), len(result.Plan.Steps))
	if directory := os.Getenv("CHIMII_SCENE_STOREY_OUTPUT"); directory != "" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(directory, "storey-fixture.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r.Scene.Objects[0].Modules[1].CeilingOpenings = nil
	if _, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0)); err == nil {
		t.Fatal("ceiling blocked the through-storey stairs without rejection")
	}
}

func TestStoreyRejectsUnsafeOpeningAndThinLintel(t *testing.T) {
	m := storeyFixture().Scene.Objects[0].Modules[1]
	m.CeilingOpenings = []DeckOpening{{X: 0, Z: 3, Width: 2, Depth: 12}}
	if _, _, err := constructionCells(m); err == nil {
		t.Fatal("ceiling hole erased the exterior bearing rim")
	}
	m = storeyFixture().Scene.Objects[0].Modules[1]
	m.Openings[0].Height = 12
	if _, _, err := constructionCells(m); err == nil {
		t.Fatal("window consumed the load-bearing lintel below ceiling")
	}
}

func TestGeneratedSizingNormalizesWheelAssemblyWithoutClampingBounds(t *testing.T) {
	r := sceneFixture("vehicle")
	r.Scene.Objects[0].Modules[0].Size = DesignVector{7, 6, 11}
	r.Scene.Objects[0].Modules[0].Attach = &ConstructionAttachment{Anchor: "ground"}
	r.Scene.Objects[0].Modules[0].Position.Y = 5
	fitted := FitGeneratedScene(r)
	if fitted.Scene.Objects[0].Modules[0].Size != (DesignVector{8, 5, 12}) || fitted.Scene.Objects[0].Modules[0].Position.Y != 0 || r.Scene.Objects[0].Modules[0].Size.Y != 6 {
		t.Fatal("wheelbase preferences were not normalized immutably")
	}
	r.Scene.Objects[0].Modules[0].Size.X = 18
	if err := ValidateScene(FitGeneratedScene(r)); err == nil {
		t.Fatal("oversized wheelbase was clipped to hide its unsupported footprint")
	}
	r.Scene.Objects[0].Modules[0].Size.X = -1
	if err := ValidateScene(FitGeneratedScene(r)); err == nil {
		t.Fatal("negative input was interpreted as a valid preference")
	}
}
