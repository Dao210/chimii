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

func circuitTestRequest(method, id string, body any, child string) *http.Request {
	r := newRequest(method, "/api/circuit/creations/"+id, body)
	r = r.WithContext(middleware.SetMemberContext(r.Context(), testWorkspaceID, db.Member{}))
	if id != "" {
		r = withURLParam(r, "id", id)
	}
	if child != "" {
		r.Header.Set("X-Actor-Source", "child_session")
		r.Header.Set("X-Child-Profile-ID", child)
	}
	return r
}

func circuitTestInput() circuitCreateRequest {
	c := circuit.StarterCatalog()
	return circuitCreateRequest{ClientRequestID: uuid.NewString(), KitID: c.KitID, CatalogVersion: c.Version, ProjectID: "fm-radio", Locale: "zh-Hans", Inventory: c.Inventory()}
}

func TestCircuitAssemblyReferenceCatalogBoundary(t *testing.T) {
	w := httptest.NewRecorder()
	testHandler.ListCircuitKits(w, circuitTestRequest(http.MethodGet, "", nil, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("kits status=%d", w.Code)
	}
	var response struct {
		Kits []struct {
			KitID string `json:"kit_id"`
		} `json:"kits"`
		References []circuit.AssemblyReference `json:"assembly_references"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Kits) != 2 || len(response.References) != 1 {
		t.Fatalf("unexpected kit/reference counts: %d/%d", len(response.Kits), len(response.References))
	}
	ref := response.References[0]
	if err := circuit.ValidateAssemblyReference(ref); err != nil || len(ref.ContentHash) != 64 {
		t.Fatalf("invalid reference: %v", err)
	}
	for _, kit := range response.Kits {
		if kit.KitID == ref.KitID {
			t.Fatal("research reference exposed in executable kit picker")
		}
	}
	input := circuitTestInput()
	input.KitID, input.CatalogVersion, input.ProjectID = ref.KitID, ref.Version, ref.ID
	circuitTestCreate(t, input, "", http.StatusConflict)
}

func circuitTestCreate(t *testing.T, input circuitCreateRequest, child string, status int) circuitCreationResponse {
	t.Helper()
	w := httptest.NewRecorder()
	testHandler.CreateCircuitCreation(w, circuitTestRequest(http.MethodPost, "", input, child))
	if w.Code != status {
		t.Fatalf("create status=%d: %s", w.Code, w.Body.String())
	}
	var got circuitCreationResponse
	if status == 200 || status == 201 {
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := testPool.Exec(context.Background(), `DELETE FROM circuit_creation WHERE id=$1`, got.ID); err != nil {
				t.Error(err)
			}
		})
	}
	return got
}

func TestCircuitCreationIdempotencyAndSnapshot(t *testing.T) {
	input := circuitTestInput()
	first := circuitTestCreate(t, input, "", 201)
	replay := circuitTestCreate(t, input, "", 200)
	if first.ID != replay.ID || string(first.Document) != string(replay.Document) {
		t.Fatal("retry did not preserve the immutable creation")
	}
	input.ProjectID = "switch-light"
	circuitTestCreate(t, input, "", 409)
	var doc circuit.Document
	if err := json.Unmarshal(first.Document, &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.Validation.Passed || doc.Validation.PhysicalVerification != "not_tested" || len(doc.Project.Placements) != 23 || doc.ContentHash == "" {
		t.Fatalf("invalid snapshot: %+v", doc.Validation)
	}
}

func TestCircuitCreationRejectsMissingPartsAndUnavailableAI(t *testing.T) {
	input := circuitTestInput()
	input.Inventory["FM"] = 0
	circuitTestCreate(t, input, "", 422)
	input = circuitTestInput()
	input.ProjectID, input.Prompt = "", "我想听广播"
	circuitTestCreate(t, input, "", 503)
	var count int
	if err := testPool.QueryRow(context.Background(), `SELECT count(*) FROM circuit_creation WHERE client_request_id=$1`, input.ClientRequestID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed generation persisted a creation: count=%d, err=%v", count, err)
	}
}

func TestCircuitCreationUsesConfiguredPlanner(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		status       int
	}{
		{"radio", `{"project_id":"fm-radio","title":"我的收音机"}`, 201},
		{"unsupported", `{"project_id":"unsupported","title":"蓝牙收音机"}`, 422},
		{"invented wiring", `{"project_id":"fm-radio","title":"收音机","wires":[]}`, 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v1/messages" {
					t.Errorf("unexpected provider path %s", r.URL.Path)
				}
				writeJSON(w, 200, map[string]any{"id": "test-message", "type": "message", "role": "assistant", "model": "test-model", "content": []map[string]string{{"type": "text", "text": tc.output}}, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
			}))
			defer provider.Close()
			previous := testHandler.LLM
			testHandler.LLM = llm.New(llm.Config{BaseURL: provider.URL, DefaultModel: "test-model"})
			defer func() { testHandler.LLM = previous }()
			input := circuitTestInput()
			input.ProjectID, input.Prompt = "", "我想听广播"
			created := circuitTestCreate(t, input, "", tc.status)
			if tc.status == 201 {
				replay := circuitTestCreate(t, input, "", 200)
				if replay.ID != created.ID {
					t.Fatal("AI retry generated a new document")
				}
				var doc circuit.Document
				if err := json.Unmarshal(created.Document, &doc); err != nil {
					t.Fatal(err)
				}
				if doc.Planner != "llm-intent-v1" || doc.Project.ID != "fm-radio" {
					t.Fatal("planner provenance was lost")
				}
			}
			if calls != 1 {
				t.Fatalf("provider calls=%d", calls)
			}
		})
	}
}

func TestCircuitActorIsolation(t *testing.T) {
	childA, childB := uuid.NewString(), uuid.NewString()
	input := circuitTestInput()
	owned := circuitTestCreate(t, input, childA, 201)
	parent := circuitTestCreate(t, input, "", 201)
	if owned.ID == parent.ID {
		t.Fatal("idempotency key crossed actor scope")
	}
	for name, child := range map[string]string{"owner": childA, "sibling": childB, "parent": ""} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			testHandler.GetCircuitCreation(w, circuitTestRequest(http.MethodGet, owned.ID, nil, child))
			want := 404
			if child == childA {
				want = 200
			}
			if w.Code != want {
				t.Fatalf("read status=%d: %s", w.Code, w.Body.String())
			}
			w = httptest.NewRecorder()
			testHandler.ListCircuitCreations(w, circuitTestRequest(http.MethodGet, "", nil, child))
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			var list struct {
				Creations []struct {
					ID string `json:"id"`
				} `json:"creations"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
				t.Fatal(err)
			}
			for _, item := range list.Creations {
				if item.ID == owned.ID && child != childA {
					t.Fatal("list exposed another actor's creation")
				}
			}
		})
	}
	for _, scope := range []string{"workspace", "user"} {
		r := circuitTestRequest(http.MethodGet, owned.ID, nil, childA)
		if scope == "workspace" {
			r = r.WithContext(middleware.SetMemberContext(r.Context(), uuid.NewString(), db.Member{}))
		} else {
			r.Header.Set("X-User-ID", uuid.NewString())
		}
		w := httptest.NewRecorder()
		testHandler.GetCircuitCreation(w, r)
		if w.Code != 404 {
			t.Fatalf("%s scope leaked: %d", scope, w.Code)
		}
	}
	w := httptest.NewRecorder()
	testHandler.UpdateCircuitProgress(w, circuitTestRequest(http.MethodPut, owned.ID, map[string]any{"current_step": 1, "observation": "not_tried", "expected_revision": 0}, childB))
	if w.Code != 404 {
		t.Fatalf("sibling progress status=%d", w.Code)
	}
}

func TestCircuitProgressRevisionAndObservation(t *testing.T) {
	created := circuitTestCreate(t, circuitTestInput(), "", 201)
	update := func(step, revision int, observation string, status int) circuitCreationResponse {
		t.Helper()
		w := httptest.NewRecorder()
		testHandler.UpdateCircuitProgress(w, circuitTestRequest(http.MethodPut, created.ID, map[string]any{"current_step": step, "expected_revision": revision, "observation": observation}, ""))
		if w.Code != status {
			t.Fatalf("progress status=%d: %s", w.Code, w.Body.String())
		}
		var got circuitCreationResponse
		if status == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
		}
		return got
	}
	update(1, 0, "worked", 400)
	update(100, 0, "not_tried", 400)
	update(1, 0, "not_tried", 200)
	update(2, 0, "not_tried", 409)
	done := update(22, 1, "worked", 200)
	if done.ProgressRevision != 2 || done.Observation != "worked" || string(done.Document) != string(created.Document) {
		t.Fatal("progress did not preserve the original verification evidence")
	}
}
