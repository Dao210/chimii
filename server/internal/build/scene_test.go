package build

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func sceneModule(id, kind string, x, y, z, sx, sy, sz, color int) Construction {
	return Construction{ID: id, Label: id, Kind: kind, Position: DesignVector{x, y, z}, Size: DesignVector{sx, sy, sz}, Color: color}
}

func sceneFixture(family string) AssemblyRecipe {
	mods := []Construction{}
	switch family {
	case "architecture":
		mods = append(mods, sceneModule("base", "solid", 0, 0, 0, 16, 6, 12, 71))
		front := sceneModule("front", "wall", 0, 6, 0, 16, 12, 2, 15)
		front.Openings = []WallOpening{{X: 6, Y: 0, Width: 4, Height: 9}}
		mods = append(mods, front, sceneModule("back", "wall", 0, 6, 10, 16, 12, 2, 15), sceneModule("left", "solid", 0, 6, 2, 2, 12, 8, 15), sceneModule("right", "solid", 14, 6, 2, 2, 12, 8, 15))
		mods = append(mods, sceneModule("stairs", "stairs", 2, 6, 2, 2, 12, 8, 4))
		for i, z := range []int{2, 6} {
			mods = append(mods, sceneModule("column"+string(rune('a'+i)), "solid", 8, 6, z, 2, 12, 2, 15))
		}
		mods = append(mods, sceneModule("floor", "solid", 0, 18, 0, 16, 6, 12, 71), sceneModule("upper", "frame", 0, 24, 0, 16, 12, 12, 15), sceneModule("roof", "roof", 0, 36, 0, 16, 12, 12, 2))
	case "vehicle":
		mods = append(mods, sceneModule("chassis", "wheelbase", 0, 0, 0, 8, 5, 12, 71), sceneModule("body", "solid", 0, 5, 0, 8, 6, 12, 4), sceneModule("cabin", "frame", 1, 11, 1, 6, 9, 10, 15), sceneModule("roof", "solid", 1, 20, 1, 6, 3, 10, 4))
	case "spacecraft":
		mods = append(mods, sceneModule("stand", "solid", 0, 0, 0, 12, 6, 18, 71), sceneModule("hull", "solid", 2, 6, 2, 8, 12, 14, 15), sceneModule("cockpit", "taper", 2, 18, 4, 8, 9, 8, 1), sceneModule("wing-left", "solid", 0, 6, 6, 2, 3, 8, 4), sceneModule("wing-right", "solid", 10, 6, 6, 2, 3, 8, 4))
	case "creature", "figure":
		mods = append(mods, sceneModule("base", "solid", 0, 0, 0, 12, 6, 12, 71), sceneModule("left-foot", "solid", 2, 6, 2, 4, 9, 6, 4), sceneModule("right-foot", "solid", 6, 6, 2, 4, 9, 6, 4), sceneModule("body", "solid", 2, 15, 2, 8, 12, 6, 4), sceneModule("head", "solid", 2, 27, 2, 8, 9, 6, 15), sceneModule("ear-left", "solid", 2, 36, 2, 2, 6, 2, 4), sceneModule("ear-right", "solid", 8, 36, 2, 2, 6, 2, 4), sceneModule("eye-left", "solid", 3, 36, 6, 1, 3, 1, 1), sceneModule("eye-right", "solid", 7, 36, 6, 1, 3, 1, 1))
	}
	return AssemblyRecipe{Version: 4, Title: family, Subject: family, Summary: family, Scene: &SceneSpec{Version: 1, Brief: family, Objects: []SceneObject{{ID: "main", Label: family, Family: family, Modules: mods}}}}
}

func sceneCatalog() PartCatalog {
	c := CatalogCopy(StarterCatalog)
	for _, p := range []PartSpec{
		{ID: "long-4", LDrawID: "3010.dat", StudsX: 4, StudsZ: 1, PlatesY: 3},
		{ID: "long-6", LDrawID: "3009.dat", StudsX: 6, StudsZ: 1, PlatesY: 3},
		{ID: "long-8", LDrawID: "3008.dat", StudsX: 8, StudsZ: 1, PlatesY: 3},
		{ID: "wide-6", LDrawID: "2456.dat", StudsX: 6, StudsZ: 2, PlatesY: 3},
	} {
		c[p.ID] = certifiedRectPart(p)
	}
	return c
}

