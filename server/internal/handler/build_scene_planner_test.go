package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

func scenePlanningFixture() buildstudio.AssemblyRecipe {
	recipe := buildstudio.AssemblyRecipe{Version: 4, Subject: "figure", Title: "Standing figure", Summary: "Standing figure with a colored head", Prompt: "a standing figure", Requirements: []string{"colored head"}, Scene: &buildstudio.SceneSpec{Version: 1, Brief: "Stable standing figure on a base", Objects: []buildstudio.SceneObject{{ID: "main", Label: "Figure", Family: "figure", Modules: []buildstudio.Construction{
		{ID: "base", Label: "Base", Kind: "solid", Attach: &buildstudio.ConstructionAttachment{Anchor: "ground"}, Size: buildstudio.DesignVector{X: 8, Y: 6, Z: 8}, Color: 71},
		{ID: "body", Label: "Body", Kind: "solid", Attach: &buildstudio.ConstructionAttachment{Parent: "base", Anchor: "top"}, Position: buildstudio.DesignVector{X: 2, Y: 6, Z: 2}, Size: buildstudio.DesignVector{X: 4, Y: 9, Z: 4}, Color: 4},
		{ID: "head", Label: "Head", Kind: "solid", Attach: &buildstudio.ConstructionAttachment{Parent: "body", Anchor: "top"}, Position: buildstudio.DesignVector{X: 2, Y: 15, Z: 2}, Size: buildstudio.DesignVector{X: 4, Y: 3, Z: 4}, Color: 15, Requirements: []int{0}},
	}}}}}
	recipe.Scene.Version = 2
	for i := range recipe.Scene.Objects[0].Modules {
		recipe.Scene.Objects[0].Modules[i].Role = "subject"
	}
	recipe.Scene.Objects[0].Modules[0].Role = "display_support"
	recipe.Scene.Objects[0].Interfaces = []buildstudio.AssemblyInterface{{ID: "body-head", A: "body", B: "head", MinimumConnections: 1}}
	return recipe

}

func sceneTestLLM(reply func(*http.Request) (string, error)) *llm.Client {
	return llm.New(llm.Config{BaseURL: "https://scene-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, err := reply(r)
		if err != nil {
			return nil, err
		}
		data := mustBuildJSON(map[string]any{"content": []map[string]string{{"type": "text", "text": raw}}})
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(data))), Request: r}, nil
	})}})
}

func TestBuildScenePlannerRepairsMalformedOutputOnce(t *testing.T) {
	calls := 0
	h := &Handler{LLM: sceneTestLLM(func(r *http.Request) (string, error) {
		calls++
		data, _ := io.ReadAll(r.Body)
		var controls struct {
			Thinking struct {
				Type   string `json:"type"`
				Budget int    `json:"budget_tokens"`
			} `json:"thinking"`
			Output struct {
				Effort string `json:"effort"`
			} `json:"output_config"`
		}
		if err := json.Unmarshal(data, &controls); err != nil || controls.Thinking.Type != "enabled" || controls.Thinking.Budget != 2048 || controls.Output.Effort != "low" {
			t.Fatalf("planning reasoning not bounded: %v", err)
		}
		if calls == 1 {
			return `{"outcome":"ready","recipe":`, nil
		}
		if !strings.Contains(string(data), "validation_error") {
			t.Fatal("repair omitted diagnostics")
		}
		recipe := scenePlanningFixture()
		return string(mustBuildJSON(buildPlanningDecision{Outcome: "ready", Recipe: &recipe})), nil
	})}
	decision, err := h.planBuildRecipe(context.Background(), "a standing figure", nil, nil, 1, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog)
	if err != nil || decision.Recipe.Scene == nil || calls != 2 {
		t.Fatalf("bounded format repair: %d %v", calls, err)
	}
	if decision.Recipe.Metadata["target_part_count"] != "" || decision.Recipe.Constraints.PartCount != 0 {
		t.Fatal("automatic scene scale inherited the legacy default count")
	}
	h.LLM = sceneTestLLM(func(*http.Request) (string, error) { calls++; return "invalid", nil })
	calls = 0
	_, err = h.planBuildRecipe(context.Background(), "a figure", nil, nil, 1, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog)
	if code, _ := buildstudio.BuildErrorCode(err); code != buildstudio.BuildErrorRequirements || calls != 2 {
		t.Fatalf("invalid output kept retrying: %d %v", calls, err)
	}
}

