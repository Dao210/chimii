package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/middleware"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
)

func TestParseBuildShapeDecision(t *testing.T) {
	d := validBuildDecision()
	d.Recipe.Version = 3
	d.Recipe.Modules = []buildstudio.ModuleInstance{}
	d.Recipe.Design = &buildstudio.DesignSpec{Version: 1, Mode: "static", Shapes: []buildstudio.ShapeNode{{ID: "outline", Label: "Outline", Kind: "ellipse", Operation: "add", Size: buildstudio.DesignVector{X: 8, Y: 6, Z: 8}, Color: 1}}}
	if _, err := parseBuildDecision(string(mustBuildJSON(d))); err != nil {
		t.Fatal(err)
	}
	d.Recipe.Design.Mode = "working-clock"
	if _, err := parseBuildDecision(string(mustBuildJSON(d))); err == nil {
		t.Fatal("unsupported motion accepted")
	}
}

func TestBuildWorkerDBLiveReusableAvailabilityAndImmutableDesignRevisions(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	ctx := context.Background()
	h.LLM = buildFakeLLM(func() buildPlanningDecision { return validBuildDecision() })
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	h.LLM = nil // Parameter edits do not call or require a model.
	design := &buildstudio.DesignSpec{Version: 1, Mode: "static", Shapes: []buildstudio.ShapeNode{{ID: "base", Label: "Base", Kind: "box", Operation: "add", Size: buildstudio.DesignVector{X: 4, Y: 6, Z: 2}, Color: 1}}}
	submit := func(index int, source, hash string, want int) buildSessionResponse {
		t.Helper()
		input := createBuildSessionRequest{Prompt: "custom shape", ClientRequestID: fmt.Sprintf("00000000-0000-4000-8000-%012d", index), Design: design, SourceCreationID: source, ExpectedContentHash: hash}
		r := httptest.NewRequest(http.MethodPost, "/api/build/sessions", strings.NewReader(string(mustBuildJSON(input))))
		r.Header.Set("X-User-ID", uuidToString(seed.CreatorUserID))
		r = r.WithContext(middleware.SetMemberContext(r.Context(), uuidToString(seed.WorkspaceID), db.Member{}))
		w := httptest.NewRecorder()
		h.CreateBuildSession(w, r)
		if w.Code != want {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var response buildSessionResponse
		if want == http.StatusAccepted {
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
		}
		return response
	}
	first := submit(1, "", "", http.StatusAccepted)
	if _, err := pool.Exec(ctx, "INSERT INTO brick_inventory(workspace_id,catalog_version,updated_by) VALUES($1,$2,$3)", seed.WorkspaceID, buildstudio.CatalogVersion, seed.CreatorUserID); err != nil {
		t.Fatal(err)
	}
	// Inventory was changed AFTER queueing. The worker must use the current
	// reusable collection and must not resurrect a frozen unlimited snapshot.
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	failed, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(first.ID))
	if err != nil || failed.Error.String != buildstudio.BuildErrorInsufficientInventory {
		t.Fatalf("live availability not used: %s %v", failed.Error.String, err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO brick_inventory_item(inventory_id,part_key,color_code,quantity) SELECT id,'brick-2x4',1,2 FROM brick_inventory"); err != nil {
		t.Fatal(err)
	}
	second := submit(2, "", "", http.StatusAccepted)
	duplicate := submit(2, "", "", http.StatusAccepted)
	if second.ID != duplicate.ID {
		t.Fatal("duplicate task")
	}
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	session, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(second.ID))
	if err != nil || session.Status != "completed" {
		t.Fatalf("%s %v", session.Status, err)
	}
	load := func(id string) db.BuildCreation {
		t.Helper()
		row, err := h.Queries.GetBuildCreationInWorkspace(ctx, db.GetBuildCreationInWorkspaceParams{ID: parseUUID(id), WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID})
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	original := load(uuidToString(session.CreationID))
	originalResponse, err := toBuildCreationResponse(original)
	if err != nil {
		t.Fatal(err)
	}
	if originalResponse.BuildPlan.Document == nil || originalResponse.BuildPlan.Inventory != nil || string(session.InventorySnapshot) != "{}" || string(original.InventorySnapshot) != "{}" {
		t.Fatal("new design missing or stock snapshot persisted")
	}
	if _, err = pool.Exec(ctx, "UPDATE build_creation SET current_step=1,progress_revision=1 WHERE id=$1", original.ID); err != nil {
		t.Fatal(err)
	}
	design.Shapes[0].Size.Y = 3
	submit(3, uuidToString(original.ID), "stale", http.StatusConflict)
	revision := submit(4, uuidToString(original.ID), originalResponse.BuildPlan.ContentHash, http.StatusAccepted)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	current, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(revision.ID))
	if err != nil || current.Status != "completed" {
		t.Fatalf("revision %s %v", current.Status, err)
	}
	updated, _ := toBuildCreationResponse(load(uuidToString(current.CreationID)))
	if updated.BuildPlan.Document.ParentCreationID != uuidToString(original.ID) || updated.BuildPlan.ContentHash == originalResponse.BuildPlan.ContentHash {
		t.Fatal("revision lineage or content missing")
	}
	preserved := load(uuidToString(original.ID))
	if string(preserved.BuildPlan) != string(original.BuildPlan) || preserved.CurrentStep != 1 || preserved.ProgressRevision != 1 {
		t.Fatal("original plan or progress was changed")
	}
	var quantity int
	if err = pool.QueryRow(ctx, "SELECT quantity FROM brick_inventory_item").Scan(&quantity); err != nil || quantity != 2 {
		t.Fatal("reusable parts were consumed")
	}
}
