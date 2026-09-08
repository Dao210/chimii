package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/codeany-ai/open-agent-sdk-go/api"
	"github.com/codeany-ai/open-agent-sdk-go/types"
)

type capturedRequest struct {
	Path             string
	APIKey           string
	Authorization    string
	AnthropicVersion string
	Body             map[string]any
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubHTTPClient(t *testing.T, handler func(got capturedRequest) string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		responseBody := handler(capturedRequest{
			Path:             r.URL.Path,
			APIKey:           r.Header.Get("X-API-Key"),
			Authorization:    r.Header.Get("Authorization"),
			AnthropicVersion: r.Header.Get("Anthropic-Version"),
			Body:             body,
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Request:    r,
		}, nil
	})}
}

func anthropicMessage(model, text string) string {
	return `{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"` + text + `"}],"model":"` + model + `","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`
}

func TestNewDisabledClient(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "ambient-secret-must-not-enable-client")
	c := New(Config{})
	if c.Enabled() {
		t.Fatal("expected disabled client with empty config")
	}
	if c.DefaultModel() != FallbackModel {
		t.Fatalf("expected fallback model %q, got %q", FallbackModel, c.DefaultModel())
	}
	if _, err := c.Message(context.Background(), api.MessagesRequest{}); err != ErrNotConfigured {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
	if _, err := c.GenerateText(context.Background(), "", "", "hi"); err != ErrNotConfigured {
		t.Fatalf("expected ErrNotConfigured from GenerateText, got %v", err)
	}
}

func TestEnabledWithBaseURLOnly(t *testing.T) {
	c := New(Config{BaseURL: "http://localhost:1234"})
	if !c.Enabled() {
		t.Fatal("expected enabled client for a keyless Anthropic-compatible gateway")
	}
}

func TestBaseURLOnlyDoesNotUseAmbientAnthropicKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "ambient-secret")
	var gotAPIKey string
	httpClient := stubHTTPClient(t, func(got capturedRequest) string {
		gotAPIKey = got.APIKey
		return anthropicMessage("claude-default", "hello")
	})
	c := New(Config{
		BaseURL:      "https://anthropic.example",
		DefaultModel: "claude-default",
		HTTPClient:   httpClient,
	})
	if _, err := c.GenerateText(context.Background(), "", "", "hi"); err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}
	if gotAPIKey != "" {
		t.Fatalf("request used ambient Anthropic API key %q", gotAPIKey)
	}
}

func TestConfiguredDefaultModel(t *testing.T) {
	c := New(Config{APIKey: "k", DefaultModel: "claude-configured"})
	if c.DefaultModel() != "claude-configured" {
		t.Fatalf("expected configured default model, got %q", c.DefaultModel())
	}
}

func TestMessageUsesAnthropicProtocolAndDefaultModel(t *testing.T) {
	var captured capturedRequest
	httpClient := stubHTTPClient(t, func(got capturedRequest) string {
		captured = got
		return anthropicMessage("claude-default", "hello")
	})

	c := New(Config{APIKey: "test-key", BaseURL: "https://anthropic.example", DefaultModel: "claude-default", HTTPClient: httpClient})
	message, err := c.Message(context.Background(), api.MessagesRequest{
		Messages: []api.APIMessage{{
			Role:    "user",
			Content: []types.ContentBlock{{Type: types.ContentBlockText, Text: "hi"}},
		}},
	})
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if captured.Path != "/v1/messages" {
		t.Fatalf("request path = %q", captured.Path)
	}
	if captured.APIKey != "test-key" || captured.AnthropicVersion == "" {
		t.Fatalf("missing Anthropic headers: %+v", captured)
	}
	if captured.Authorization != "" {
		t.Fatalf("request unexpectedly used an OpenAI Authorization header: %q", captured.Authorization)
	}
	if captured.Body["model"] != "claude-default" {
		t.Fatalf("request model = %v", captured.Body["model"])
	}
	if len(message.Content) != 1 || message.Content[0].Text != "hello" {
		t.Fatalf("unexpected message: %+v", message)
	}
}