func TestBuildScenePlannerFitsCorrectedOutputAndRequiresAnchors(t *testing.T) {
	calls := 0
	h := &Handler{LLM: sceneTestLLM(func(r *http.Request) (string, error) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "color_names") {
			t.Fatal("planner omitted color name mapping")
		}
		recipe := scenePlanningFixture()
		if calls == 1 {
			recipe.Scene.Objects[0].Modules[2].Attach = nil
		} else {
			recipe.Scene.Objects[0].Modules[2].Kind = "taper"
			recipe.Scene.Objects[0].Modules[2].Size = buildstudio.DesignVector{X: 2, Y: 6, Z: 2}
		}
		return string(mustBuildJSON(buildPlanningDecision{Outcome: "ready", Recipe: &recipe})), nil
	})}
	decision, err := h.planBuildRecipe(context.Background(), "a standing figure", nil, nil, 1, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog)
	if err != nil || calls != 2 || decision.Recipe.Scene.Objects[0].Modules[2].Size.X != 4 || decision.Recipe.Metadata["construction_sizing"] == "" {
		t.Fatalf("corrected output bypassed sizing/anchors: calls=%d err=%v", calls, err)
	}
}

func TestBuildSceneRevisionCanRestoreExplicitRequiredColor(t *testing.T) {
	original := scenePlanningFixture()
	original.Requirements = []string{"white head"}
	original.Scene.Objects[0].Modules[2].Color = 1
	decision, err := parseSceneRevision(`{"outcome":"ready","message":"restore white head","updates":[{"object":"main","module":"head","color":15}]}`, original)
	if err != nil || decision.Recipe.Scene.Objects[0].Modules[2].Color != 15 || original.Scene.Objects[0].Modules[2].Color != 1 {
		t.Fatalf("required color repair: %v", err)
	}
}

func TestBuildScenePlannerRequiresV2ForNewIdeasAndKeepsLegacyDraft(t *testing.T) {
	legacy := scenePlanningFixture()
	legacy.Scene.Version = 1
	calls := 0
	h := &Handler{LLM: sceneTestLLM(func(*http.Request) (string, error) {
		calls++
		return string(mustBuildJSON(buildPlanningDecision{Outcome: "ready", Recipe: &legacy})), nil
	})}
	if _, err := h.planBuildRecipe(context.Background(), "a figure", nil, nil, 1, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog); err == nil || calls != 2 {
		t.Fatalf("new idea accepted legacy scene: calls=%d err=%v", calls, err)
	}
	decision, err := h.planBuildRecipe(context.Background(), "edit this figure", nil, &legacy, 1, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog)
	if err != nil || decision.Recipe == nil || decision.Recipe.Scene.Version != 1 {
		t.Fatalf("legacy edit lost compatibility: %v", err)
	}
}

func TestBuildSceneRevisionKeepsCeilingAccess(t *testing.T) {
	original := scenePlanningFixture()
	module := &original.Scene.Objects[0].Modules[2]
	module.Kind, module.Size = "storey", buildstudio.DesignVector{X: 10, Y: 18, Z: 14}
	module.CeilingOpenings = []buildstudio.DeckOpening{{X: 2, Z: 3, Width: 2, Depth: 8}}
	if _, err := parseSceneRevision(`{"outcome":"ready","message":"hide hole","updates":[{"object":"main","module":"head","ceiling_openings":[]}]}`, original); err == nil {
		t.Fatal("repair erased a stair/access opening")
	}
	decision, err := parseSceneRevision(`{"outcome":"ready","message":"move opening clear of a pier","updates":[{"object":"main","module":"head","ceiling_openings":[{"x":3,"z":3,"width":2,"depth":8}]}]}`, original)
	if err != nil || decision.Recipe.Scene.Objects[0].Modules[2].CeilingOpenings[0].X != 3 || original.Scene.Objects[0].Modules[2].CeilingOpenings[0].X != 2 {
		t.Fatalf("immutable access patch: %v", err)
	}
}

