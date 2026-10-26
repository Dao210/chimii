package build

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestPlatformHasBoundedBearingGapsAndEmptyInterior(t *testing.T) {
	for length := 4; length <= 32; length++ {
		piers := platformPiers(length)
		if piers[0] != 0 || piers[len(piers)-1] != length-2 {
			t.Fatalf("%d: missing edge bearings: %v", length, piers)
		}
		for i := 1; i < len(piers); i++ {
			if piers[i] <= piers[i-1] || piers[i]-piers[i-1] > 6 {
				t.Fatalf("%d: unsupported gap or duplicate: %v", length, piers)
			}
		}
	}
	m := sceneModule("deck", "platform", 0, 0, 0, 14, 18, 14, 71)
	cells, fixed, err := constructionCells(m)
	if err != nil || len(fixed) != 0 {
		t.Fatalf("platform: %v", err)
	}
	if cells[DesignVector{3, 6, 3}] || cells[DesignVector{3, 12, 3}] || !cells[DesignVector{0, 11, 0}] || !cells[DesignVector{3, 12, 0}] || !cells[DesignVector{3, 15, 3}] || !cells[DesignVector{13, 17, 13}] {
		t.Fatal("platform pillars/cap do not retain the declared open volume")
	}
	m.Size.Y = 8
	if _, _, err := constructionCells(m); err == nil {
		t.Fatal("a short unsupported deck passed as a platform")
	}
}

func TestPlatformSupportsArchitectureAndSpacecraftPayloads(t *testing.T) {
	for _, family := range []string{"architecture", "spacecraft", "figure"} {
		t.Run(family, func(t *testing.T) {
			mods := []Construction{
				sceneModule("base", "solid", 0, 0, 0, 14, 6, 18, 71),
				sceneModule("deck", "platform", 0, 0, 0, 14, 15, 18, 72),
				sceneModule("payload", "solid", 0, 0, 0, 6, 6, 16, 15),
				sceneModule("left", "solid", 0, 0, 0, 2, 6, 8, 4),
				sceneModule("right", "solid", 0, 0, 0, 2, 6, 8, 4),
			}
			mods[0].Attach = &ConstructionAttachment{Anchor: "ground"}
			for i := 1; i < len(mods); i++ {
				parent := "deck"
				if i == 1 {
					parent = "base"
				}
				mods[i].Attach = &ConstructionAttachment{Parent: parent, Anchor: "top"}
			}
			mods[3].Attach.Offset.X, mods[4].Attach.Offset.X = -5, 5
			r := AssemblyRecipe{Version: 4, Title: family, Subject: family, Summary: family, Scene: &SceneSpec{Version: 1, Brief: family, Objects: []SceneObject{{ID: "main", Label: family, Family: family, Modules: mods}}}}
			original := *r.Scene
			out, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
			if err != nil || !out.Plan.Validation.Buildable {
				t.Fatalf("supported %s: %v", family, err)
			}
			if !reflect.DeepEqual(original, *r.Scene) {
				t.Fatal("compilation modified source attachments")
			}
			if out.Recipe.Scene.Objects[0].Modules[2].Position.Y != 21 {
				t.Fatal("payload was not placed above the complete support assembly")
			}
		})
	}
}

func TestPlatformCannotLandItsPierGridOnANarrowPedestal(t *testing.T) {
	base := sceneModule("base", "solid", 0, 0, 0, 4, 6, 4, 71)
	base.Attach = &ConstructionAttachment{Anchor: "ground"}
	deck := sceneModule("deck", "platform", 0, 0, 0, 14, 15, 18, 72)
	deck.Attach = &ConstructionAttachment{Parent: "base", Anchor: "top"}
	scene := SceneSpec{Version: 1, Brief: "undersized bearing", Objects: []SceneObject{{ID: "main", Label: "stand", Family: "spacecraft", Modules: []Construction{base, deck}}}}
	if _, err := resolveSceneAttachments(context.Background(), scene, sceneCatalog()); err == nil {
		t.Fatal("one corner contact replaced the platform's distributed bearing grid")
	}
}

func TestPaintedWindowSurroundKeepsRealHole(t *testing.T) {
	r := sceneFixture("vehicle")
	cabin := &r.Scene.Objects[0].Modules[2]
	cabin.Openings = []WallOpening{{Face: "front", X: 2, Y: 3, Width: 2, Height: 3}}
	trim := sceneModule("window-trim", "paint", 0, 0, 0, 6, 9, 2, 1)
	trim.Attach = &ConstructionAttachment{Parent: "cabin", Anchor: "front"}
	trim.Openings = []WallOpening{{X: 2, Y: 3, Width: 2, Height: 3}}
	r.Scene.Objects[0].Modules = append(r.Scene.Objects[0].Modules, trim)
	resolved, err := resolveSceneAttachments(context.Background(), *r.Scene, sceneCatalog())
	if err != nil {
		t.Fatal(err)
	}
	target, _, err := sceneObjectTarget(resolved.Objects[0], sceneCatalog())
	if err != nil {
		t.Fatal(err)
	}
	if _, filled := target[DesignVector{3, 14, 1}]; filled {
		t.Fatal("window surround filled the original window")
	}
	if target[DesignVector{1, 14, 1}].Color != 1 {
		t.Fatal("surround did not use real bricks in its chosen color")
	}
	out, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil || !out.Plan.Validation.Buildable {
		t.Fatalf("window surround did not compile: %v", err)
	}
	r.Scene.Objects[0].Modules[len(r.Scene.Objects[0].Modules)-1].Openings = nil
	if _, err = CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0)); err == nil {
		t.Fatal("solid painted rectangle was allowed to fill the window")
	}
}