func TestSceneRequirementColorsRejectUnrelatedSubstitutions(t *testing.T) {
	r := sceneFixture("figure")
	r.Requirements = []string{"白色头部"}
	head := &r.Scene.Objects[0].Modules[4]
	head.Requirements = []int{0}
	head.Color = 1
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "LDraw color 15") {
		t.Fatalf("blue passed as white: %v", err)
	}
	head.Color = 15
	if err := ValidateScene(r); err != nil {
		t.Fatal(err)
	}
	for _, example := range []struct {
		text string
		want []int
	}{
		{"dark blue cockpit and white wings", []int{272, 15}},
		{"深蓝色头盔、白色面板", []int{272, 15}},
		{"dark tan body with blue trim", []int{28, 1}},
		{"a blueprint of a redstone workshop", []int{}},
	} {
		if got := RequiredSceneColors(example.text); !reflect.DeepEqual(got, example.want) {
			t.Fatalf("%q: %v, want %v", example.text, got, example.want)
		}
	}
}

func TestShapeEligibilityRejectsCatalogAxisMismatch(t *testing.T) {
	p := certifiedRectPart(PartSpec{ID: "test", StudsX: 2, StudsZ: 4, PlatesY: 3})
	if !IsShapePartEligible(p) {
		t.Fatal("reviewed matching footprint rejected")
	}
	p.Bounds.MaxX = p.Bounds.MinX + 80
	p.Bounds.MaxZ = p.Bounds.MinZ + 40
	if IsShapePartEligible(p) {
		t.Fatal("physical GLB axes disagreeing with occupancy were accepted")
	}
	p = certifiedRectPart(PartSpec{ID: "test", StudsX: 2, StudsZ: 4, PlatesY: 3})
	p.Bounds.MaxY = 40
	if IsShapePartEligible(p) {
		t.Fatal("physical GLB height disagreeing with connector plane was accepted")
	}
}

func TestSceneFunctionalRequirementsNeedRealConstructions(t *testing.T) {
	r := sceneFixture("figure")
	r.Requirements = []string{"车窗"}
	r.Scene.Objects[0].Modules[3].Requirements = []int{0}
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "real openings") {
		t.Fatalf("label accepted as window: %v", err)
	}
	r.Requirements = []string{"双腿"}
	r.Scene.Objects[0].Modules[3].Requirements = []int{0, 0}
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "duplicate requirement") {
		t.Fatalf("duplicate attribution accepted as paired limbs: %v", err)
	}
	r.Scene.Objects[0].Modules[3].Requirements = []int{0}
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "two attributed") {
		t.Fatalf("single box accepted as two legs: %v", err)
	}
	r.Scene.Objects[0].Modules[1].Requirements = []int{0}
	if err := ValidateScene(r); err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		text            string
		stairs, windows bool
		wheels          int
		valid           bool
	}{
		{"楼梯", false, true, 4, false},
		{"stairs", true, false, 0, true},
		{"four real wheels", false, false, 8, false},
		{"四个真实车轮", false, false, 4, true},
		{"no wheels", false, false, 0, true},
	} {
		if err := checkSceneRequirementFunction(sample.text, sample.stairs, sample.windows, sample.wheels); (err == nil) != sample.valid {
			t.Fatalf("%q: %v", sample.text, err)
		}
	}
}