func TestBuildSceneReviewPreservesFeaturesAndCheckpoints(t *testing.T) {
	original := scenePlanningFixture()
	for _, raw := range []string{
		`{"outcome":"ready","message":"repair","requirements":[],"updates":[{"object":"main","module":"head","size":{"x":4,"y":3,"z":4}}]}`,
		`{"outcome":"ready","message":"repair","updates":[{"object":"main","module":"head","color":1}]}`,
		`{"outcome":"ready","message":"repair","updates":[{"object":"main","module":"head","kind":"wheelbase"}]}`,
		`{"outcome":"ready","message":"repair","updates":[{"object":"main","module":"missing","size":{"x":4,"y":3,"z":4}}]}`,
	} {
		if _, err := parseSceneRevision(raw, original); err == nil {
			t.Fatal("repair weakened a requested feature")
		}
	}
	broken := scenePlanningFixture()
	broken.Scene.Objects[0].Modules[2].Attach = nil
	broken.Scene.Objects[0].Modules[2].Position.Y = 18
	calls, checkpoints := 0, 0
	h := &Handler{LLM: sceneTestLLM(func(r *http.Request) (string, error) {
		calls++
		if calls == 1 {
			data, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(data), "compiler_error") {
				t.Fatal("missing diagnostic")
			}
			return string(mustBuildJSON(sceneReviewDecision{Outcome: "ready", Message: "Place head directly on body", Updates: []sceneModuleUpdate{{Object: "main", Module: "head", Attach: &buildstudio.ConstructionAttachment{Parent: "body", Anchor: "top"}}}})), nil
		}
		return `{"outcome":"approved","message":"The colored head and connected base match the request."}`, nil
	})}
	checkpoint := func(buildstudio.AssemblyRecipe) error { checkpoints++; return nil }
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	result, err := h.compileAndReviewScene(context.Background(), broken, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, checkpoint)
	if err != nil || !result.Plan.Validation.Buildable || calls != 2 || checkpoints != 3 {
		t.Fatalf("repair/review: %d %d %v", calls, checkpoints, err)
	}
	if result.Plan.Validation.Evidence.Physical != "not_tested" || result.Plan.Validation.Evidence.Semantic != "model_reviewed_not_proven" {
		t.Fatal("review claimed physical proof")
	}
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	_, err = h.compileAndReviewScene(context.Background(), result.Recipe, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, checkpoint)
	if err != nil || calls != 2 {
		t.Fatalf("recovery repeated accepted review: %v", err)
	}
	interrupted := scenePlanningFixture()
	interrupted.Scene.Objects[0].Modules[2].Attach = nil
	interrupted.Scene.Objects[0].Modules[2].Position.Y = 18
	h.LLM = sceneTestLLM(func(*http.Request) (string, error) { return "", errors.New("transport interrupted") })
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	_, _ = h.compileAndReviewScene(context.Background(), interrupted, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, func(r buildstudio.AssemblyRecipe) error { interrupted = r; return nil })
	if interrupted.Metadata["scene_repairs"] != "1" {
		t.Fatal("transport failure reset repair budget")
	}
}

