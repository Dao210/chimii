package handler

import (
	"context"
	"os"
	"testing"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

// Explicit provider evaluation, never an ambient agent executable. Normal tests
// neither read provider configuration nor contact a model.
func TestBuildConversationLivePlanner(t *testing.T) {
	if os.Getenv("CHIMII_CONVERSATION_LIVE") != "1" {
		t.Skip("set CHIMII_CONVERSATION_LIVE=1 with an explicitly configured utility model")
	}
	h := &Handler{LLM: llm.New(llm.Config{APIKey: os.Getenv("CHIMII_LLM_API_KEY"), BaseURL: os.Getenv("CHIMII_LLM_BASE_URL"), DefaultModel: os.Getenv("CHIMII_LLM_DEFAULT_MODEL")})}
	if !h.LLM.Enabled() {
		t.Fatal("utility model is not configured")
	}
	w := NewBuildWorker(h)
	t.Run("route_radio", func(t *testing.T) {
		d, err := w.routeBuild(context.Background(), db.BuildSession{Prompt: "我想搭一个可以听广播的收音机"}, map[string]string{}, nil)
		if err != nil {
			t.Fatalf("provider route failed (%T)", err)
		}
		if d.Kind != "circuit" {
			t.Fatalf("route: %s", d.Kind)
		}
	})
	t.Run("hybrid_requires_choice", func(t *testing.T) {
		d, err := w.routeBuild(context.Background(), db.BuildSession{Prompt: "做一只兔子形状的乐高收音机，外壳和能播放的电路都要"}, map[string]string{}, nil)
		if err != nil {
			t.Fatalf("provider route failed (%T)", err)
		}
		if d.Kind != "clarify" {
			t.Fatalf("hybrid was silently routed to %s", d.Kind)
		}
	})
	var boson circuit.Catalog
	for _, c := range circuit.Catalogs() {
		if c.ConnectionSystem == "boson" {
			boson = c
		}
	}
	t.Run("boson_button_light", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		d, err := circuit.PlanConversation(ctx, h.LLM, boson, map[string]any{"idea": "用按钮控制灯，按下亮，松开就灭", "questions_remaining": 2})
		if err != nil {
			t.Fatalf("provider circuit plan failed (%T)", err)
		}
		if d.Outcome != "ready" || d.ProjectID != "boson-button-light" {
			t.Fatalf("outcome %s project %s", d.Outcome, d.ProjectID)
		}
	})
	t.Run("boson_radio_unsupported", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		d, err := circuit.PlanConversation(ctx, h.LLM, boson, map[string]any{"idea": "用这套 BOSON 做能收听 FM 广播的收音机", "questions_remaining": 2})
		if err != nil {
			t.Fatalf("provider circuit plan failed (%T)", err)
		}
		if d.Outcome != "unsupported" {
			t.Fatalf("unsupported request became %s", d.Outcome)
		}
	})
	t.Run("brick_edit_retains_doorway", func(t *testing.T) {
		draft := &buildstudio.AssemblyRecipe{Version: 3, Subject: "gate", Title: "有门洞的大门", Summary: "一扇有门洞的大门", Requirements: []string{"保留贯通的门洞"}, Metadata: map[string]string{}, Modules: []buildstudio.ModuleInstance{}, Design: &buildstudio.DesignSpec{Version: 1, Mode: "static", Shapes: []buildstudio.ShapeNode{{ID: "wall", Label: "门墙", Kind: "box", Operation: "add", Size: buildstudio.DesignVector{X: 8, Y: 12, Z: 4}, Color: 1}, {ID: "doorway", Label: "门洞", Kind: "box", Operation: "subtract", Position: buildstudio.DesignVector{X: 3}, Size: buildstudio.DesignVector{X: 2, Y: 6, Z: 4}, Color: 1}}}}
		decision, err := h.planBuildRecipe(context.Background(), "让大门矮一点，保留门洞，其他不要改", map[string]string{}, draft, 1, buildstudio.UnlimitedInventory(), nil)
		if err != nil {
			t.Fatalf("provider brick edit failed (%T)", err)
		}
		if decision.Outcome != "ready" || decision.Recipe == nil || decision.Recipe.Design == nil {
			t.Fatalf("edit outcome %s", decision.Outcome)
		}
		doorway, lower := false, false
		for _, s := range decision.Recipe.Design.Shapes {
			if s.ID == "doorway" && s.Operation == "subtract" {
				doorway = true
			}
			if s.ID == "wall" && s.Size.Y < 12 {
				lower = true
			}
		}
		if !doorway || !lower {
			t.Fatalf("edit failed to preserve doorway or lower wall: doorway=%v lower=%v", doorway, lower)
		}
	})
}
