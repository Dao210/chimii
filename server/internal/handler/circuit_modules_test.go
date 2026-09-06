package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	"github.com/chimii-ai/chimii/server/internal/middleware"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
	"github.com/google/uuid"
)

func circuitBoxRequest(method string, body any, child string) *http.Request {
	return withURLParam(circuitTestRequest(method, "", body, child), "kitID", "dfrobot-edu0080-en")
}

func circuitBox(t *testing.T, quantities map[string]int, revision int, child string, status int) circuitInventoryResponse {
	t.Helper()
	c, _ := circuit.FindCatalog("dfrobot-edu0080-en")
	w := httptest.NewRecorder()
	testHandler.SaveCircuitInventory(w, circuitBoxRequest("PUT", map[string]any{"catalog_version": c.Version, "quantities": quantities, "expected_revision": revision}, child))
	if w.Code != status {
		t.Fatalf("box status=%d: %s", w.Code, w.Body.String())
	}
	var box circuitInventoryResponse
	if status == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), &box); err != nil {
			t.Fatal(err)
		}
	}
	return box
}

func resetCircuitBox(t *testing.T) circuit.Catalog {
	t.Helper()
	clean := func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM circuit_inventory WHERE workspace_id=$1 AND parent_user_id=$2`, testWorkspaceID, testUserID); err != nil {
			t.Error(err)
		}
	}
	clean()
	t.Cleanup(clean)
	c, _ := circuit.FindCatalog("dfrobot-edu0080-en")
	return c
}

func TestCircuitSavedInventoryAndScope(t *testing.T) {
	c := resetCircuitBox(t)
	read := func(r *http.Request) circuitInventoryResponse {
		t.Helper()
		w := httptest.NewRecorder()
		testHandler.GetCircuitInventory(w, r)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var box circuitInventoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &box); err != nil {
			t.Fatal(err)
		}
		return box
	}
	empty := read(circuitBoxRequest("GET", nil, ""))
	if empty.Confirmed || empty.Revision != 0 || !empty.CanEdit {
		t.Fatal("new box claimed ownership")
	}
	for _, n := range empty.Quantities {
		if n != 0 {
			t.Fatal("new box is not empty")
		}
	}
	first := circuitBox(t, c.Inventory(), 0, "", 200)
	if !first.Confirmed || first.Revision != 1 {
		t.Fatal("box was not persisted")
	}
	child := read(circuitBoxRequest("GET", nil, uuid.NewString()))
	if child.CanEdit || child.Revision != 1 {
		t.Fatal("child did not inherit a read-only family box")
	}
	circuitBox(t, c.Inventory(), 1, uuid.NewString(), 403)
	circuitBox(t, c.Inventory(), 0, "", 409)
	incomplete := c.Inventory()
	delete(incomplete, "BOS0036")
	circuitBox(t, incomplete, 1, "", 400)
	unknown := c.Inventory()
	unknown["other-brand"] = 1
	circuitBox(t, unknown, 1, "", 400)
	for _, scope := range []string{"workspace", "parent"} {
		r := circuitBoxRequest("GET", nil, "")
		if scope == "workspace" {
			r = r.WithContext(middleware.SetMemberContext(r.Context(), uuid.NewString(), db.Member{}))
		} else {
			r.Header.Set("X-User-ID", uuid.NewString())
		}
		if read(r).Confirmed {
			t.Fatalf("box leaked across %s", scope)
		}
	}
	second := circuitBox(t, c.Inventory(), 1, "", 200)
	if second.Revision != 2 {
		t.Fatal("revision did not advance")
	}
}

func TestCircuitModulesRequireSavedInventoryAndReplaySnapshot(t *testing.T) {
	c := resetCircuitBox(t)
	input := circuitCreateRequest{ClientRequestID: uuid.NewString(), KitID: c.KitID, CatalogVersion: c.Version, ProjectID: c.Projects[0].ID, Inventory: c.Inventory()}
	circuitTestCreate(t, input, "", 409)
	revision := int32(1)
	input.InventoryRevision = &revision
	circuitTestCreate(t, input, "", 409)
	circuitBox(t, c.Inventory(), 0, "", 200)
	for _, project := range c.Projects {
		input.ClientRequestID = uuid.NewString()
		input.ProjectID = project.ID
		created := circuitTestCreate(t, input, "", 201)
		var doc circuit.Document
		if err := json.Unmarshal(created.Document, &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Version != 2 || doc.InventoryRevision == nil || *doc.InventoryRevision != 1 || doc.Validation.PhysicalVerification != "not_tested" {
			t.Fatalf("bad provenance: %+v", doc)
		}
		resealed, err := circuit.SealDocument(doc)
		if err != nil || resealed.ContentHash != doc.ContentHash {
			t.Fatal("invalid snapshot hash")
		}
	}
	created := circuitTestCreate(t, input, "", 200)
	fewer := c.Inventory()
	fewer["BOSON-CABLE-10CM"] = 0
	circuitBox(t, fewer, 1, "", 200)
	replay := circuitTestCreate(t, input, "", 200)
	if string(created.Document) != string(replay.Document) {
		t.Fatal("inventory edit changed a saved snapshot")
	}
	input.ClientRequestID = uuid.NewString()
	circuitTestCreate(t, input, "", 409)
	revision = 2
	circuitTestCreate(t, input, "", 409)
	input.Inventory = fewer
	circuitTestCreate(t, input, "", 422)
}

func TestCircuitInventoryRecheckedAfterPlanning(t *testing.T) {
	c := resetCircuitBox(t)
	circuitBox(t, c.Inventory(), 0, "", 200)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate the parent editing the box while the model is still generating.
		if _, err := testPool.Exec(r.Context(), `UPDATE circuit_inventory SET revision=revision+1 WHERE workspace_id=$1 AND parent_user_id=$2`, testWorkspaceID, testUserID); err != nil {
			t.Error(err)
		}
		writeJSON(w, 200, map[string]any{"id": "test-message", "type": "message", "role": "assistant", "model": "test-model", "content": []map[string]string{{"type": "text", "text": `{"project_id":"boson-button-light","title":"Light"}`}}, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
	}))
	defer provider.Close()
	previous := testHandler.LLM
	testHandler.LLM = llm.New(llm.Config{BaseURL: provider.URL, DefaultModel: "test-model"})
	defer func() { testHandler.LLM = previous }()
	revision := int32(1)
	input := circuitCreateRequest{ClientRequestID: uuid.NewString(), KitID: c.KitID, CatalogVersion: c.Version, Prompt: "button light", Inventory: c.Inventory(), InventoryRevision: &revision}
	circuitTestCreate(t, input, "", 409)
	var count int
	if err := testPool.QueryRow(context.Background(), `SELECT count(*) FROM circuit_creation WHERE client_request_id=$1`, input.ClientRequestID).Scan(&count); err != nil || count != 0 {
		t.Fatal("stale box was saved", err, count)
	}
}

func TestCircuitTrialEvidenceIdempotencyAndPrivacy(t *testing.T) {
	child := uuid.NewString()
	creation := circuitTestCreate(t, circuitTestInput(), child, 201)
	t.Cleanup(func() {
		_, err := testPool.Exec(context.Background(), `DELETE FROM circuit_trial WHERE creation_id=$1`, creation.ID)
		if err != nil {
			t.Error(err)
		}
	})
	input := map[string]any{"client_request_id": uuid.NewString(), "hardware_label": "SC-500 sample A", "result": "worked", "notes": "Observed audio", "adult_checked": true}
	create := func(actor string, status int) circuitTrialResponse {
		t.Helper()
		w := httptest.NewRecorder()
		testHandler.CreateCircuitTrial(w, circuitTestRequest("POST", creation.ID, input, actor))
		if w.Code != status {
			t.Fatalf("trial status=%d: %s", w.Code, w.Body.String())
		}
		var v circuitTrialResponse
		if status == 200 || status == 201 {
			if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
		}
		return v
	}
	create(child, 400)
	w := httptest.NewRecorder()
	testHandler.UpdateCircuitProgress(w, circuitTestRequest("PUT", creation.ID, map[string]any{"current_step": 22, "expected_revision": 0, "observation": "not_tried"}, child))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	input["adult_checked"] = false
	create(child, 400)
	input["adult_checked"] = true
	first := create(child, 201)
	replay := create(child, 200)
	if first.ID != replay.ID || first.EvidenceKind != "family_report" {
		t.Fatal("trial evidence or idempotency lost")
	}
	input["notes"] = "different"
	create(child, 409)
	for _, actor := range []string{"", uuid.NewString()} {
		create(actor, 404)
		w := httptest.NewRecorder()
		testHandler.ListCircuitTrials(w, circuitTestRequest("GET", creation.ID, nil, actor))
		if w.Code != 404 {
			t.Fatal("trial leaked to another actor")
		}
	}
	for _, scope := range []string{"workspace", "parent"} {
		r := circuitTestRequest("GET", creation.ID, nil, child)
		if scope == "workspace" {
			r = r.WithContext(middleware.SetMemberContext(r.Context(), uuid.NewString(), db.Member{}))
		} else {
			r.Header.Set("X-User-ID", uuid.NewString())
		}
		w := httptest.NewRecorder()
		testHandler.ListCircuitTrials(w, r)
		if w.Code != 404 {
			t.Fatal("trial scope leaked", scope)
		}
	}
	w = httptest.NewRecorder()
	testHandler.ListCircuitTrials(w, circuitTestRequest("GET", creation.ID, nil, child))
	var list struct {
		Trials []circuitTrialResponse `json:"trials"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Trials) != 1 {
		t.Fatal("trial retry duplicated", err)
	}
	w = httptest.NewRecorder()
	testHandler.GetCircuitCreation(w, circuitTestRequest("GET", creation.ID, nil, child))
	var after circuitCreationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	var beforeDoc, afterDoc circuit.Document
	_ = json.Unmarshal(creation.Document, &beforeDoc)
	_ = json.Unmarshal(after.Document, &afterDoc)
	if beforeDoc.ContentHash != afterDoc.ContentHash || first.DocumentHash != beforeDoc.ContentHash || afterDoc.Validation.PhysicalVerification != "not_tested" {
		t.Fatal("family report promoted physical evidence")
	}
}

