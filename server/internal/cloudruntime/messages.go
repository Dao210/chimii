package cloudruntime

import (
	"strings"

	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
)

// TaskMessage is the execution-engine-neutral transcript shape persisted by
// the control plane. Seq is assigned by the worker after conversion.
type TaskMessage struct {
	Seq     int
	Type    string
	Tool    string
	Content string
	Input   map[string]any
	Output  string
}

func TaskMessagesFromSDK(message sdktypes.Message) []TaskMessage {
	out := make([]TaskMessage, 0, len(message.Content))
	for _, block := range message.Content {
		switch block.Type {
		case sdktypes.ContentBlockText:
			if message.Role == "assistant" && strings.TrimSpace(block.Text) != "" {
				out = append(out, TaskMessage{Type: "text", Content: block.Text})
			}
		case sdktypes.ContentBlockThinking:
			if strings.TrimSpace(block.Thinking) != "" {
				out = append(out, TaskMessage{Type: "thinking", Content: block.Thinking})
			}
		case sdktypes.ContentBlockToolUse:
			out = append(out, TaskMessage{Type: "tool_use", Tool: block.Name, Input: block.Input})
		case sdktypes.ContentBlockToolResult:
			var output strings.Builder
			for _, nested := range block.Content {
				if nested.Type == sdktypes.ContentBlockText {
					output.WriteString(nested.Text)
				}
			}
			out = append(out, TaskMessage{Type: "tool_result", Output: output.String()})
		}
	}
	return out
}