func TestSceneWindowedFrameKeepsHolesAndCompilesTwoStories(t *testing.T) {
	r := sceneFixture("architecture")
	r.Requirements = []string{"门窗", "楼梯", "二层平台", "屋顶"}
	base := sceneModule("base", "solid", 0, 0, 0, 10, 6, 12, 71)
	room := sceneModule("lower", "frame", 0, 6, 0, 10, 12, 12, 15)
	room.Requirements = []int{0}
	room.Openings = []WallOpening{{Face: "front", X: 4, Y: 0, Width: 2, Height: 9}, {Face: "left", X: 4, Y: 3, Width: 2, Height: 6}, {Face: "right", X: 4, Y: 3, Width: 2, Height: 6}}
	cells, _, err := constructionCells(room)
	if err != nil {
		t.Fatal(err)
	}
	if cells[DesignVector{4, 0, 0}] || cells[DesignVector{0, 3, 4}] || cells[DesignVector{9, 3, 4}] || !cells[DesignVector{4, 9, 0}] || !cells[DesignVector{0, 3, 3}] {
		t.Fatal("face openings destroyed wall piers/lintels or filled holes")
	}
	stairs := sceneModule("stairs", "stairs", 2, 6, 2, 2, 12, 8, 19)
	stairs.Requirements = []int{1}
	floor := sceneModule("floor", "solid", 0, 18, 0, 10, 6, 12, 71)
	floor.Requirements = []int{2}
	upper := room
	upper.ID, upper.Label, upper.Position.Y = "upper", "upper", 24
	roof := sceneModule("roof", "roof", 0, 36, 0, 10, 6, 12, 4)
	roof.Requirements = []int{3}
	r.Scene.Objects[0].Modules = []Construction{base, room, stairs, floor, upper, roof}
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Plan.Validation.Buildable || len(result.Plan.Placements) < 100 {
		t.Fatal("two-story frame lost its real geometry")
	}
	room.Openings = append(room.Openings, WallOpening{Face: "front", X: 5, Y: 0, Width: 2, Height: 6})
	if _, _, err := constructionCells(room); err == nil {
		t.Fatal("overlapping openings erased a pier")
	}
}

func TestSceneFaceMarksRemainDistinct(t *testing.T) {
	face := sceneModule("head", "solid", 0, 0, 0, 6, 12, 6, 25)
	face.Attach = &ConstructionAttachment{Anchor: "ground"}
	modules := []Construction{face}
	for i, id := range []string{"left-eye", "right-eye", "nose"} {
		mark := sceneModule(id, "paint", 0, 0, 0, 2, 3, 1, 0)
		mark.Parent = "head"
		mark.Attach = &ConstructionAttachment{Parent: "head", Anchor: "front", Offset: DesignVector{X: []int{-1, 1, 0}[i]}}
		modules = append(modules, mark)
	}
	scene := SceneSpec{Version: 1, Brief: "separate black eyes and nose", Objects: []SceneObject{{ID: "main", Label: "face", Family: "creature", Modules: modules}}}
	resolved, err := resolveSceneAttachments(context.Background(), scene, StarterCatalog)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range resolved.Objects[0].Modules[1:] {
		if a.Position.Z != 0 {
			t.Fatal("a front facial mark was buried inside the head")
		}
		for _, b := range resolved.Objects[0].Modules[i+2:] {
			// All marks have equal depth. Require a visible stud or plate gap
			// between the projected rectangles, so this is not a single bar.
			if a.Position.X <= b.Position.X+b.Size.X && b.Position.X <= a.Position.X+a.Size.X && a.Position.Y <= b.Position.Y+b.Size.Y && b.Position.Y <= a.Position.Y+a.Size.Y {
				t.Fatalf("distinct facial marks visually merged: %s and %s", a.ID, b.ID)
			}
		}
	}
	scene.Objects[0].Modules[0].Size.Y = 3
	if _, err = resolveSceneAttachments(context.Background(), scene, StarterCatalog); err == nil {
		t.Fatal("crowded face marks were hidden inside the head to force a pass")
	}
}

