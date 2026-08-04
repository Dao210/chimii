package cloudruntime

import (
	"testing"

	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
)

func TestTaskMessagesFromSDK(t *testing.T) {
	t.Parallel()
	got := TaskMessagesFromSDK(sdktypes.Message{
		Role: "assistant",
		Content: []sdktypes.ContentBlock{
			{Type: sdktypes.ContentBlockThinking, Thinking: "considering"},
			{Type: sdktypes.ContentBlockText, Text: "answer"},
			{Type: sdktypes.ContentBlockToolUse, Name: "Read", Input: map[string]any{"file_path": "/workspace/a"}},
		},
	})
	if len(got) != 3 {
		t.Fatalf("message count = %d, want 3: %+v", len(got), got)
	}
	if got[0].Type != "thinking" || got[1].Content != "answer" || got[2].Tool != "Read" {
		t.Fatalf("unexpected conversion: %+v", got)
	}
}

func TestTaskMessagesFromSDKSkipsUserPromptText(t *testing.T) {
	t.Parallel()
	got := TaskMessagesFromSDK(sdktypes.Message{
		Role:    "user",
		Content: []sdktypes.ContentBlock{{Type: sdktypes.ContentBlockText, Text: "private prompt input"}},
	})
	if len(got) != 0 {
		t.Fatalf("user input leaked into execution transcript: %+v", got)
	}
}
