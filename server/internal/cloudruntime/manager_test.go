package cloudruntime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chimii-ai/chimii/server/internal/runtimeconfig"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type managerQueryStub struct {
	runtimes []db.AgentRuntime
}

func (s managerQueryStub) ListOnlineCloudAgentRuntimes(context.Context) ([]db.AgentRuntime, error) {
	return s.runtimes, nil
}

type managerExecutorStub struct {
	result *RunResult
	err    error
}

func (s managerExecutorStub) Execute(ctx context.Context, input RunInput) (*RunResult, error) {
	if input.OnSession != nil {
		if err := input.OnSession(ctx, "crs_"+uuid.NewString()); err != nil {
			return nil, err
		}
	}
	return s.result, s.err
}

func (s managerExecutorStub) MarkSessionCancelled(context.Context, string) error {
	return nil
}

type managerCoordinatorStub struct {
	mu sync.Mutex

	claimed   *ClaimedRun
	started   int
	pinned    int
	complete  int
	failed    int
	cancelled int
	reason    string
	status    string
}

func (s *managerCoordinatorStub) ClaimCloudTask(context.Context, db.AgentRuntime) (*ClaimedRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimed == nil {
		return nil, nil
	}
	run := *s.claimed
	s.claimed = nil
	return &run, nil
}

func (s *managerCoordinatorStub) StartCloudTask(context.Context, ClaimedRun) error {
	s.mu.Lock()
	s.started++
	s.mu.Unlock()
	return nil
}

func (s *managerCoordinatorStub) PinCloudTaskSession(_ context.Context, _ ClaimedRun, sessionID, workDir string) error {
	if sessionID == "" || workDir == "" {
		return errors.New("session and workdir must be pinned together")
	}
	s.mu.Lock()
	s.pinned++
	s.mu.Unlock()
	return nil
}

func (s *managerCoordinatorStub) AppendCloudTaskMessages(_ context.Context, _ ClaimedRun, messages []TaskMessage) error {
	if len(messages) == 0 {
		return errors.New("empty task message batch")
	}
	return nil
}

func (s *managerCoordinatorStub) CompleteCloudTask(_ context.Context, _ ClaimedRun, result *RunResult, workDir string) error {
	if result == nil || workDir == "" {
		return errors.New("result and workdir are required")
	}
	s.mu.Lock()
	s.complete++
	s.mu.Unlock()
	return nil
}

func (s *managerCoordinatorStub) FailCloudTask(_ context.Context, _ ClaimedRun, _ error, classified ClassifiedError, _, _ string) error {
	s.mu.Lock()
	s.failed++
	s.reason = classified.Reason
	s.mu.Unlock()
	return nil
}

func (s *managerCoordinatorStub) CloudTaskStatus(context.Context, ClaimedRun) (string, error) {
	if s.status != "" {
		return s.status, nil
	}
	return "running", nil
}

func (s *managerCoordinatorStub) AcknowledgeCloudTaskCancelled(context.Context, ClaimedRun) error {
	s.mu.Lock()
	s.cancelled++
	s.mu.Unlock()
	return nil
}

func TestManagerExecuteRunPinsSessionAndCompletes(t *testing.T) {
	t.Parallel()
	run := validClaimedRun()
	coordinator := &managerCoordinatorStub{}
	manager := &Manager{
		config: runtimeconfig.Config{
			CloudEnabled: true,
			WorkRoot:     t.TempDir(),
		},
		coordinator: coordinator,
		executor: managerExecutorStub{result: &RunResult{
			Output: "done",
		}},
	}

	manager.executeRun(context.Background(), run)

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.started != 1 || coordinator.pinned != 1 || coordinator.complete != 1 || coordinator.failed != 0 {
		t.Fatalf("unexpected lifecycle counts: started=%d pinned=%d complete=%d failed=%d", coordinator.started, coordinator.pinned, coordinator.complete, coordinator.failed)
	}
}

func TestManagerExecuteRunClassifiesProviderFailure(t *testing.T) {
	t.Parallel()
	run := validClaimedRun()
	coordinator := &managerCoordinatorStub{}
	manager := &Manager{
		config: runtimeconfig.Config{
			CloudEnabled: true,
			WorkRoot:     t.TempDir(),
		},
		coordinator: coordinator,
		executor:    managerExecutorStub{err: errors.New("status 429: rate limit")},
	}

	manager.executeRun(context.Background(), run)

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.failed != 1 || coordinator.complete != 0 {
		t.Fatalf("unexpected terminal counts: complete=%d failed=%d", coordinator.complete, coordinator.failed)
	}
	if coordinator.reason != FailureRateLimited {
		t.Fatalf("failure reason = %q, want %q", coordinator.reason, FailureRateLimited)
	}
}

func TestManagerAcknowledgesCancelledCloudTask(t *testing.T) {
	t.Parallel()
	run := validClaimedRun()
	coordinator := &managerCoordinatorStub{status: "cancelled"}
	manager := &Manager{
		config: runtimeconfig.Config{
			CloudEnabled: true,
			WorkRoot:     t.TempDir(),
		},
		coordinator: coordinator,
		executor:    managerExecutorStub{err: context.Canceled},
	}

	manager.executeRun(context.Background(), run)

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.cancelled != 1 || coordinator.failed != 0 || coordinator.complete != 0 {
		t.Fatalf("unexpected cancellation lifecycle counts: acknowledged=%d failed=%d complete=%d", coordinator.cancelled, coordinator.failed, coordinator.complete)
	}
}

func TestManagerClaimOneOnlyUsesCloudRuntimeRows(t *testing.T) {
	t.Parallel()
	run := validClaimedRun()
	coordinator := &managerCoordinatorStub{claimed: &run}
	manager := &Manager{
		queries: managerQueryStub{runtimes: []db.AgentRuntime{{
			ID:            pgUUID(run.RuntimeID),
			RuntimeMode:   "cloud",
			ExecutionType: "cloud",
			Status:        "online",
		}}},
		coordinator: coordinator,
	}

	got, err := manager.claimOne(context.Background())
	if err != nil {
		t.Fatalf("claimOne: %v", err)
	}
	if got == nil || got.TaskID != run.TaskID {
		t.Fatalf("claimed run = %#v, want task %s", got, run.TaskID)
	}
}

func validClaimedRun() ClaimedRun {
	return ClaimedRun{
		TaskID:      uuid.NewString(),
		WorkspaceID: uuid.NewString(),
		RuntimeID:   uuid.NewString(),
		AgentID:     uuid.NewString(),
		Provider:    runtimeconfig.ProviderAnthropic,
		Model:       "claude-test",
		Prompt:      "hello",
		ContextType: "direct",
		ContextID:   uuid.NewString(),
	}
}

func pgUUID(raw string) pgtype.UUID {
	parsed := uuid.MustParse(raw)
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func TestIsTerminalTaskStatus(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"completed", "failed", "cancelled"} {
		if !isTerminalTaskStatus(status) {
			t.Errorf("%q should be terminal", status)
		}
	}
	if isTerminalTaskStatus("running") {
		t.Error("running should not be terminal")
	}
}

func TestRetryTerminalActionRecoversTransientFailure(t *testing.T) {
	t.Parallel()
	attempts := 0
	err := retryTerminalAction(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary database error")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRetryTerminalActionStopsAtContextDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	want := errors.New("persistent database error")
	if got := retryTerminalAction(ctx, func() error { return want }); !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}
