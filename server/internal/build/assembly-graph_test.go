package build

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAssemblyGraphRejectsPedestalOnlySubjects(t *testing.T) {
	r := sceneFixture("spacecraft")
	r.Scene.Version = 2
	for i := range r.Scene.Objects[0].Modules {
		r.Scene.Objects[0].Modules[i].Role = "subject"
	}
	r.Scene.Objects[0].Modules[0].Role = "display_support"
	if err := ValidateScene(r); err == nil || !strings.Contains(err.Error(), "interface graph") {
		t.Fatalf("unconnected subject accepted: %v", err)
	}
}

func TestAssemblyInterfaceUsesRealConnectionsAndBridgingParts(t *testing.T) {
	cat := CatalogCopy(StarterCatalog)
	scene := SceneSpec{Version: 1, Brief: "Interface evidence", Objects: []SceneObject{{ID: "main", Label: "Model", Family: "figure", Modules: []Construction{
		sceneModule("base", "solid", 0, 0, 0, 4, 3, 2, 4), sceneModule("left", "solid", 0, 3, 0, 2, 3, 2, 15), sceneModule("right", "solid", 2, 3, 0, 2, 3, 2, 15),
	}, Interfaces: []AssemblyInterface{{ID: "cross-bond", A: "left", B: "right", MinimumConnections: 1}}}}}
	plan := BuildPlan{Placements: []Placement{
		{ID: "base", PartID: "brick-2x4", Color: 4, Step: 1, Object: "main", Module: "main/base"},
		{ID: "bridge", PartID: "brick-2x4", Color: 15, Y: 3, Step: 2, Object: "main", Module: "main/left"},
	}}
	plan.Connections = deriveExactConnections(plan.Placements, cat)
	evidence, err := inspectAssemblyInterfaces(plan, scene, cat)
	if err != nil || len(evidence) != 1 || len(evidence[0].SharedParts) != 1 {
		t.Fatalf("real shared part lost: %+v %v", evidence, err)
	}
	scene.Objects[0].Modules[1].Size.Y = 6
	scene.Objects[0].Interfaces[0].MinimumConnections = 2
	plan.Placements = append(plan.Placements, Placement{ID: "left-top", PartID: "brick-2x2", Color: 15, Y: 6, Step: 3, Object: "main", Module: "main/left"})
	plan.Connections = deriveExactConnections(plan.Placements, cat)
	if _, err = inspectAssemblyInterfaces(plan, scene, cat); err == nil {
		t.Fatal("one shared brick and its internal contact counted as two bridges")
	}
	scene.Objects[0].Modules[1].Size.Y = 3
	scene.Objects[0].Interfaces[0].MinimumConnections = 1
	plan.Placements = []Placement{
		{ID: "base", PartID: "brick-2x4", Color: 4, Step: 1, Object: "main", Module: "main/base"},
		{ID: "left", PartID: "brick-2x2", Color: 15, Y: 3, Step: 2, Object: "main", Module: "main/left"},
		{ID: "right", PartID: "brick-2x2", Color: 15, X: 2, Y: 3, Step: 2, Object: "main", Module: "main/right"},
	}
	plan.Connections = deriveExactConnections(plan.Placements, cat)
	if _, err = inspectAssemblyInterfaces(plan, scene, cat); err == nil {
		t.Fatal("two columns on common base passed as a direct subject interface")
	}
}

func TestAssemblyGraphCompilesAndChecksContracts(t *testing.T) {
	r := AssemblyRecipe{Version: 4, Title: "Tower", Subject: "tower", Summary: "Two connected sections", Scene: &SceneSpec{Version: 2, Brief: "A bonded tower", Objects: []SceneObject{{ID: "main", Label: "Tower", Family: "architecture", Modules: []Construction{
		{ID: "lower", Label: "Lower", Kind: "solid", Role: "subject", Size: DesignVector{4, 6, 4}, Color: 4, Attach: &ConstructionAttachment{Anchor: "ground"}},
		{ID: "upper", Label: "Upper", Kind: "solid", Role: "subject", Size: DesignVector{4, 6, 4}, Color: 15, Attach: &ConstructionAttachment{Parent: "lower", Anchor: "top"}},
	}, Interfaces: []AssemblyInterface{{ID: "joint", A: "lower", B: "upper", MinimumConnections: 1}}}}}}
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil || len(result.Plan.Validation.Evidence.Interfaces) != 1 {
		t.Fatalf("valid interface rejected: %v", err)
	}
	r.Scene.Objects[0].Interfaces[0].B = "missing"
	if err = ValidateScene(r); err == nil {
		t.Fatal("unknown interface endpoint accepted")
	}
}