func TestBuildSceneReviewCacheRequiresSameIdeaAndRequirements(t *testing.T) {
	for _, sample := range []struct {
		name   string
		change func(*buildstudio.AssemblyRecipe)
	}{
		{"source_idea", func(r *buildstudio.AssemblyRecipe) { r.Prompt = "a smiling standing figure" }},
		{"requirements", func(r *buildstudio.AssemblyRecipe) { r.Requirements = []string{"white head"} }},
		{"constraints", func(r *buildstudio.AssemblyRecipe) { r.Constraints.NoWheels = true }},
		{"subject", func(r *buildstudio.AssemblyRecipe) { r.Subject = "astronaut" }},
	} {
		t.Run(sample.name, func(t *testing.T) {
			calls := 0
			invalidated := false
			h := &Handler{LLM: sceneTestLLM(func(request *http.Request) (string, error) {
				calls++
				data, _ := io.ReadAll(request.Body)
				if !strings.Contains(string(data), "compiled_inspection") || !strings.Contains(string(data), "components_after_removing_assigned_parts") {
					t.Fatal("review omitted actual part-interface evidence")
				}
				if strings.Contains(string(data), "scene_review_note") || strings.Contains(string(data), "scene_review_hash") {
					t.Fatal("obsolete approval biased a fresh review")
				}
				if calls > 1 && !invalidated {
					t.Fatal("approval invalidation was not checkpointed before reviewing")
				}
				return `{"outcome":"approved","message":"The connected figure matches this request."}`, nil
			})}
			checkpoint := func(r buildstudio.AssemblyRecipe) error {
				if calls == 1 && r.Metadata["scene_review_hash"] == "" {
					invalidated = true
				}
				return nil
			}
			h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
			approved, err := h.compileAndReviewScene(context.Background(), scenePlanningFixture(), buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, checkpoint)
			if err != nil || calls != 1 {
				t.Fatalf("initial review: calls=%d err=%v", calls, err)
			}
			h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
			unchanged, err := h.compileAndReviewScene(context.Background(), approved.Recipe, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, checkpoint)
			if err != nil || calls != 1 || unchanged.Plan.ContentHash != approved.Plan.ContentHash {
				t.Fatalf("same request did not reuse review: calls=%d err=%v", calls, err)
			}
			changed := approved.Recipe
			sample.change(&changed)
			h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
			reviewed, err := h.compileAndReviewScene(context.Background(), changed, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, checkpoint)
			if err != nil || calls != 2 || reviewed.Plan.ContentHash != approved.Plan.ContentHash {
				t.Fatalf("different request reused geometry-only approval: calls=%d err=%v", calls, err)
			}
			if approved.Recipe.Metadata["scene_review_hash"] != unchanged.Recipe.Metadata["scene_review_hash"] {
				t.Fatal("a new review mutated the original approved recipe")
			}
		})
	}
}

func TestBuildSceneWorkerDBStoresReviewedModelAndMissingParts(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE build_session SET inventory_snapshot=$2 WHERE id=$1", seed.ID, []byte(`{"configured":true,"catalog_version":"","items":[],"revision":0}`)); err != nil {
		t.Fatal(err)
	}
	// Availability is read from the inventory tables, never the old session snapshot.
	if _, err := pool.Exec(ctx, "INSERT INTO brick_inventory(workspace_id,catalog_version,configured,revision,updated_by) VALUES($1,$2,true,1,$3)", seed.WorkspaceID, buildstudio.CatalogVersion, seed.CreatorUserID); err != nil {
		t.Fatal(err)
	}
	calls := 0
	h.LLM = sceneTestLLM(func(*http.Request) (string, error) {
		calls++
		if calls == 1 {
			recipe := scenePlanningFixture()
			return string(mustBuildJSON(buildPlanningDecision{Outcome: "ready", Recipe: &recipe})), nil
		}
		return `{"outcome":"approved","message":"The figure retains a connected base and colored head."}`, nil
	})
	if worked, err := h.BuildWorker.ProcessNext(ctx); err != nil || !worked {
		t.Fatal(err)
	}
	current, err := h.Queries.GetBuildSessionForWorker(ctx, seed.ID)
	if err != nil || current.Status != "completed" {
		t.Fatalf("scene worker status=%s error=%s read=%v", current.Status, current.Error.String, err)
	}
	row, err := h.Queries.GetBuildCreationInWorkspace(ctx, db.GetBuildCreationInWorkspaceParams{ID: current.CreationID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID})
	if err != nil {
		t.Fatal(err)
	}
	response, err := toBuildCreationResponse(row)
	if err != nil {
		t.Fatal(err)
	}
	if response.BuildPlan.Document.Scene == nil || !response.BuildPlan.InventoryCompared || len(response.BuildPlan.MissingParts) == 0 || response.Validation.Evidence.Semantic != "model_reviewed_not_proven" {
		t.Fatal("scene evidence/BOM lost at persistence boundary")
	}
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM build_creation").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate scene artifact")
	}
}

