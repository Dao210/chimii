package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/codeany-ai/open-agent-sdk-go/api"
	"github.com/codeany-ai/open-agent-sdk-go/types"
)

type Image struct {
	MediaType string
	Data      []byte
}
type GenerationOptions struct {
	Effort  string
	Purpose string
	Images  []Image
}

// GenerateContent provides explicit per-task reasoning and real image input.
// It never substitutes a text projection for an unavailable image.
func (c *Client) GenerateContent(ctx context.Context, model, system, user string, options GenerationOptions) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}
	if len(options.Images) > 6 {
		return "", errors.New("llm: at most six review images are supported")
	}
	thinking := &api.ThinkingConfig{Type: "disabled"}
	var output *api.OutputConfig
	if options.Effort != "" {
		switch options.Effort {
		case "low", "medium", "high", "max":
		default:
			return "", fmt.Errorf("llm: unsupported reasoning effort %q", options.Effort)
		}
		thinking = &api.ThinkingConfig{Type: "enabled", BudgetTokens: 2048}
		output = &api.OutputConfig{Effort: options.Effort}
	}
	content := []types.ContentBlock{{Type: types.ContentBlockText, Text: user}}
	for _, image := range options.Images {
		if len(image.Data) == 0 || len(image.Data) > 2<<20 || (image.MediaType != "image/png" && image.MediaType != "image/jpeg") {
			return "", errors.New("llm: invalid or oversized review image")
		}
		content = append(content, types.ContentBlock{Type: types.ContentBlockImage, Source: &types.ImageSource{Type: "base64", MediaType: image.MediaType, Data: base64.StdEncoding.EncodeToString(image.Data)}})
	}
	request := api.MessagesRequest{Model: model, Thinking: thinking, OutputConfig: output, Messages: []api.APIMessage{{Role: "user", Content: content}}}
	if strings.TrimSpace(system) != "" {
		request.System = []api.SystemBlock{{Type: "text", Text: system}}
	}
	started := time.Now()
	message, err := c.Message(ctx, request)
	if err != nil {
		requestedModel := model
		if strings.TrimSpace(requestedModel) == "" {
			requestedModel = c.defaultModel
		}
		// Keep failure timing visible without logging prompts, images or keys.
		slog.Warn("llm: task generation failed", "purpose", options.Purpose, "model", requestedModel, "effort", options.Effort, "images", len(options.Images), "elapsed_ms", time.Since(started).Milliseconds(), "deadline_exceeded", errors.Is(err, context.DeadlineExceeded))
		return "", err
	}
	if message == nil {
		return "", errors.New("llm: upstream returned no message")
	}
	metrics := []any{"purpose", options.Purpose, "model", message.Model, "effort", options.Effort, "images", len(options.Images), "elapsed_ms", time.Since(started).Milliseconds()}
	if message.Usage != nil {
		metrics = append(metrics, "input_tokens", message.Usage.InputTokens, "output_tokens", message.Usage.OutputTokens)
	}
	slog.Info("llm: task generation completed", metrics...)
	var text strings.Builder
	for _, block := range message.Content {
		if block.Type == types.ContentBlockText {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return "", errors.New("llm: upstream returned no text content")
	}
	return text.String(), nil
}