func TestCircuitCleanupIncludesBoxesAndTrials(t *testing.T) {
	c := resetCircuitBox(t)
	circuitBox(t, c.Inventory(), 0, "", 200)
	creation := circuitTestCreate(t, circuitTestInput(), "", 201)
	ctx := context.Background()
	for _, scope := range []string{"parent", "workspace"} {
		t.Run(scope, func(t *testing.T) {
			tx, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			other := uuid.NewString()
			if _, err := tx.Exec(ctx, `INSERT INTO circuit_trial (id,workspace_id,parent_user_id,actor_key,creation_id,document_hash,request_hash,hardware_label,result,notes) VALUES ($1,$2,$3,'parent',$4,'hash','request','test','worked','')`, uuid.NewString(), testWorkspaceID, testUserID, creation.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO circuit_inventory (workspace_id,parent_user_id,kit_id,catalog_version,quantities) VALUES ($1,$2,'other','v1','{}')`, testWorkspaceID, other); err != nil {
				t.Fatal(err)
			}
			q := testHandler.Queries.WithTx(tx)
			session, err := q.CreateBuildSession(ctx, db.CreateBuildSessionParams{WorkspaceID: parseUUID(testWorkspaceID), CreatorUserID: parseUUID(testUserID), ClientRequestID: parseUUID(uuid.NewString()), Prompt: "cleanup fixture", Status: "queued", InventorySnapshot: []byte(`{}`)})
			if err != nil {
				t.Fatal(err)
			}
			if err = appendBuildUserMessage(ctx, q, session, "cleanup fixture", "fixture", "fixture"); err != nil {
				t.Fatal(err)
			}
			if scope == "parent" {
				err = q.DeleteBuildAndChildDataForParentInWorkspace(ctx, db.DeleteBuildAndChildDataForParentInWorkspaceParams{WorkspaceID: parseUUID(testWorkspaceID), ParentUserID: parseUUID(testUserID)})
			} else {
				err = q.DeleteWorkspace(ctx, parseUUID(testWorkspaceID))
			}
			if err != nil {
				t.Fatal(err)
			}
			var remainingMessages int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM build_message WHERE conversation_id=$1", session.ConversationID).Scan(&remainingMessages); err != nil || remainingMessages != 0 {
				t.Fatal("cleanup left conversation messages", remainingMessages, err)
			}
			for _, table := range []string{"circuit_inventory", "circuit_trial"} {
				var count int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE workspace_id=$1 AND parent_user_id=$2`, testWorkspaceID, testUserID).Scan(&count); err != nil || count != 0 {
					t.Fatal("cleanup left rows", table, count, err)
				}
			}
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM circuit_inventory WHERE workspace_id=$1 AND parent_user_id=$2`, testWorkspaceID, other).Scan(&count); err != nil {
				t.Fatal(err)
			}
			want := 0
			if scope == "parent" {
				want = 1
			}
			if count != want {
				t.Fatal("cleanup crossed the requested scope", count, want)
			}
		})
	}
}
