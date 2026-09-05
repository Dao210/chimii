package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/google/uuid"
)

func TestBuildProgressAndSummary(t *testing.T) {
	ctx := context.Background()
	id, child := uuid.NewString(), uuid.NewString()
	// Large immutable fields make accidental full-document summary responses visible.
	_, err := testPool.Exec(ctx, `INSERT INTO build_creation(id,workspace_id,creator_user_id,child_profile_id,session_id,title,prompt,archetype,recipe,build_plan,validation,ldraw_mpd)
 VALUES($1,$2,$3,$4,gen_random_uuid(),'Robot','robot','robot','{}','{}','{"step_count":3,"part_count":6}',repeat('x',100000))`, id, testWorkspaceID, testUserID, child)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM build_creation WHERE id=$1`, id) })
	write := func(step, rev int, complete bool, actor string, want int) buildProgressResponse {
		t.Helper()
		req := circuitTestRequest("PUT", id, map[string]any{"current_step": step, "expected_revision": rev, "completed": complete}, actor)
		w := httptest.NewRecorder()
		testHandler.UpdateBuildProgress(w, req)
		if w.Code != want {
			t.Fatalf("status %d want %d: %s", w.Code, want, w.Body.String())
		}
		var v buildProgressResponse
		if want == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
		}
		return v
	}
	write(1, 0, false, uuid.NewString(), 404)
	missingRevision := httptest.NewRecorder()
	testHandler.UpdateBuildProgress(missingRevision, circuitTestRequest("PUT", id, map[string]any{"current_step": 1}, child))
	if missingRevision.Code != 400 {
		t.Fatal("accepted progress without revision", missingRevision.Code)
	}
	first := write(1, 0, false, child, 200)
	if first.Revision != 1 || first.CompletedAt != nil {
		t.Fatal(first)
	}
	write(2, 0, false, child, 409)
	write(4, 1, false, child, 400)
	write(2, 1, true, child, 400)
	done := write(3, 1, true, "", 200)
	if done.Revision != 2 || done.CompletedAt == nil {
		t.Fatal(done)
	}
	w := httptest.NewRecorder()
	testHandler.GetBuildProgress(w, circuitTestRequest("GET", id, nil, child))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var loaded buildProgressResponse
	_ = json.Unmarshal(w.Body.Bytes(), &loaded)
	if loaded.CurrentStep != 3 || loaded.Revision != 2 {
		t.Fatal(loaded)
	}
	w = httptest.NewRecorder()
	testHandler.ListBuildCreationSummaries(w, circuitTestRequest("GET", "", nil, child))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var list struct {
		Creations []map[string]json.RawMessage `json:"creations"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Creations) != 1 || list.Creations[0]["build_plan"] != nil || list.Creations[0]["recipe"] != nil {
		t.Fatal(w.Body.String())
	}
	if w.Body.Len() > 1000 {
		t.Fatal("summary contains document payload")
	}
	var mpd string
	if err := testPool.QueryRow(ctx, `SELECT ldraw_mpd FROM build_creation WHERE id=$1`, id).Scan(&mpd); err != nil || len(mpd) != 100000 {
		t.Fatal("progress changed immutable content", err)
	}
	t.Logf("summary response bytes=%d for a creation with 100000 MPD bytes", w.Body.Len())
	if _, err := testPool.Exec(ctx, `UPDATE build_creation SET validation='{}' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	testHandler.GetBuildProgress(w, circuitTestRequest("GET", id, nil, child))
	if w.Code != 500 {
		t.Fatal("invalid saved validation produced successful progress", w.Code)
	}
	write(1, 2, false, child, 500)
}

func TestBuildWorkerDBDeletedWorkspaceRejectsCompletion(t *testing.T) {
	h, pool, s := buildTestDB(t)
	ctx := context.Background()
	job, err := h.Queries.ClaimBuildJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `DELETE FROM workspace WHERE id=$1`, s.WorkspaceID); err != nil {
		t.Fatal(err)
	}
	applied := false
	err = h.BuildWorker.finish(ctx, job, func(*db.Queries) error { applied = true; return nil })
	if err == nil || applied {
		t.Fatalf("deleted workspace accepted worker result: %v %v", applied, err)
	}
}
