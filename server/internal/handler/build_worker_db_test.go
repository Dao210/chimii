package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/middleware"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type buildTestTransport func(*http.Request) (*http.Response, error)

func (f buildTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func buildTestDB(t *testing.T) (*Handler, *pgxpool.Pool, db.BuildSession) {
	t.Helper()
	url := os.Getenv("CHIMII_BUILD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set CHIMII_BUILD_TEST_DATABASE_URL to an isolated PostgreSQL instance")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("build_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	for _, n := range []string{"234", "235", "236", "237", "246", "247", "248", "257", "258", "259", "260", "261", "262", "263", "264", "268", "269", "270", "271", "272", "273", "274", "275", "276", "277", "278", "279", "280", "281", "282", "283", "284", "285", "289", "290", "291", "292", "293", "294", "295"} {
		paths, _ := filepath.Glob("../../migrations/" + n + "_*.up.sql")
		for _, p := range paths {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, string(raw)); err != nil {
				t.Fatalf("%s: %v", p, err)
			}
		}
	}
	if _, err = pool.Exec(ctx, "CREATE TABLE workspace (id UUID); CREATE TABLE member (id UUID DEFAULT gen_random_uuid(), workspace_id UUID, user_id UUID)"); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	h := &Handler{Queries: q, TxStarter: pool}
	h.BuildWorker = NewBuildWorker(h)
	var ws, user, request pgtype.UUID
	if err = pool.QueryRow(ctx, "INSERT INTO member(workspace_id,user_id) VALUES(gen_random_uuid(),gen_random_uuid()) RETURNING workspace_id,user_id").Scan(&ws, &user); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO workspace(id) VALUES($1)", ws); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, "SELECT gen_random_uuid()").Scan(&request); err != nil {
		t.Fatal(err)
	}
	s, err := q.CreateBuildSession(ctx, db.CreateBuildSessionParams{WorkspaceID: ws, CreatorUserID: user, ClientRequestID: request, Prompt: "a robot", Status: "queued", Answers: []byte(`{}`), InventorySnapshot: []byte(`{"configured":false,"catalog_version":"","items":[]}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = q.EnqueueBuildJob(ctx, db.EnqueueBuildJobParams{WorkspaceID: ws, SessionID: s.ID}); err != nil {
		t.Fatal(err)
	}
	return h, pool, s
}
func buildFakeLLM(decide func() buildPlanningDecision) *llm.Client {
	return llm.New(llm.Config{BaseURL: "https://build-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
		raw, _ := json.Marshal(decide())
		body, _ := json.Marshal(map[string]any{"id": "test", "type": "message", "role": "assistant", "model": "test", "content": []map[string]string{{"type": "text", "text": string(raw)}}, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: r}, nil
	})}})
}
func TestBuildWorkerDBClarifyResumeAndNoDuplicateCreation(t *testing.T) {
	h, pool, s := buildTestDB(t)
	ctx := context.Background()
	var calls atomic.Int32
	h.LLM = buildFakeLLM(func() buildPlanningDecision {
		d := validBuildDecision()
		if calls.Add(1) == 1 {
			d.Outcome = "clarify"
			d.Recipe.Modules = nil
			d.Question = &buildPlanningQuestion{Prompt: "Which robot?", Choices: []buildstudio.QuestionChoice{{ID: "standing", Label: "A standing robot"}}}
		}
		return d
	})
	if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err := h.Queries.GetBuildSessionForWorker(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != "clarifying" {
		t.Fatalf("status %s", s.Status)
	}
	var q buildstudio.ClarifyingQuestion
	_ = json.Unmarshal(s.Question, &q)

	var submissions sync.WaitGroup
	responses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		submissions.Add(1)
		go func() {
			defer submissions.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/build/sessions/"+uuidToString(s.ID)+"/answers", strings.NewReader(string(mustBuildJSON(submitBuildAnswersRequest{Revision: s.Revision, Answers: map[string]string{q.ID: "standing"}}))))
			req.Header.Set("X-User-ID", uuidToString(s.CreatorUserID))
			req = withURLParam(req, "id", uuidToString(s.ID))
			req = req.WithContext(middleware.SetMemberContext(req.Context(), uuidToString(s.WorkspaceID), db.Member{}))
			response := httptest.NewRecorder()
			h.SubmitBuildAnswers(response, req)
			responses <- response.Code
		}()
	}
	submissions.Wait()
	close(responses)
	for code := range responses {
		if code != http.StatusOK {
			t.Fatalf("duplicate submission returned %d", code)
		}
	}
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err = h.Queries.GetBuildSessionForWorker(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != "completed" || calls.Load() != 2 {
		t.Fatalf("status %s calls %d", s.Status, calls.Load())
	}
	if worked, err := h.BuildWorker.ProcessNext(ctx); err != nil || worked {
		t.Fatalf("duplicate work %v %v", worked, err)
	}
	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM build_creation").Scan(&count); err != nil || count != 1 {
		t.Fatalf("creations %d %v", count, err)
	}
}
func TestBuildWorkerDBExpiredLeaseCannotFailOrFinishNewJob(t *testing.T) {
	h, pool, s := buildTestDB(t)
	ctx := context.Background()
	old, err := h.Queries.ClaimBuildJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "UPDATE build_job SET leased_until = now()-interval '1 second' WHERE id=$1", old.ID); err != nil {
		t.Fatal(err)
	}
	next, err := h.Queries.ClaimBuildJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.Queries.MarkBuildSessionGenerating(ctx, db.MarkBuildSessionGeneratingParams{ID: s.ID, Revision: s.Revision, LeaseToken: next.LeaseToken}); err != nil {
		t.Fatal(err)
	}
	called := false
	if err = h.BuildWorker.finish(ctx, old, func(*db.Queries) error { called = true; return nil }); err != nil || called {
		t.Fatalf("stale worker completion: applied=%v err=%v", called, err)
	}
	if err = h.BuildWorker.failPermanently(ctx, old, "wrong", errors.New("old failure")); err != nil {
		t.Fatal(err)
	}
	if err = h.BuildWorker.retry(ctx, old, errors.New("old retry")); err != nil {
		t.Fatal(err)
	}
	current, err := h.Queries.GetBuildSessionForWorker(ctx, s.ID)
	if err != nil || current.Status != "generating" {
		t.Fatalf("stale failure wrote session: %s %v", current.Status, err)
	}
}
func TestBuildWorkerDBSavedRecipeSkipsLLMOnResume(t *testing.T) {
	h, pool, s := buildTestDB(t)
	ctx := context.Background()
	var calls atomic.Int32
	h.LLM = buildFakeLLM(func() buildPlanningDecision { calls.Add(1); return validBuildDecision() })
	job, err := h.Queries.ClaimBuildJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.Queries.MarkBuildSessionGenerating(ctx, db.MarkBuildSessionGeneratingParams{ID: s.ID, Revision: s.Revision, LeaseToken: job.LeaseToken}); err != nil {
		t.Fatal(err)
	}
	recipe := buildstudio.ExampleRecipe("robot")
	recipe.Metadata["module_library_version"] = buildstudio.ModuleLibraryVersion
	if _, err = h.Queries.SaveBuildSessionRecipe(ctx, db.SaveBuildSessionRecipeParams{ID: s.ID, Revision: s.Revision, LeaseToken: job.LeaseToken, Recipe: mustBuildJSON(recipe)}); err != nil {
		t.Fatal(err)
	}
	if _, err = h.Queries.RetryBuildJob(ctx, db.RetryBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken, AvailableAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}); err != nil {
		t.Fatal(err)
	}
	// Use the database clock: Docker's clock may lag the host clock.
	if _, err = pool.Exec(ctx, "UPDATE build_job SET available_at = now() WHERE id=$1", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = h.BuildWorker.ProcessNext(ctx); err != nil {
		t.Fatal(err)
	}
	s, err = h.Queries.GetBuildSessionForWorker(ctx, s.ID)
	if err != nil || s.Status != "completed" || calls.Load() != 0 {
		t.Fatalf("resume status %s calls %d err %v", s.Status, calls.Load(), err)
	}
}

func TestBuildWorkerDBCancellationFencesActiveWorker(t *testing.T) {
	h, _, s := buildTestDB(t)
	ctx := context.Background()
	job, err := h.Queries.ClaimBuildJob(ctx)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/build/sessions/"+uuidToString(s.ID)+"/cancel", strings.NewReader(`{"revision":1}`))
	req.Header.Set("X-User-ID", uuidToString(s.CreatorUserID))
	req = withURLParam(req, "id", uuidToString(s.ID))
	req = req.WithContext(middleware.SetMemberContext(req.Context(), uuidToString(s.WorkspaceID), db.Member{}))
	response := httptest.NewRecorder()
	h.CancelBuildSession(response, req)
	if response.Code != 200 {
		t.Fatalf("cancel: %d %s", response.Code, response.Body.String())
	}
	wrote := false
	if err = h.BuildWorker.finish(ctx, job, func(*db.Queries) error { wrote = true; return nil }); err != nil || wrote {
		t.Fatalf("cancelled worker wrote %v %v", wrote, err)
	}
	current, err := h.Queries.GetBuildSessionForWorker(ctx, s.ID)
	if err != nil || current.Error.String != "BUILD_CANCELLED" {
		t.Fatalf("cancel state: %#v %v", current, err)
	}
}

func TestBuildWorkerDBClearAndUnsupportedIdeasUseOneCall(t *testing.T) {
	for _, outcome := range []string{"ready", "unsupported"} {
		t.Run(outcome, func(t *testing.T) {
			h, pool, s := buildTestDB(t)
			ctx := context.Background()
			var calls atomic.Int32
			h.LLM = buildFakeLLM(func() buildPlanningDecision {
				calls.Add(1)
				if outcome == "unsupported" {
					return buildPlanningDecision{Outcome: outcome, Message: "This subject is not supported."}
				}
				return validBuildDecision()
			})
			if _, err := h.BuildWorker.ProcessNext(ctx); err != nil {
				t.Fatal(err)
			}
			current, err := h.Queries.GetBuildSessionForWorker(ctx, s.ID)
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
			var count int
			if err = pool.QueryRow(ctx, "SELECT count(*) FROM build_creation").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if outcome == "ready" {
				if current.Status != "completed" || count != 1 {
					t.Fatalf("clear idea %s count %d", current.Status, count)
				}
			} else {
				if current.Status != "failed" || current.Error.String != buildstudio.BuildErrorUnsupported || count != 0 {
					t.Fatalf("unsupported idea %s count %d", current.Status, count)
				}
			}
		})
	}
}
