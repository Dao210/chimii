package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	"github.com/chimii-ai/chimii/server/internal/middleware"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

func conversationRequest(h *Handler, seed db.BuildSession, path string, input any, child string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/build/sessions", strings.NewReader(string(mustBuildJSON(input))))
	req.Header.Set("X-User-ID", uuidToString(seed.CreatorUserID))
	if child != "" {
		req.Header.Set("X-Actor-Source", "child_session")
		req.Header.Set("X-Child-Profile-ID", child)
	}
	req = req.WithContext(middleware.SetMemberContext(req.Context(), uuidToString(seed.WorkspaceID), db.Member{}))
	if path != "" {
		req = withURLParam(req, "conversationID", path)
	}
	out := httptest.NewRecorder()
	h.CreateBuildSession(out, req)
	return out
}
func conversationResponse(t *testing.T, r *httptest.ResponseRecorder, want int) buildSessionResponse {
	t.Helper()
	if r.Code != want {
		t.Fatalf("status %d want %d: %s", r.Code, want, r.Body.String())
	}
	var v buildSessionResponse
	if want == 202 {
		if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
	}
	return v
}
func TestBuildConversationDBCircuitVersionsReplayScopeAndInventory(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE build_job SET status='completed'; UPDATE build_session SET status='failed'; ALTER TABLE brick_inventory RENAME TO unavailable_brick_inventory"); err != nil {
		t.Fatal(err)
	}
	c := circuit.StarterCatalog()
	quantities := map[string]int{}
	for _, p := range c.Parts {
		quantities[p.ID] = p.Quantity
	}
	inv, err := h.Queries.SaveCircuitInventory(ctx, db.SaveCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID, CatalogVersion: c.Version, Quantities: mustBuildJSON(quantities)})
	if err != nil {
		t.Fatal(err)
	}
	input := createBuildSessionRequest{Prompt: "收音机", ClientRequestID: "00000000-0000-4000-8000-000000000201", Kind: "circuit", Circuit: &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, InventoryRevision: inv.Revision, ProjectID: "fm-radio", Locale: "zh-Hans"}}
	first := conversationResponse(t, conversationRequest(h, seed, "", input, ""), 202)
	replay := conversationResponse(t, conversationRequest(h, seed, "", input, ""), 202)
	if first.ID != replay.ID {
		t.Fatal("duplicate run")
	}
	changed := input
	changed.Prompt = "different"
	conversationResponse(t, conversationRequest(h, seed, "", changed, ""), 409)
	next := input
	next.ClientRequestID = "00000000-0000-4000-8000-000000000202"
	next.ExpectedSessionID = first.ID
	next.ExpectedRevision = 1
	conversationResponse(t, conversationRequest(h, seed, first.ID, next, ""), 409) // busy
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	done, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(first.ID))
	if err != nil || done.Status != "completed" || !done.CircuitCreationID.Valid || done.CreationID.Valid {
		t.Fatalf("not circuit completed: %+v %v", done, err)
	}
	original, err := h.Queries.GetCircuitCreation(ctx, db.GetCircuitCreationParams{ID: done.CircuitCreationID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent"})
	if err != nil {
		t.Fatal(err)
	}
	var doc circuit.Document
	if err = json.Unmarshal(original.Document, &doc); err != nil {
		t.Fatal(err)
	}
	next.SourceCreationID = uuidToString(original.ID)
	next.ExpectedContentHash = "stale"
	conversationResponse(t, conversationRequest(h, seed, first.ID, next, ""), 409)
	next.ExpectedContentHash = doc.ContentHash
	next.Circuit = &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, InventoryRevision: inv.Revision, ProjectID: "switch-light", Locale: "zh-Hans"}
	next.Prompt = "改成灯"
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); responses <- conversationRequest(h, seed, first.ID, next, "") }()
	}
	wg.Wait()
	close(responses)
	var second buildSessionResponse
	for response := range responses {
		v := conversationResponse(t, response, 202)
		if second.ID != "" && second.ID != v.ID {
			t.Fatal("duplicate continuation")
		}
		second = v
	}
	if second.ID == first.ID || second.ConversationID != first.ID {
		t.Fatal("old run reused")
	}
	conversationResponse(t, conversationRequest(h, seed, first.ID, next, "00000000-0000-4000-8000-000000000099"), 404)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	if worked, err := h.BuildWorker.ProcessNext(ctx); err != nil || worked {
		t.Fatalf("job unexpectedly runnable %v %v", worked, err)
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM circuit_creation").Scan(&count); err != nil || count != 2 {
		t.Fatalf("creations %d %v", count, err)
	}
	rows, err := h.Queries.ListBuildMessages(ctx, db.ListBuildMessagesParams{ConversationID: parseUUID(first.ID), PageSize: 50})
	if err != nil || len(rows) != 4 {
		t.Fatalf("messages %d %v", len(rows), err)
	}
	for _, v := range rows {
		if v.Role == "assistant" && v.Kind != "result" {
			t.Fatal("missing committed result")
		}
	}
	old, _ := h.Queries.GetCircuitCreation(ctx, db.GetCircuitCreationParams{ID: original.ID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent"})
	if string(old.Document) != string(original.Document) || old.ProgressRevision != 0 {
		t.Fatal("source mutated")
	}
	current, _ := h.Queries.GetCircuitInventory(ctx, db.GetCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID})
	if string(current.Quantities) != string(inv.Quantities) || current.Revision != inv.Revision {
		t.Fatal("inventory consumed")
	}
	// The aggregated query must read both real artifact tables and paginate stably.
	list, err := h.Queries.ListInventionSummaries(ctx, db.ListInventionSummariesParams{WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent", PageSize: 1})
	if err != nil || len(list) != 1 || list[0].Kind != "circuit" || list[0].Completed != false {
		t.Fatalf("gallery %+v %v", list, err)
	}
	tail, err := h.Queries.ListInventionSummaries(ctx, db.ListInventionSummariesParams{WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID, ActorKey: "parent", PageSize: 1, BeforeTime: list[0].CreatedAt, BeforeID: list[0].ID, BeforeKind: list[0].Kind})
	if err != nil || len(tail) != 1 || tail[0].ID == list[0].ID {
		t.Fatalf("pagination %+v %v", tail, err)
	}
	// A new request with an old box revision fails without adding an artifact.
	newest, _ := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(second.ID))
	next.ExpectedSessionID = second.ID
	next.ExpectedRevision = newest.Revision
	next.ClientRequestID = "00000000-0000-4000-8000-000000000203"
	if _, err = pool.Exec(ctx, "UPDATE circuit_inventory SET revision=revision+1"); err != nil {
		t.Fatal(err)
	}
	stale := conversationResponse(t, conversationRequest(h, seed, first.ID, next, ""), 202)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	failed, _ := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(stale.ID))
	if failed.Error.String != "CIRCUIT_INVENTORY_CHANGED" {
		t.Fatalf("stale box accepted: %+v", failed)
	}
}

