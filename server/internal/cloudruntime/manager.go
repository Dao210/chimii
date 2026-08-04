package cloudruntime

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/chimii-ai/chimii/server/internal/runtimeconfig"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultPollInterval = time.Second

type ClaimedRun struct {
	TaskID         string
	WorkspaceID    string
	RuntimeID      string
	AgentID        string
	IssueID        string
	Provider       string
	Model          string
	Prompt         string
	SystemPrompt   string
	PriorSessionID string
	ContextType    string
	ContextID      string
}

type Coordinator interface {
	ClaimCloudTask(context.Context, db.AgentRuntime) (*ClaimedRun, error)
	StartCloudTask(context.Context, ClaimedRun) error
	PinCloudTaskSession(context.Context, ClaimedRun, string, string) error
	AppendCloudTaskMessages(context.Context, ClaimedRun, []TaskMessage) error
	CompleteCloudTask(context.Context, ClaimedRun, *RunResult, string) error
	FailCloudTask(context.Context, ClaimedRun, error, ClassifiedError, string, string) error
	CloudTaskStatus(context.Context, ClaimedRun) (string, error)
	AcknowledgeCloudTaskCancelled(context.Context, ClaimedRun) error
}

type runtimeQueries interface {
	ListOnlineCloudAgentRuntimes(context.Context) ([]db.AgentRuntime, error)
}

type runExecutor interface {
	Execute(context.Context, RunInput) (*RunResult, error)
	MarkSessionCancelled(context.Context, string) error
}

type Manager struct {
	config       runtimeconfig.Config
	queries      runtimeQueries
	coordinator  Coordinator
	executor     runExecutor
	pollInterval time.Duration

	workers chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	next    int
}

func NewManager(config runtimeconfig.Config, queries *db.Queries, coordinator Coordinator, executor *Executor) *Manager {
	return &Manager{
		config:       config,
		queries:      queries,
		coordinator:  coordinator,
		executor:     executor,
		pollInterval: defaultPollInterval,
		workers:      make(chan struct{}, config.MaxConcurrentTasks),
	}
}

