package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
	"github.com/jackc/pgx/v5/pgtype"
)

// Simulate the production failure: headers arrive, but the complete JSON body
// takes longer than the old deadline. Virtual time keeps this test fast.
func TestBuildPlannerSlowResponseAndTimeout(t *testing.T) {
	for _, tc := range []struct {
		name        string
		delay       time.Duration
		wantTimeout bool
	}{
		{"slow_complete_response", 25 * time.Second, false},
		{"planner_deadline", 65 * time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := &Handler{LLM: llm.New(llm.Config{BaseURL: "https://build-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
					reader, writer := io.Pipe()
					go func() {
						select {
						case <-r.Context().Done():
							_ = writer.CloseWithError(r.Context().Err())
						case <-time.After(tc.delay):
							decision := string(mustBuildJSON(validBuildDecision()))
							_, _ = writer.Write(mustBuildJSON(map[string]any{"content": []map[string]string{{"type": "text", "text": decision}}}))
							_ = writer.Close()
						}
					}()
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: reader, Request: r}, nil
				})}})}
				started := time.Now()
				decision, err := h.planBuildRecipe(context.Background(), "a robot", map[string]string{}, nil, 1, buildstudio.UnlimitedInventory(), nil)
				if tc.wantTimeout {
					code, _ := buildstudio.BuildErrorCode(err)
					if code != "BUILD_PLANNER_TIMEOUT" || !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("expected a classified planner timeout, got %v", err)
					}
					if time.Since(started) != 60*time.Second {
						t.Fatalf("timeout after %s", time.Since(started))
					}
				} else if err != nil || decision.Outcome != "ready" {
					t.Fatalf("slow valid response failed: outcome=%s err=%v", decision.Outcome, err)
				}
			})
		})
	}
}

func TestBuildPlannerDoesNotClassifyParentCancellationAsProviderTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &Handler{LLM: llm.New(llm.Config{BaseURL: "https://build-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}})}
	_, err := h.planBuildRecipe(ctx, "a robot", map[string]string{}, nil, 1, buildstudio.UnlimitedInventory(), nil)
	if _, classified := buildstudio.BuildErrorCode(err); classified || !errors.Is(err, context.Canceled) {
		t.Fatalf("parent cancellation was changed: %v", err)
	}
}

type buildErrorBody struct{ err error }

func (b buildErrorBody) Read([]byte) (int, error) { return 0, b.err }
func (b buildErrorBody) Close() error             { return nil }

func TestBuildWorkerDBPlannerFailurePolicy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cause    error
		attempts int
		code     string
	}{
		{"timeout_is_terminal", context.DeadlineExceeded, 1, "BUILD_PLANNER_TIMEOUT"},
		{"interrupted_body_can_retry", io.ErrUnexpectedEOF, 3, "BUILD_GENERATION_FAILED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, pool, session := buildTestDB(t)
			ctx := context.Background()
			calls := 0
			h.LLM = llm.New(llm.Config{BaseURL: "https://build-test.invalid", HTTPClient: &http.Client{Transport: buildTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: buildErrorBody{tc.cause}, Request: r}, nil
			})}})
			for i := 0; i < tc.attempts; i++ {
				if worked, err := h.BuildWorker.ProcessNext(ctx); err != nil || !worked {
					t.Fatalf("attempt %d: worked=%v error=%v", i+1, worked, err)
				}
				if _, err := pool.Exec(ctx, "UPDATE build_job SET available_at=now() WHERE session_id=$1", session.ID); err != nil {
					t.Fatal(err)
				}
			}
			current, err := h.Queries.GetBuildSessionForWorker(ctx, session.ID)
			if err != nil || current.Status != "failed" || current.Error.String != tc.code {
				t.Fatalf("session: status=%s error=%s read_error=%v", current.Status, current.Error.String, err)
			}
			if toBuildSessionResponse(current).Error != tc.code {
				t.Fatal("public response lost the actionable failure code")
			}
			if worked, err := h.BuildWorker.ProcessNext(ctx); err != nil || worked || calls != tc.attempts {
				t.Fatalf("terminal job was repeated: worked=%v calls=%d error=%v", worked, calls, err)
			}
			var status, cause string
			var attempts, outcomes, creations int
			if err := pool.QueryRow(ctx, "SELECT status, attempts, last_error FROM build_job WHERE session_id=$1", session.ID).Scan(&status, &attempts, &cause); err != nil {
				t.Fatal(err)
			}
			if status != "failed" || attempts != tc.attempts || cause == "" {
				t.Fatalf("job: status=%s attempts=%d cause=%s", status, attempts, cause)
			}
			if err := pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM build_message WHERE session_id=$1 AND kind='error'), (SELECT count(*) FROM build_creation WHERE session_id=$1)", session.ID).Scan(&outcomes, &creations); err != nil {
				t.Fatal(err)
			}
			if outcomes != 1 || creations != 0 {
				t.Fatalf("terminal outcome count=%d creations=%d", outcomes, creations)
			}
		})
	}
}

func TestBuildWorkerDBLeaseCoversPlanningAndCompletion(t *testing.T) {
	h, _, _ := buildTestDB(t)
	job, err := h.Queries.ClaimBuildJob(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	lease := job.LeasedUntil.Time.Sub(job.UpdatedAt.Time)
	if buildWorkerTimeout < buildIntentTimeout+buildRecipeTimeout+10*time.Second || lease < buildWorkerTimeout+20*time.Second {
		t.Fatalf("lease %s does not cover routing, recipe, compilation and final writes", lease)
	}
}

func TestBuildTimeoutResponseDoesNotExposeProviderError(t *testing.T) {
	for _, value := range []string{"BUILD_PLANNER_TIMEOUT", "provider timeout with private response text"} {
		response := toBuildSessionResponse(db.BuildSession{Status: "failed", Error: pgtype.Text{String: value, Valid: true}})
		want := "BUILD_GENERATION_FAILED"
		if value == "BUILD_PLANNER_TIMEOUT" {
			want = value
		}
		if response.Error != want {
			t.Fatalf("unexpected public error: %q", response.Error)
		}
	}
}