func TestBuildConversationDBAnswerReplayExpiryAndSiblingIsolation(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE build_job SET status='completed'; UPDATE build_session SET status='failed'"); err != nil {
		t.Fatal(err)
	}
	c := circuit.StarterCatalog()
	child := "00000000-0000-4000-8000-000000000011"
	input := createBuildSessionRequest{Prompt: "lamp", ClientRequestID: "00000000-0000-4000-8000-000000000301", Kind: "circuit", Circuit: &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, ProjectID: "switch-light", Locale: "en"}}
	first := conversationResponse(t, conversationRequest(h, seed, "", input, child), 202)
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	pending, _ := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(first.ID))
	if pending.Status != "clarifying" {
		t.Fatal(pending.Status)
	}
	question := toBuildSessionResponse(pending).Question
	answer := input
	answer.ClientRequestID = "00000000-0000-4000-8000-000000000302"
	answer.ExpectedSessionID = first.ID
	answer.ExpectedRevision = pending.Revision
	answer.QuestionID = question.ID
	answer.Prompt = "ready"
	// Sibling reads and writes cannot enter this conversation.
	for _, other := range []string{"00000000-0000-4000-8000-000000000012", ""} {
		conversationResponse(t, conversationRequest(h, seed, first.ID, answer, other), 404)
	}
	read := httptest.NewRequest("GET", "/", nil)
	read.Header.Set("X-User-ID", uuidToString(seed.CreatorUserID))
	read.Header.Set("X-Actor-Source", "child_session")
	read.Header.Set("X-Child-Profile-ID", "00000000-0000-4000-8000-000000000012")
	read = withURLParam(read, "conversationID", first.ID)
	read = read.WithContext(middleware.SetMemberContext(read.Context(), uuidToString(seed.WorkspaceID), db.Member{}))
	out := httptest.NewRecorder()
	h.GetBuildConversation(out, read)
	if out.Code != 404 {
		t.Fatal(out.Code)
	}
	quantities := map[string]int{}
	for _, p := range c.Parts {
		quantities[p.ID] = p.Quantity
	}
	inv, err := h.Queries.SaveCircuitInventory(ctx, db.SaveCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID, CatalogVersion: c.Version, Quantities: mustBuildJSON(quantities)})
	if err != nil {
		t.Fatal(err)
	}
	answer.Circuit = &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, InventoryRevision: inv.Revision, ProjectID: "switch-light"}
	v := conversationResponse(t, conversationRequest(h, seed, first.ID, answer, child), 202)
	if v.Revision != pending.Revision+1 {
		t.Fatal("answer did not advance revision")
	}
	conversationResponse(t, conversationRequest(h, seed, first.ID, answer, child), 202)
	modified := answer
	modified.Prompt = "other"
	conversationResponse(t, conversationRequest(h, seed, first.ID, modified, child), 409)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	// A stale question cannot become a fresh command accidentally.
	modified.ClientRequestID = "00000000-0000-4000-8000-000000000303"
	conversationResponse(t, conversationRequest(h, seed, first.ID, modified, child), 409)
	if _, err = pool.Exec(ctx, "UPDATE build_session SET expires_at=now()-interval '1 minute' WHERE id=$1", parseUUID(first.ID)); err != nil {
		t.Fatal(err)
	}
	fresh := input
	fresh.ClientRequestID = "00000000-0000-4000-8000-000000000304"
	fresh.ExpectedSessionID = first.ID
	fresh.ExpectedRevision = v.Revision
	fresh.Circuit = answer.Circuit
	newer := conversationResponse(t, conversationRequest(h, seed, first.ID, fresh, child), 202)
	if newer.ID == first.ID {
		t.Fatal("expired run reused")
	}
	// Cursor history is bounded and ordered, including after the run expires.
	rows, err := h.Queries.ListBuildMessages(ctx, db.ListBuildMessagesParams{ConversationID: parseUUID(first.ID), PageSize: 2})
	if err != nil || len(rows) != 2 || rows[0].Sequence <= rows[1].Sequence {
		t.Fatalf("history %v %v", rows, err)
	}
	older, err := h.Queries.ListBuildMessages(ctx, db.ListBuildMessagesParams{ConversationID: parseUUID(first.ID), PageSize: 2, BeforeSequence: rows[1].Sequence})
	if err != nil || len(older) == 0 || older[0].Sequence >= rows[1].Sequence {
		t.Fatal("cursor repeated message")
	}
}