func (m *Manager) Run(ctx context.Context) {
	if m == nil || !m.config.CloudEnabled {
		return
	}
	slog.Info("cloud runtime worker started", "max_concurrent_tasks", cap(m.workers))
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	for {
		m.dispatchAvailable(ctx)
		select {
		case <-ctx.Done():
			m.wg.Wait()
			slog.Info("cloud runtime worker stopped")
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) dispatchAvailable(ctx context.Context) {
	for {
		select {
		case m.workers <- struct{}{}:
		case <-ctx.Done():
			return
		default:
			return
		}
		run, err := m.claimOne(ctx)
		if err != nil {
			<-m.workers
			slog.Warn("cloud runtime task claim failed", "error", err)
			return
		}
		if run == nil {
			<-m.workers
			return
		}
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			defer func() { <-m.workers }()
			m.executeRun(ctx, *run)
		}()
	}
}

func (m *Manager) claimOne(ctx context.Context) (*ClaimedRun, error) {
	runtimes, err := m.queries.ListOnlineCloudAgentRuntimes(ctx)
	if err != nil || len(runtimes) == 0 {
		return nil, err
	}
	m.mu.Lock()
	start := m.next % len(runtimes)
	m.next = (start + 1) % len(runtimes)
	m.mu.Unlock()
	for offset := range runtimes {
		runtime := runtimes[(start+offset)%len(runtimes)]
		run, err := m.coordinator.ClaimCloudTask(ctx, runtime)
		if err != nil {
			return nil, err
		}
		if run != nil {
			return run, nil
		}
	}
	return nil, nil
}

func (m *Manager) executeRun(parent context.Context, run ClaimedRun) {
	workDir, err := PrepareWorkDir(m.config.WorkRoot, run.WorkspaceID, run.TaskID)
	if err != nil {
		m.failWithFreshContext(run, err, "", "")
		return
	}
	if err := m.coordinator.StartCloudTask(parent, run); err != nil {
		m.failWithFreshContext(run, err, "", workDir)
		return
	}

	runCtx, cancel := context.WithCancel(parent)
	defer cancel()
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		m.watchCancellation(runCtx, run, cancel)
	}()

	sessionID := ""
	messageSeq := 0
	result, runErr := m.executor.Execute(runCtx, RunInput{
		SessionSpec: SessionSpec{
			WorkspaceID: parsePGUUID(run.WorkspaceID),
			RuntimeID:   parsePGUUID(run.RuntimeID),
			AgentID:     parsePGUUID(run.AgentID),
			Provider:    run.Provider,
			ContextType: run.ContextType,
			ContextID:   run.ContextID,
		},
		ResumeID:     run.PriorSessionID,
		Prompt:       run.Prompt,
		SystemPrompt: run.SystemPrompt,
		Model:        run.Model,
		WorkDir:      workDir,
		OnSession: func(ctx context.Context, id string) error {
			sessionID = id
			return m.coordinator.PinCloudTaskSession(ctx, run, id, workDir)
		},
		OnMessage: func(ctx context.Context, _ string, message sdktypes.Message) error {
			messages := TaskMessagesFromSDK(message)
			for i := range messages {
				messageSeq++
				messages[i].Seq = messageSeq
			}
			if len(messages) == 0 {
				return nil
			}
			return m.coordinator.AppendCloudTaskMessages(ctx, run, messages)
		},
	})
	cancel()
	<-watchDone
	if runErr != nil {
		terminalCtx, terminalCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer terminalCancel()
		if status, statusErr := m.coordinator.CloudTaskStatus(terminalCtx, run); statusErr == nil && isTerminalTaskStatus(status) {
			if status == "cancelled" {
				if sessionID != "" {
					if err := m.executor.MarkSessionCancelled(terminalCtx, sessionID); err != nil {
						slog.Warn("cloud runtime session cancellation status update failed", "task_id", run.TaskID, "session_id", sessionID, "error", err)
					}
				}
				if err := retryTerminalAction(terminalCtx, func() error {
					return m.coordinator.AcknowledgeCloudTaskCancelled(terminalCtx, run)
				}); err != nil {
					slog.Warn("cloud runtime cancellation acknowledgement failed", "task_id", run.TaskID, "error", err)
				}
			}
			return
		}
		classified := ClassifyError(runErr)
		if err := retryTerminalAction(terminalCtx, func() error {
			return m.coordinator.FailCloudTask(terminalCtx, run, runErr, classified, sessionID, workDir)
		}); err != nil {
			slog.Error("cloud runtime task failure finalization failed", "task_id", run.TaskID, "error", err)
		}
		return
	}
	terminalCtx, terminalCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer terminalCancel()
	if err := retryTerminalAction(terminalCtx, func() error {
		return m.coordinator.CompleteCloudTask(terminalCtx, run, result, workDir)
	}); err != nil {
		slog.Error("cloud runtime task completion failed", "task_id", run.TaskID, "error", err)
	}
}

func (m *Manager) failWithFreshContext(run ClaimedRun, runErr error, sessionID, workDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	classified := ClassifyError(runErr)
	if err := retryTerminalAction(ctx, func() error {
		return m.coordinator.FailCloudTask(ctx, run, runErr, classified, sessionID, workDir)
	}); err != nil {
		slog.Error("cloud runtime task failure finalization failed", "task_id", run.TaskID, "error", err)
	}
}

func retryTerminalAction(ctx context.Context, action func() error) error {
	const retryDelay = 200 * time.Millisecond
	var lastErr error
	for {
		if err := action(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lastErr
		case <-timer.C:
		}
	}
}

func (m *Manager) watchCancellation(ctx context.Context, run ClaimedRun, cancel context.CancelFunc) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status, err := m.coordinator.CloudTaskStatus(ctx, run)
			if err == nil && isTerminalTaskStatus(status) {
				cancel()
				return
			}
		}
	}
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func parsePGUUID(raw string) pgtype.UUID {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}