// Captures only returned text decisions, never credentials or private thinking blocks.
func sceneLiveLLM(sample func() string) *llm.Client {
	var requestNumber atomic.Int32
	transport := buildTestTransport(func(request *http.Request) (response *http.Response, err error) {
		requestID := requestNumber.Add(1)
		request, finish := traceSceneLiveRequest(request, sample(), requestID)
		status := 0
		defer func() { finish(status, err) }()
		response, err = http.DefaultTransport.RoundTrip(request)
		if err != nil {
			return response, err
		}
		status = response.StatusCode
		data, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		response.Body = io.NopCloser(bytes.NewReader(data))
		var payload struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal(data, &payload) == nil {
			var output strings.Builder
			for _, block := range payload.Content {
				if block.Type == "text" {
					output.WriteString(block.Text)
				}
			}
			if directory := os.Getenv("CHIMII_SCENE_LIVE_OUTPUT"); directory != "" && output.Len() > 0 {
				_ = os.WriteFile(filepath.Join(directory, fmt.Sprintf("%s-model-%d.txt", sample(), requestID)), []byte(output.String()), 0600)
			}
		}
		return response, nil
	})
	return llm.New(llm.Config{APIKey: os.Getenv("CHIMII_LLM_API_KEY"), BaseURL: os.Getenv("CHIMII_LLM_BASE_URL"), DefaultModel: os.Getenv("CHIMII_LLM_DEFAULT_MODEL"), HTTPClient: &http.Client{Transport: transport}})
}