func TestSceneTypedAttachmentsCannotHideSeparateGroundedLimbs(t *testing.T) {
	base := sceneModule("base", "solid", 0, 0, 0, 8, 6, 8, 71)
	base.Attach = &ConstructionAttachment{Anchor: "ground"}
	body := sceneModule("body", "solid", 0, 0, 0, 4, 9, 4, 25)
	body.Attach = &ConstructionAttachment{Parent: "base", Anchor: "top"}
	leg := sceneModule("leg", "solid", 0, 0, 0, 2, 6, 2, 25)
	leg.Attach = &ConstructionAttachment{Anchor: "ground", Offset: DesignVector{Z: 9}}
	r := AssemblyRecipe{Version: 4, Scene: &SceneSpec{Version: 1, Brief: "connected model", Objects: []SceneObject{{ID: "main", Label: "model", Family: "creature", Modules: []Construction{base, body, leg}}}}}
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "disconnected attachment groups") {
		t.Fatalf("separate leg got past planning: %v", err)
	}
	r.Scene.Objects[0].Modules[2].Attach = &ConstructionAttachment{Parent: "base", Anchor: "top"}
	if err := ValidateScene(r); err != nil {
		t.Fatal(err)
	}
}

func TestSceneFamiliesCompileWithAdvisoryInventory(t *testing.T) {
	for _, family := range []string{"architecture", "vehicle", "spacecraft", "creature", "figure"} {
		t.Run(family, func(t *testing.T) {
			r := sceneFixture(family)
			result, err := CompileScene(context.Background(), r, NewInventorySnapshot(true, 1, nil), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%d parts, %d steps, %d nodes", len(result.Plan.Placements), len(result.Plan.Steps), result.Solver.Visited)
			if !result.Plan.Validation.Buildable || result.Plan.Document.Scene == nil || !result.Plan.InventoryCompared || len(result.Plan.MissingParts) == 0 || result.Plan.Inventory != nil {
				t.Fatal("incomplete scene delivery")
			}
			if result.Plan.Validation.Evidence.Physical != "not_tested" || result.Plan.Validation.Evidence.RetailColors != "not_verified" {
				t.Fatal("unearned physical evidence")
			}
			if strings.Count(result.MPD, "\r\n1 ") != len(result.Plan.Placements) {
				t.Fatal("MPD omitted parts")
			}
		})
	}
}

func TestSceneObjectsCannotHideDisconnectedOrOverlappingParts(t *testing.T) {
	r := sceneFixture("creature")
	other := r.Scene.Objects[0]
	other.ID = "other"
	other.Position.X = 20
	r.Scene.Objects = append(r.Scene.Objects, other)
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.Validation.Evidence.Objects != 2 {
		t.Fatal("object evidence missing")
	}
	r.Scene.Objects[1].Position.X = 0
	if _, err = CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0)); err == nil {
		t.Fatal("cross-object collision accepted")
	}
	r = sceneFixture("creature")
	r.Scene.Objects[0].Modules = append(r.Scene.Objects[0].Modules, sceneModule("loose", "solid", 20, 0, 0, 2, 3, 2, 1))
	if _, err = CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0)); err == nil {
		t.Fatal("disconnected feature accepted as another object")
	}
}

func TestSceneInputsEvidenceAndCancellation(t *testing.T) {
	r := sceneFixture("creature")
	r.Requirements = []string{"two eyes"}
	if err := ValidateScene(r); err == nil {
		t.Fatal("unrepresented requirement accepted")
	}
	r.Scene.Objects[0].Modules[len(r.Scene.Objects[0].Modules)-1].Requirements = []int{0}
	if err := ValidateScene(r); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(r)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := CompileScene(ctx, r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0)); err == nil || len(result.Plan.Placements) > 0 {
		t.Fatal("cancelled search delivered parts")
	}
	after, _ := json.Marshal(r)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("compilation mutated input")
	}
}

