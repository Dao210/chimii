package cloudruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chimii-ai/chimii/server/internal/runtimeconfig"
	sdkagent "github.com/codeany-ai/open-agent-sdk-go/agent"
	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
)

type executorSessionStub struct {
	options sdkagent.Options
}

func (s *executorSessionStub) Init(context.Context) error { return nil }
func (s *executorSessionStub) Close()                     {}
func (s *executorSessionStub) Prompt(ctx context.Context, prompt string) (*sdkagent.QueryResult, error) {
	if s.options.OnMessage != nil {
		if err := s.options.OnMessage(ctx, sdktypes.Message{Role: "user", Content: []sdktypes.ContentBlock{{Type: sdktypes.ContentBlockText, Text: prompt}}}); err != nil {
			return nil, err
		}
		if err := s.options.OnMessage(ctx, sdktypes.Message{Role: "assistant", Content: []sdktypes.ContentBlock{{Type: sdktypes.ContentBlockText, Text: "done"}}}); err != nil {
			return nil, err
		}
	}
	return &sdkagent.QueryResult{Text: "done", NumTurns: 1}, nil
}

func TestExecutorUsesExplicitProviderAndDisablesHostTools(t *testing.T) {
	t.Parallel()
	queries := &fakeSessionQueries{}
	config := runtimeconfig.Config{
		CloudEnabled: true,
		CloudProviders: map[string]runtimeconfig.ProviderConfig{
			runtimeconfig.ProviderAnthropic: {
				APIKey:       "server-secret",
				BaseURL:      "https://anthropic.example",
				DefaultModel: "claude-approved",
			},
		},
		TaskTimeout: time.Minute,
	}
	executor := NewExecutor(config, &SessionStore{queries: queries})
	var options sdkagent.Options
	executor.newAgent = func(got sdkagent.Options) sdkSession {
		options = got
		return &executorSessionStub{options: got}
	}
	pinned := ""
	result, err := executor.Execute(context.Background(), RunInput{
		SessionSpec: SessionSpec{
			WorkspaceID: testUUID(1),
			RuntimeID:   testUUID(2),
			AgentID:     testUUID(3),
			Provider:    runtimeconfig.ProviderAnthropic,
			ContextType: "issue",
			ContextID:   "issue-1",
		},
		Prompt: "hello",
		OnSession: func(_ context.Context, sessionID string) error {
			pinned = sessionID
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "done" || result.Model != "claude-approved" || pinned == "" {
		t.Fatalf("result=%+v pinned=%q", result, pinned)
	}
	if options.Provider != runtimeconfig.ProviderAnthropic || options.Model != "claude-approved" || options.APIKey != "server-secret" {
		t.Fatalf("provider options = provider=%q model=%q key=%q", options.Provider, options.Model, options.APIKey)
	}
	if !options.DisableDefaultTools || options.MaxTurns != 1 || options.FallbackModel != "" {
		t.Fatalf("unsafe executor options: disable_tools=%v max_turns=%d fallback=%q", options.DisableDefaultTools, options.MaxTurns, options.FallbackModel)
	}
	if len(queries.appended) != 2 {
		t.Fatalf("durable messages = %d, want 2", len(queries.appended))
	}
}

func TestExecutorRejectsUnconfiguredModelWithoutCreatingSession(t *testing.T) {
	t.Parallel()
	queries := &fakeSessionQueries{}
	executor := NewExecutor(runtimeconfig.Config{
		CloudEnabled: true,
		CloudProviders: map[string]runtimeconfig.ProviderConfig{
			runtimeconfig.ProviderOpenAI: {APIKey: "secret", DefaultModel: "gpt-approved"},
		},
		TaskTimeout: time.Minute,
	}, &SessionStore{queries: queries})
	_, err := executor.Execute(context.Background(), RunInput{
		SessionSpec: SessionSpec{
			WorkspaceID: testUUID(1),
			RuntimeID:   testUUID(2),
			AgentID:     testUUID(3),
			Provider:    runtimeconfig.ProviderOpenAI,
			ContextType: "issue",
			ContextID:   "issue-1",
		},
		Model:  "unapproved-model",
		Prompt: "hello",
	})
	if !errors.Is(err, ErrModelNotConfigured) {
		t.Fatalf("error = %v, want ErrModelNotConfigured", err)
	}
	if queries.record.ID != "" {
		t.Fatal("executor created a session before rejecting the model")
	}
}