// Explicit provider evaluation. Default tests never read credentials/contact it.
func TestBuildSceneLiveGeneration(t *testing.T) {
	if os.Getenv("CHIMII_SCENE_LIVE") != "1" {
		t.Skip("set CHIMII_SCENE_LIVE=1 to evaluate the configured model")
	}
	var currentSample string
	h := &Handler{LLM: sceneLiveLLM(func() string { return currentSample })}
	h.cfg.BuildPlannerEffort = os.Getenv("CHIMII_BUILD_PLANNER_EFFORT")
	h.cfg.BuildReviewEffort = os.Getenv("CHIMII_BUILD_REVIEW_EFFORT")
	if os.Getenv("CHIMII_SCENE_LIVE_CATALOG_DB") == "1" {
		h.Queries = db.New(testPool)
	} else {
		h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	}
	if !h.LLM.Enabled() {
		t.Fatal("utility model missing")
	}
	for _, sample := range []struct{ name, prompt string }{
		{"architecture", "搭一个有门窗、楼梯、二层平台和屋顶的两层小屋，完整细节，尺度配色由你决定"},
		{"vehicle", "搭一辆复古旅行巴士，有四个真实车轮、车窗、前灯、行李架，尺度配色由你决定"},
		{"spacecraft", "搭一艘探索飞船，有驾驶舱、两侧机翼和两个发动机，作为有底座的展示模型"},
		{"creature", "搭一只坐着的狐狸，有尖耳朵、眼睛、白色胸口和尾巴，尺度配色由你决定"},
		{"figure", "搭一个站着的宇航员，有头盔、胸前面板、双臂和双腿，姿态由你决定"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			currentSample = sample.name
			ctx, cancel := context.WithTimeout(context.Background(), 540*time.Second)
			defer cancel()
			started := time.Now()
			inventory := buildstudio.UnlimitedInventory()
			catalog := buildstudio.StarterCatalog
			if os.Getenv("CHIMII_SCENE_LIVE_CATALOG_DB") == "1" {
				active, err := loadActiveBuildCatalog(ctx, db.New(testPool))
				if err != nil {
					t.Fatal(err)
				}
				catalog = active.Parts
				inventory.CatalogVersion = active.Version
			}
			decision, err := h.planBuildRecipe(ctx, sample.prompt, nil, nil, 1, inventory, catalog)
			if err != nil {
				message := err.Error()
				for _, key := range []string{"CHIMII_LLM_API_KEY", "CHIMII_LLM_BASE_URL"} {
					if value := os.Getenv(key); value != "" {
						message = strings.ReplaceAll(message, value, "[redacted]")
					}
				}
				if len(message) > 1600 {
					message = message[:1600]
				}
				t.Fatalf("planning failed after %s: %s", time.Since(started), message)
			}
			if decision.Outcome != "ready" || decision.Recipe == nil || decision.Recipe.Scene == nil {
				t.Fatalf("clear idea was not a scene: %s", decision.Outcome)
			}
			save := func(recipe buildstudio.AssemblyRecipe) error {
				if output := os.Getenv("CHIMII_SCENE_LIVE_OUTPUT"); output != "" {
					return os.WriteFile(filepath.Join(output, sample.name+"-recipe.json"), mustBuildJSON(recipe), 0600)
				}
				return nil
			}
			_ = save(*decision.Recipe)
			result, err := h.compileAndReviewScene(ctx, *decision.Recipe, inventory, catalog, save)
			if output := os.Getenv("CHIMII_SCENE_LIVE_OUTPUT"); output != "" {
				_ = os.WriteFile(filepath.Join(output, sample.name+"-result.json"), mustBuildJSON(result), 0600)
				if err == nil {
					_ = os.WriteFile(filepath.Join(output, sample.name+".mpd"), []byte(result.MPD), 0600)
				}
			}
			if err != nil {
				t.Fatalf("compile/review: %v", err)
			}
			t.Logf("parts=%d modules=%d steps=%d elapsed=%s", result.Plan.Validation.PartCount, result.Plan.Validation.Evidence.Modules, result.Plan.Validation.StepCount, time.Since(started))
		})
	}
}