func TestMessageRespectsRequestModel(t *testing.T) {
	var gotModel string
	httpClient := stubHTTPClient(t, func(got capturedRequest) string {
		gotModel, _ = got.Body["model"].(string)
		return anthropicMessage(gotModel, "hello")
	})

	c := New(Config{APIKey: "test-key", BaseURL: "https://anthropic.example", DefaultModel: "claude-default", HTTPClient: httpClient})
	_, err := c.Message(context.Background(), api.MessagesRequest{
		Model: "claude-caller",
		Messages: []api.APIMessage{{
			Role:    "user",
			Content: []types.ContentBlock{{Type: types.ContentBlockText, Text: "hi"}},
		}},
	})
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if gotModel != "claude-caller" {
		t.Fatalf("expected caller model preserved, got %q", gotModel)
	}
}

func TestGenerateText(t *testing.T) {
	var captured capturedRequest
	httpClient := stubHTTPClient(t, func(got capturedRequest) string {
		captured = got
		return anthropicMessage("claude-default", "a title")
	})

	c := New(Config{APIKey: "k", BaseURL: "https://anthropic.example", DefaultModel: "claude-default", HTTPClient: httpClient})
	out, err := c.GenerateText(context.Background(), "", "you are helpful", "make a title")
	if err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}
	if out != "a title" {
		t.Fatalf("expected %q, got %q", "a title", out)
	}
	thinking, ok := captured.Body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Fatalf("utility request inherited provider thinking defaults: %#v", captured.Body["thinking"])
	}
	if _, present := thinking["budget_tokens"]; present {
		t.Fatal("disabled thinking must not send a token budget")
	}
	if _, present := captured.Body["output_config"]; present {
		t.Fatal("simple text calls must not request reasoning effort")
	}
	system, ok := captured.Body["system"].([]any)
	if !ok || len(system) != 1 || system[0].(map[string]any)["text"] != "you are helpful" {
		t.Fatalf("unexpected system prompt: %#v", captured.Body["system"])
	}
	messages, ok := captured.Body["messages"].([]any)
	if !ok || len(messages) != 1 || messages[0].(map[string]any)["role"] != "user" {
		t.Fatalf("unexpected messages: %#v", captured.Body["messages"])
	}
}

func TestGenerateReasonedTextUsesLowEffortAndReturnsOnlyText(t *testing.T) {
	var captured capturedRequest
	c := New(Config{BaseURL: "https://anthropic.example", HTTPClient: stubHTTPClient(t, func(got capturedRequest) string {
		captured = got
		return `{"content":[{"type":"thinking","thinking":"private reasoning"},{"type":"text","text":"the plan"}]}`
	})})
	out, err := c.GenerateReasonedText(context.Background(), "", "plan geometry", "a tower")
	if err != nil || out != "the plan" {
		t.Fatalf("unexpected final text: %q, error: %v", out, err)
	}
	thinking, ok := captured.Body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["budget_tokens"] != float64(2048) {
		t.Fatalf("missing bounded thinking configuration: %#v", captured.Body["thinking"])
	}
	output, ok := captured.Body["output_config"].(map[string]any)
	if !ok || output["effort"] != "low" {
		t.Fatalf("request inherited provider effort defaults: %#v", captured.Body["output_config"])
	}
}

func TestMessagePreservesExplicitThinkingBudget(t *testing.T) {
	var captured capturedRequest
	c := New(Config{BaseURL: "https://anthropic.example", HTTPClient: stubHTTPClient(t, func(got capturedRequest) string {
		captured = got
		return anthropicMessage("claude-default", "answer")
	})})
	_, err := c.Message(context.Background(), api.MessagesRequest{
		Thinking: &api.ThinkingConfig{Type: "enabled", BudgetTokens: 1024},
		Messages: []api.APIMessage{{Role: "user", Content: []types.ContentBlock{{Type: types.ContentBlockText, Text: "reason about this"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	thinking, ok := captured.Body["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" || thinking["budget_tokens"] != float64(1024) {
		t.Fatalf("explicit reasoning budget was changed: %#v", captured.Body["thinking"])
	}
}
