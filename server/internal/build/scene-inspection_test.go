package build

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSceneInspectionExposesBaseOnlyLimbConnections(t *testing.T) {
	base := sceneModule("base", "solid", 0, 0, 0, 10, 6, 6, 71)
	body := sceneModule("body", "solid", 4, 6, 2, 2, 9, 2, 15)
	head := sceneModule("head", "solid", 4, 15, 2, 2, 3, 2, 15)
	left := sceneModule("left-arm", "solid", 1, 6, 2, 2, 9, 2, 15)
	right := sceneModule("right-arm", "solid", 7, 6, 2, 2, 9, 2, 15)
	r := AssemblyRecipe{Version: 4, Title: "Figure", Scene: &SceneSpec{Version: 1, Brief: "Arms rely on a display base", Objects: []SceneObject{{ID: "main", Label: "Figure", Family: "figure", Rotation: 90, Modules: []Construction{base, body, head, left, right}}}}}
	compiled, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, sceneCatalog(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	inspection := SceneInspection(compiled.Plan)
	if len(inspection) != 1 || inspection[0].OccupancySize != (DesignVector{6, 18, 10}) {
		t.Fatalf("compiled rotated dimensions were not preserved: %+v", inspection)
	}
	parts := 0
	for _, group := range inspection[0].Modules {
		parts += group.Parts
		if group.Module == "main/base" && group.ComponentsAfterRemoving != 3 {
			t.Fatalf("base-only connections hidden: %+v", group)
		}
		if group.Module == "main/left-arm" || group.Module == "main/right-arm" {
			if len(group.ConnectionsTo) != 1 || group.ConnectionsTo["main/base"] == 0 {
				t.Fatalf("arm falsely attributed to torso: %+v", group)
			}
		}
	}
	if parts != len(compiled.Plan.Placements) {
		t.Fatal("inspection omitted compiled parts")
	}
	a, _ := json.Marshal(inspection)
	b, _ := json.Marshal(SceneInspection(compiled.Plan))
	if string(a) != string(b) {
		t.Fatal("inspection depends on map iteration order")
	}
}
