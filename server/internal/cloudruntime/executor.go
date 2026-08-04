package cloudruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chimii-ai/chimii/server/internal/runtimeconfig"
	sdkagent "github.com/codeany-ai/open-agent-sdk-go/agent"
	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
)

var (
	ErrProviderDisabled   = errors.New("cloud runtime provider is disabled")
	ErrModelNotConfigured = errors.New("cloud runtime model is not configured")
)

type RunInput struct {
	SessionSpec  SessionSpec
	ResumeID     string
	Prompt       string
	SystemPrompt string
	Model        string
	WorkDir      string
	OnSession    func(context.Context, string) error
	OnMessage    func(context.Context, string, sdktypes.Message) error
}

type RunResult struct {
	Output    string
	SessionID string
	Model     string
	Usage     sdktypes.Usage
	CostUSD   float64
	Turns     int
	Duration  time.Duration
}

type sdkSession interface {
	Init(context.Context) error
	Prompt(context.Context, string) (*sdkagent.QueryResult, error)
	Close()
}

type agentFactory func(sdkagent.Options) sdkSession

type Executor struct {
	config   runtimeconfig.Config
	store    *SessionStore
	newAgent agentFactory
}

func NewExecutor(config runtimeconfig.Config, store *SessionStore) *Executor {
	return &Executor{
		config: config,
		store:  store,
		newAgent: func(options sdkagent.Options) sdkSession {
			return sdkagent.New(options)
		},
	}
}

func (e *Executor) Execute(ctx context.Context, input RunInput) (*RunResult, error) {
	provider := strings.ToLower(strings.TrimSpace(input.SessionSpec.Provider))
	providerConfig, ok := e.config.CloudProviders[provider]
	if !e.config.CloudEnabled || !ok {
		return nil, ErrProviderDisabled
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = strings.TrimSpace(providerConfig.DefaultModel)
	}
	if model == "" || model != strings.TrimSpace(providerConfig.DefaultModel) {
		return nil, ErrModelNotConfigured
	}
	input.SessionSpec.Provider = provider
	input.SessionSpec.Model = model
	session, err := e.store.Open(ctx, input.ResumeID, input.SessionSpec)
	if err != nil {
		return nil, err
	}
	if input.OnSession != nil {
		if err := input.OnSession(ctx, session.Record.ID); err != nil {
			return nil, fmt.Errorf("pin cloud runtime session: %w", err)
		}
	}

	taskCtx, cancel := context.WithTimeout(ctx, e.config.TaskTimeout)
	defer cancel()
	a := e.newAgent(sdkagent.Options{
		APIKey:              providerConfig.APIKey,
		BaseURL:             providerConfig.BaseURL,
		Provider:            provider,
		Model:               model,
		CWD:                 input.WorkDir,
		SystemPrompt:        input.SystemPrompt,
		MaxTurns:            1,
		DisableDefaultTools: true,
		InitialMessages:     session.Messages,
		OnMessage: func(messageCtx context.Context, message sdktypes.Message) error {
			if err := e.store.Append(messageCtx, session.Record.ID, message); err != nil {
				return err
			}
			if input.OnMessage != nil {
				return input.OnMessage(messageCtx, session.Record.ID, message)
			}
			return nil
		},
	})
	defer a.Close()
	if err := a.Init(taskCtx); err != nil {
		e.markSessionFailed(session.Record.ID, err)
		return nil, fmt.Errorf("initialize cloud runtime agent: %w", err)
	}
	result, err := a.Prompt(taskCtx, input.Prompt)
	if err != nil {
		e.markSessionFailed(session.Record.ID, err)
		return nil, err
	}
	if err := e.store.MarkCompleted(taskCtx, session.Record.ID); err != nil {
		return nil, fmt.Errorf("complete cloud runtime session: %w", err)
	}
	return &RunResult{
		Output:    result.Text,
		SessionID: session.Record.ID,
		Model:     model,
		Usage:     result.Usage,
		CostUSD:   result.Cost,
		Turns:     result.NumTurns,
		Duration:  result.Duration,
	}, nil
}

func (e *Executor) markSessionFailed(sessionID string, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.store.MarkFailed(ctx, sessionID, cause); err != nil {
		// The task failure path remains authoritative. A session-status write
		// must not replace the provider error that caused the task to fail.
		return
	}
}

func (e *Executor) MarkSessionCancelled(ctx context.Context, sessionID string) error {
	return e.store.MarkCancelled(ctx, sessionID)
}