func TestSceneComplexCompositeAndShapeColor(t *testing.T) {
	r := sceneFixture("architecture")
	for i, family := range []string{"vehicle", "spacecraft", "creature", "figure"} {
		o := sceneFixture(family).Scene.Objects[0]
		o.ID = family
		o.Position = []DesignVector{{20, 0, 0}, {0, 0, 20}, {20, 0, 20}, {40, 0, 20}}[i]
		r.Scene.Objects = append(r.Scene.Objects, o)
	}
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Plan.Placements) < 500 || result.Plan.Validation.Evidence.Objects != 5 {
		t.Fatal("complex composite fell back to a small model")
	}
	t.Logf("complex composite: %d parts, %d modules, %d steps", len(result.Plan.Placements), result.Plan.Validation.Evidence.Modules, len(result.Plan.Steps))
	if output := os.Getenv("CHIMII_SCENE_FIXTURE_OUTPUT"); output != "" {
		if err := os.MkdirAll(output, 0700); err != nil {
			t.Fatal(err)
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(filepath.Join(output, "complex-scene.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r = sceneFixture("creature")
	r.Scene.Objects[0].Modules = []Construction{{ID: "local", Label: "colored local design", Kind: "shape", Color: 71, Design: &DesignSpec{Version: 1, Mode: "static", Shapes: []ShapeNode{
		{ID: "body", Label: "body", Kind: "box", Operation: "add", Size: DesignVector{4, 3, 4}, Color: 4},
		{ID: "eyes", Label: "eyes", Kind: "box", Operation: "add", Position: DesignVector{1, 3, 1}, Size: DesignVector{2, 3, 2}, Color: 15},
	}}}}
	result, err = CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	colors := map[int]bool{}
	for _, p := range result.Plan.Placements {
		colors[p.Color] = true
	}
	if !colors[4] || !colors[15] || colors[71] {
		t.Fatal("local shape lost feature colors")
	}
}

func TestSceneColoredBricksCannotFillHolesOrEraseFeatures(t *testing.T) {
	r := sceneFixture("figure")
	paint := sceneModule("panel", "paint", 3, 30, 2, 4, 3, 1, 1)
	paint.Parent = "head"
	r.Scene.Objects[0].Modules = append(r.Scene.Objects[0].Modules, paint)
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range result.Plan.Placements {
		found = found || p.Module == "main/panel"
	}
	if !found {
		t.Fatal("colored brick feature was omitted")
	}
	r.Scene.Objects[0].Modules[len(r.Scene.Objects[0].Modules)-1].Position.Z = 12
	if _, _, err := sceneObjectTarget(r.Scene.Objects[0], StarterCatalog); err == nil {
		t.Fatal("paint added unsupported volume")
	}
	r = sceneFixture("architecture")
	paint = sceneModule("fill-door", "paint", 6, 6, 0, 4, 9, 2, 15)
	paint.Parent = "front"
	r.Scene.Objects[0].Modules = append(r.Scene.Objects[0].Modules, paint)
	if _, _, err := sceneObjectTarget(r.Scene.Objects[0], sceneCatalog()); err == nil {
		t.Fatal("paint filled a protected hole")
	}
}

func TestSceneAttachmentsComputePositionsWithoutInventingConnections(t *testing.T) {
	r := sceneFixture("vehicle")
	mods := r.Scene.Objects[0].Modules
	for i := 1; i < len(mods); i++ {
		mods[i].Attach = &ConstructionAttachment{Parent: mods[i-1].ID, Anchor: "top"}
		mods[i].Position = DesignVector{}
	}
	before, _ := json.Marshal(r)
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	resolved := result.Plan.Document.Scene.Objects[0].Modules
	if resolved[1].Position.Y != 5 || resolved[2].Position != (DesignVector{1, 11, 1}) || resolved[3].Position != (DesignVector{1, 20, 1}) {
		t.Fatalf("incorrect relative anchors: %+v", resolved)
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("attachment resolution mutated source")
	}
	r.Scene.Objects[0].Modules[1].Attach.Parent = "roof"
	if _, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0)); err == nil {
		t.Fatal("cyclic anchors accepted")
	}
	r = sceneFixture("figure")
	paint := sceneModule("face", "paint", 0, 0, 0, 2, 3, 1, 1)
	paint.Attach = &ConstructionAttachment{Parent: "head", Anchor: "front"}
	r.Scene.Objects[0].Modules = append(r.Scene.Objects[0].Modules, paint)
	result, err = CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	face := result.Plan.Document.Scene.Objects[0].Modules[len(r.Scene.Objects[0].Modules)-1]
	if face.Position != (DesignVector{5, 30, 2}) || face.Parent != "head" {
		t.Fatalf("front paint misplaced: %+v", face)
	}
	r.Scene.Objects[0].Modules[len(r.Scene.Objects[0].Modules)-1].Kind = "solid"
	if _, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0)); err == nil {
		t.Fatal("invented side connectors accepted")
	}
}