// Compiles an explicitly supplied scene against a synchronized local catalog.
// This is an asset/workflow fixture, separate from real-model acceptance.
func TestBuildSceneCatalogFixture(t *testing.T) {
	input, output := os.Getenv("CHIMII_SCENE_CATALOG_FIXTURE_INPUT"), os.Getenv("CHIMII_SCENE_CATALOG_FIXTURE_OUTPUT")
	if input == "" || output == "" {
		t.Skip("requires explicit local scene fixture input/output")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	var source buildstudio.CompileResult
	if err = json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	catalog, err := loadActiveBuildCatalog(context.Background(), db.New(testPool))
	if err != nil {
		t.Fatal(err)
	}
	result, err := buildstudio.CompileScene(context.Background(), source.Recipe, buildstudio.UnlimitedInventory(), catalog.Version, catalog.Parts, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.Plan.Parts {
		asset, err := db.New(testPool).GetLDrawPartRevisionByVersionAndPartID(context.Background(), db.GetLDrawPartRevisionByVersionAndPartIDParams{CatalogVersion: catalog.Version, PartID: p.LDrawID})
		if err != nil || len(asset.Payload) < 20 || string(asset.Payload[:4]) != "glTF" {
			t.Fatalf("missing real catalog GLB for %s: %v", p.LDrawID, err)
		}
	}
	if err = os.WriteFile(output, mustBuildJSON(result), 0600); err != nil {
		t.Fatal(err)
	}
	// Exercise the production catalog loader and real rendering separately from
	// any model review; no semantic approval is added to this authored fixture.
	h := &Handler{Queries: db.New(testPool)}
	images, visual, err := h.sceneReviewImages(context.Background(), result.Plan)
	if err != nil {
		t.Fatal(err)
	}
	dir := strings.TrimSuffix(output, filepath.Ext(output)) + "-views"
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i, image := range images {
		if err = os.WriteFile(filepath.Join(dir, visual.Views[i].Direction+".png"), image.Data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(filepath.Join(dir, "render-evidence.json"), mustBuildJSON(visual), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("catalog=%s parts=%d steps=%d", catalog.Version, len(result.Plan.Placements), len(result.Plan.Steps))
}

// Resumes the compiled candidate of an interrupted live review. This never
// replaces its source plan with a hand-authored scene or resets repair state.
func TestBuildSceneLiveSavedReview(t *testing.T) {
	input, output := os.Getenv("CHIMII_SCENE_LIVE_REVIEW_INPUT"), os.Getenv("CHIMII_SCENE_LIVE_REVIEW_OUTPUT")
	if input == "" || output == "" || os.Getenv("CHIMII_SCENE_LIVE") != "1" {
		t.Skip("requires an explicit live model candidate and output")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	var source buildstudio.CompileResult
	if err = json.Unmarshal(data, &source); err != nil || source.Recipe.Scene == nil {
		t.Fatalf("live source: %v", err)
	}
	catalog, err := loadActiveBuildCatalog(context.Background(), db.New(testPool))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{LLM: sceneLiveLLM(func() string { return strings.TrimSuffix(filepath.Base(output), ".json") })}
	ctx, cancel := context.WithTimeout(context.Background(), 540*time.Second)
	defer cancel()
	started := time.Now()
	inventory := buildstudio.UnlimitedInventory()
	inventory.CatalogVersion = catalog.Version
	result, err := h.compileAndReviewScene(ctx, source.Recipe, inventory, catalog.Parts, func(recipe buildstudio.AssemblyRecipe) error {
		return os.WriteFile(output+".checkpoint.json", mustBuildJSON(recipe), 0600)
	})
	if writeErr := os.WriteFile(output, mustBuildJSON(result), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if err != nil {
		t.Fatalf("resumed compile/review: %v", err)
	}
	t.Logf("parts=%d modules=%d steps=%d elapsed=%s", result.Plan.Validation.PartCount, result.Plan.Validation.Evidence.Modules, result.Plan.Validation.StepCount, time.Since(started))
}

func TestBuildSceneRevisionUsesBoundedPatches(t *testing.T) {
	original := scenePlanningFixture()
	original.Scene.Objects[0].Modules[2].Attach = nil
	original.Scene.Objects[0].Modules[2].Position.Y = 18
	calls := 0
	h := &Handler{LLM: sceneTestLLM(func(request *http.Request) (string, error) {
		calls++
		if calls == 1 {
			return `{"outcome":"ready","message":"fix","updates":[{"object":"main","module":"head","size":{"x":3.5,"y":3,"z":4}}]}`, nil
		}
		if calls == 2 {
			data, _ := io.ReadAll(request.Body)
			if !strings.Contains(string(data), "revision_validation_error") {
				t.Fatal("format correction omitted precise diagnostic")
			}
			return `{"outcome":"ready","message":"attach head above body","updates":[{"object":"main","module":"head","attach":{"parent":"body","anchor":"top"}}]}`, nil
		}
		return `{"outcome":"approved","message":"connected colored head matches the requested figure"}`, nil
	})}
	saved := original
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	result, err := h.compileAndReviewScene(context.Background(), original, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, func(r buildstudio.AssemblyRecipe) error { saved = r; return nil })
	if err != nil || !result.Plan.Validation.Buildable || calls != 3 || saved.Metadata["scene_repairs"] != "2" {
		t.Fatalf("bounded patch flow: calls=%d budget=%s err=%v", calls, saved.Metadata["scene_repairs"], err)
	}
	if saved.Scene.Objects[0].Modules[2].Color != original.Scene.Objects[0].Modules[2].Color || len(saved.Scene.Objects[0].Modules) != 3 {
		t.Fatal("patch changed unaffected identity")
	}
}
