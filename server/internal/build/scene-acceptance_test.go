package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This authored fixture checks compiler/render/workflow integration. It is not
// evidence that a generative model designed the figure or reviewed its images.
func TestSceneV2ComplexAcceptanceFixture(t *testing.T) {
	r := sceneFixture("figure")
	r.Scene.Version = 2
	r.Title = "积木角色"
	r.Prompt = "有双腿、头部、尖耳和眼睛的积木角色"
	r.Scene.Objects[0].Interfaces = []AssemblyInterface{
		{ID: "left-hip", A: "left-foot", B: "body", MinimumConnections: 1},
		{ID: "right-hip", A: "right-foot", B: "body", MinimumConnections: 1},
		{ID: "neck", A: "body", B: "head", MinimumConnections: 2},
		{ID: "left-ear", A: "head", B: "ear-left", MinimumConnections: 1},
		{ID: "right-ear", A: "head", B: "ear-right", MinimumConnections: 1},
	}
	for i := range r.Scene.Objects[0].Modules {
		m := &r.Scene.Objects[0].Modules[i]
		m.Role = "subject"
		if m.ID == "base" {
			m.Role = "display_support"
		}
		if m.ID == "eye-left" || m.ID == "eye-right" {
			m.Role = "detail"
		}
	}
	started := time.Now()
	result, err := CompileScene(context.Background(), r, UnlimitedInventory(), CatalogVersion, StarterCatalog, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Plan.Placements) < 80 || len(result.Plan.Validation.Evidence.Interfaces) != 5 {
		t.Fatal("lost complex assembly/interface evidence")
	}
	views, err := RenderSceneViews(context.Background(), result.Plan, reviewTestAssets(t), map[int]uint32{71: 0x969696, 15: 0xf4f4f4, 19: 0xd7ba8c, 4: 0xb40000, 1: 0x1e5aa8})
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.Validation.Evidence.Physical != "not_tested" || result.Plan.Validation.Evidence.LoadEstimate.Status != "not_evaluated" {
		t.Fatal("large fixture claimed a load or physical result")
	}
	t.Logf("authored fixture: %d parts, %d steps, %d stages, %s", len(result.Plan.Placements), len(result.Plan.Steps), result.Solver.CompileStages, time.Since(started))
	if output := os.Getenv("CHIMII_SCENE_ACCEPTANCE_OUTPUT"); output != "" {
		if err := os.MkdirAll(output, 0700); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(output, "authored-figure.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		for _, view := range views {
			if err = os.WriteFile(filepath.Join(output, view.Direction+".png"), view.PNG, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