func TestSceneGeneratedSizingPreservesFeaturesAndSource(t *testing.T) {
	r := sceneFixture("creature")
	r.Scene.Objects[0].Modules = []Construction{sceneModule("ear", "taper", 0, 0, 0, 2, 6, 2, 4), sceneModule("wall", "wall", 4, 0, 0, 8, 6, 2, 15)}
	r.Scene.Objects[0].Modules[1].Openings = []WallOpening{{X: 1, Y: 0, Width: 4, Height: 6}}
	before, _ := json.Marshal(r)
	fitted := FitGeneratedScene(r)
	if fitted.Scene.Objects[0].Modules[0].Size != (DesignVector{4, 6, 4}) || fitted.Scene.Objects[0].Modules[1].Size.Y != 9 || fitted.Scene.Objects[0].Modules[1].Openings[0] != (WallOpening{X: 2, Y: 0, Width: 4, Height: 6}) {
		t.Fatalf("bad parameter fitting: %+v", fitted.Scene)
	}
	if fitted.Scene.Objects[0].Modules[0].Color != 4 || len(fitted.Scene.Objects[0].Modules[1].Openings) != 1 || fitted.Metadata["construction_sizing"] == "" {
		t.Fatal("fitting hid or recolored features")
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("sizing mutated the model's source")
	}
	if err := ValidateScene(r); err == nil {
		t.Fatal("authoritative compiler silently fitted invalid API geometry")
	}
	if err := ValidateScene(fitted); err != nil {
		t.Fatal(err)
	}
}

func TestSceneSharedAnchorsAndActualCurvedSurface(t *testing.T) {
	r := sceneFixture("figure")
	modules := []Construction{sceneModule("base", "solid", 0, 0, 0, 16, 6, 12, 71), sceneModule("left", "solid", 0, 0, 0, 4, 9, 8, 4), sceneModule("right", "solid", 0, 0, 0, 4, 9, 8, 4), sceneModule("torso", "solid", 0, 0, 0, 10, 9, 8, 15), sceneModule("head", "ellipse", 0, 0, 0, 8, 6, 8, 15), sceneModule("visor", "paint", 0, 0, 0, 4, 3, 1, 1)}
	modules[0].Attach = &ConstructionAttachment{Anchor: "ground"}
	modules[1].Attach = &ConstructionAttachment{Parent: "base", Anchor: "top", Offset: DesignVector{X: -3}}
	modules[2].Attach = &ConstructionAttachment{Parent: "base", Anchor: "top", Offset: DesignVector{X: 3}}
	modules[3].Attach = &ConstructionAttachment{Parents: []string{"left", "right"}, Anchor: "top"}
	modules[4].Attach = &ConstructionAttachment{Parent: "torso", Anchor: "top"}
	modules[5].Attach = &ConstructionAttachment{Parent: "head", Anchor: "front"}
	r.Scene.Objects[0].Modules = modules
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	got := result.Plan.Document.Scene.Objects[0].Modules
	if got[3].Position.Y != 15 || got[4].Position.Y != 24 || got[5].Position.Y < 24 || got[5].Parent != "head" {
		t.Fatalf("bad joint/curved anchors: %+v", got)
	}
	if len(result.Plan.Validation.Issues) > 0 {
		t.Fatal("anchors bypassed mechanics")
	}
	r.Scene.Objects[0].Modules[3].Attach.Parents = []string{"left", "left"}
	if err := ValidateScene(r); err == nil {
		t.Fatal("duplicate joint parent accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolveSceneAttachments(ctx, *r.Scene, sceneCatalog()); !errors.Is(err, context.Canceled) {
		t.Fatalf("resolver ignored cancellation: %v", err)
	}
}
