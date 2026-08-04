package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/codeany-ai/open-agent-sdk-go/types"
)

func TestDisableDefaultToolsLeavesRegistryEmpty(t *testing.T) {
	a := New(Options{DisableDefaultTools: true, APIKey: "test", Model: "test"})
	defer a.Close()
	if got := a.toolRegistry.Names(); len(got) != 0 {
		t.Fatalf("tool registry = %v, want empty", got)
	}
}

func TestAppendMessageFailsClosedWhenPersistenceFails(t *testing.T) {
	want := errors.New("database unavailable")
	a := New(Options{
		DisableDefaultTools: true,
		APIKey:              "test",
		Model:               "test",
		OnMessage: func(context.Context, types.Message) error {
			return want
		},
	})
	defer a.Close()
	err := a.appendMessage(context.Background(), types.Message{Role: "user"})
	if !errors.Is(err, want) {
		t.Fatalf("append error = %v, want %v", err, want)
	}
}

func TestInitialMessagesAreRestoredAndCopied(t *testing.T) {
	initial := []types.Message{{
		Type:      types.MessageTypeUser,
		Role:      "user",
		UUID:      "message-1",
		Timestamp: time.Unix(1, 0),
		Content:   []types.ContentBlock{{Type: types.ContentBlockText, Text: "hello"}},
	}}
	a := New(Options{DisableDefaultTools: true, APIKey: "test", Model: "test", InitialMessages: initial})
	defer a.Close()
	initial[0].Role = "mutated"
	got := a.GetMessages()
	if len(got) != 1 || got[0].Role != "user" {
		t.Fatalf("messages = %+v", got)
	}
}