func TestParseCreationRouteAndReply(t *testing.T) {
	for _, raw := range []string{`{"kind":"brick"}`, `{"kind":"circuit"}`, `{"kind":"clarify","question":"外形还是电路？"}`} {
		if _, err := parseBuildRoute(raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{"kind":"hybrid"}`, `{"kind":"brick","question":"ignored"}`, `{"kind":"clarify"}`, `{"kind":"circuit","tool":"shell"}`, `{"kind":"brick"} {}`} {
		if _, err := parseBuildRoute(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := parseBuildDecision(`{"outcome":"reply","message":"这个门洞由两侧支撑。"}`); err != nil {
		t.Fatal(err)
	}
}

func TestBuildConversationDBAutoSwitchAndSourceRestoration(t *testing.T) {
	h, pool, seed := buildTestDB(t)
	ctx := context.Background()
	h.LLM = buildFakeLLM(func() buildPlanningDecision { return validBuildDecision() })
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	originalSession, err := h.Queries.GetBuildSessionForWorker(ctx, seed.ID)
	if err != nil {
		t.Fatal(err)
	}
	original, err := h.Queries.GetBuildCreationInWorkspace(ctx, db.GetBuildCreationInWorkspaceParams{ID: originalSession.CreationID, WorkspaceID: seed.WorkspaceID, CreatorUserID: seed.CreatorUserID})
	if err != nil {
		t.Fatal(err)
	}
	source, err := toBuildCreationResponse(original)
	if err != nil {
		t.Fatal(err)
	}
	var captured []string
	h.LLM = llm.New(llm.Config{BaseURL: "https://conversation-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(r.Body)
		captured = append(captured, string(raw))
		answer := `{"kind":"brick"}`
		if len(captured) == 2 {
			answer = string(mustBuildJSON(validBuildDecision()))
		}
		if len(captured) == 3 {
			answer = `{"kind":"circuit"}`
		}
		if len(captured) == 4 {
			answer = `{"outcome":"ready","project_id":"switch-light","title":"New light"}`
		}
		body := mustBuildJSON(map[string]any{"id": "test", "type": "message", "role": "assistant", "model": "test", "content": []map[string]string{{"type": "text", "text": answer}}, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}})
	req := createBuildSessionRequest{Prompt: "make the same robot blue", ClientRequestID: "00000000-0000-4000-8000-000000000401", Kind: "auto", SourceKind: "brick", SourceCreationID: uuidToString(original.ID), ExpectedContentHash: source.BuildPlan.ContentHash, ExpectedSessionID: uuidToString(seed.ID), ExpectedRevision: 1}
	edited := conversationResponse(t, conversationRequest(h, seed, uuidToString(seed.ID), req, ""), 202)
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	current, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(edited.ID))
	if err != nil || current.Status != "completed" || current.Kind != "brick" {
		t.Fatalf("auto brick %s %v", current.Status, err)
	}
	if len(captured) != 2 || !strings.Contains(captured[1], "parent_creation_id") || !strings.Contains(captured[1], uuidToString(original.ID)) {
		t.Fatal("selected source was lost after routing")
	}
	c := circuit.StarterCatalog()
	quantities := map[string]int{}
	for _, p := range c.Parts {
		quantities[p.ID] = p.Quantity
	}
	inv, err := h.Queries.SaveCircuitInventory(ctx, db.SaveCircuitInventoryParams{WorkspaceID: seed.WorkspaceID, ParentUserID: seed.CreatorUserID, KitID: c.KitID, CatalogVersion: c.Version, Quantities: mustBuildJSON(quantities)})
	if err != nil {
		t.Fatal(err)
	}
	req.ClientRequestID = "00000000-0000-4000-8000-000000000402"
	req.Prompt = "Now make a working lamp with my electronics"
	req.ExpectedSessionID = edited.ID
	req.Circuit = &circuitBuildPlan{KitID: c.KitID, CatalogVersion: c.Version, InventoryRevision: inv.Revision}
	switched := conversationResponse(t, conversationRequest(h, seed, uuidToString(seed.ID), req, ""), 202)
	// The circuit branch must not touch the brick availability path.
	if _, err = pool.Exec(ctx, "ALTER TABLE brick_inventory RENAME TO unavailable_brick_inventory"); err != nil {
		t.Fatal(err)
	}
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	final, err := h.Queries.GetBuildSessionForWorker(ctx, parseUUID(switched.ID))
	if err != nil || final.Status != "completed" || final.Kind != "circuit" || !final.CircuitCreationID.Valid {
		t.Fatalf("auto circuit %s %v", final.Status, err)
	}
	if len(captured) != 4 || !strings.Contains(captured[3], "make the same robot blue") {
		t.Fatal("conversation history missing after switching domains")
	}
}
