package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

func TestBuildCompositionDBVersioningContextAndInventory(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE build_job SET status='completed'; UPDATE build_session SET status='failed'"); err != nil {
		t.Fatal(err)
	}
	c, _ := circuit.FindCatalog("dfrobot-edu0080-en")
	inv, err := h.Queries.SaveCircuitInventory(ctx, db.SaveCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID, CatalogVersion: c.Version, Quantities: mustBuildJSON(c.Inventory())})
	if err != nil {
		t.Fatal(err)
	}
	output := "BOS0021"
	var captured string
	h.LLM = llm.New(llm.Config{BaseURL: "https://composition-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
		request, _ := io.ReadAll(r.Body)
		captured = string(request)
		decision := circuit.ConversationDecision{Outcome: "ready", Title: "My combination", Composition: &circuit.CompositionSpec{Inputs: []string{"BOS0002-R", "BOS0013"}, Operation: "and", Output: output}}
		body := mustBuildJSON(map[string]any{"id": "test", "type": "message", "role": "assistant", "model": "test", "content": []map[string]string{{"type": "text", "text": string(mustBuildJSON(decision))}}, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}})
	req := createBuildSessionRequest{Prompt: "both button and motion control a fan", ClientRequestID: "00000000-0000-4000-8000-000000000601", Kind: "circuit", Circuit: &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, InventoryRevision: inv.Revision, Locale: "en"}}
	first := conversationResponse(t, conversationRequest(h, seed, "", req, ""), 202)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(first.ID))
	if err != nil || s.Status != "completed" {
		t.Fatalf("composition did not complete: %s %v", s.Status, err)
	}
	original, err := h.Queries.GetCircuitCreation(ctx, db.GetCircuitCreationParams{ID: s.CircuitCreationID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent"})
	if err != nil {
		t.Fatal(err)
	}
	var doc circuit.Document
	if err = json.Unmarshal(original.Document, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Composition == nil || doc.Behavior == nil || !doc.Behavior.Passed || len(doc.Behavior.Cases) != 5 {
		t.Fatal("missing functional evidence")
	}
	output = "BOS0017-R"
	req.Prompt = "change to a light, keep both conditions"
	req.ClientRequestID = "00000000-0000-4000-8000-000000000602"
	req.ExpectedSessionID = first.ID
	req.ExpectedRevision = s.Revision
	req.SourceKind = "circuit"
	req.SourceCreationID = uuidToString(original.ID)
	req.ExpectedContentHash = doc.ContentHash
	next := conversationResponse(t, conversationRequest(h, seed, first.ID, req, ""), 202)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err = h.Queries.GetBuildSessionForWorker(ctx, parseUUID(next.ID))
	if err != nil || s.Status != "completed" || s.CircuitCreationID == original.ID {
		t.Fatal("missing independent version")
	}
	if !strings.Contains(captured, "previous_composition") || !strings.Contains(captured, "BOS0021") {
		t.Fatal("source functional requirements were not passed to planner")
	}
	untouched, err := h.Queries.GetCircuitCreation(ctx, db.GetCircuitCreationParams{ID: original.ID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent"})
	if err != nil || string(untouched.Document) != string(original.Document) {
		t.Fatal("source document changed")
	}
	quantities := c.Inventory()
	quantities[output] = 0
	inv, err = h.Queries.UpdateCircuitInventory(ctx, db.UpdateCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID, CatalogVersion: c.Version, Quantities: mustBuildJSON(quantities), ExpectedRevision: inv.Revision})
	if err != nil {
		t.Fatal(err)
	}
	req.ClientRequestID = "00000000-0000-4000-8000-000000000603"
	req.ExpectedSessionID = next.ID
	req.ExpectedRevision = s.Revision
	req.Circuit.InventoryRevision = inv.Revision
	missing := conversationResponse(t, conversationRequest(h, seed, first.ID, req, ""), 202)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err = h.Queries.GetBuildSessionForWorker(ctx, parseUUID(missing.ID))
	if err != nil || s.Status != "failed" || s.CircuitCreationID.Valid {
		t.Fatal("missing inventory produced a creation")
	}
}
