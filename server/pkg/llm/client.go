// Package llm is a thin, reusable wrapper around the Anthropic Messages API
// implementation in server/runtime. It gives server-internal features such as
// chat-title generation and Build planning one configured entry point without
// exposing a general-purpose LLM proxy to clients.
package llm

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/codeany-ai/open-agent-sdk-go/api"
	"github.com/codeany-ai/open-agent-sdk-go/types"
)

const (
	// FallbackModel is used when CHIMII_LLM_DEFAULT_MODEL is empty. It matches
	// the default used by the in-process runtime's Anthropic client.
	FallbackModel = "sonnet-4-6"

	defaultBaseURL        = "https://api.anthropic.com"
	defaultRequestTimeout = 60 * time.Second
)

// ErrNotConfigured is returned when neither an API key nor a base URL was
// configured. Internal callers treat this as a disabled-LLM signal and fall
// back gracefully.
var ErrNotConfigured = errors.New("llm: no API key or base URL configured")

// Config holds the Anthropic Messages API connection settings. An empty Config
// yields a disabled client; a base URL without a key remains valid for a
// keyless Anthropic-compatible gateway.
type Config struct {
	// APIKey authenticates against the upstream. Maps to CHIMII_LLM_API_KEY.
	APIKey string
	// BaseURL is the Anthropic API origin. The client appends /v1/messages.
	// When empty, https://api.anthropic.com is used. Maps to
	// CHIMII_LLM_BASE_URL.
	BaseURL string
	// DefaultModel is used when a call omits the model. Maps to
	// CHIMII_LLM_DEFAULT_MODEL. When empty, FallbackModel is used.
	DefaultModel string
	// HTTPClient replaces the default transport. It is primarily a test seam.
	HTTPClient *http.Client
}

// Client is a configured, reusable Anthropic Messages API caller.
type Client struct {
	sdk          *api.Client
	defaultModel string
	enabled      bool
}

// New builds a Client. It never returns an error so deployments without a
// utility LLM can keep the feature disabled and degrade gracefully.
func New(cfg Config) *Client {
	apiKey := strings.TrimSpace(cfg.APIKey)
	configuredBaseURL := strings.TrimSpace(cfg.BaseURL)
	defaultModel := strings.TrimSpace(cfg.DefaultModel)
	if defaultModel == "" {
		defaultModel = FallbackModel
	}

	baseURL := configuredBaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}

	return &Client{
		sdk: api.NewClient(api.ClientConfig{
			APIKey:             apiKey,
			BaseURL:            baseURL,
			Model:              defaultModel,
			Provider:           api.ProviderAnthropic,
			HTTPClient:         httpClient,
			CustomHeaders:      map[string]string{},
			DisableEnvFallback: true,
		}),
		defaultModel: defaultModel,
		enabled:      apiKey != "" || configuredBaseURL != "",
	}
}

// Enabled reports whether the deployment supplied an API key or base URL.
func (c *Client) Enabled() bool { return c != nil && c.enabled }

// DefaultModel returns the effective default model.
func (c *Client) DefaultModel() string { return c.defaultModel }

// Message performs a non-streaming Anthropic Messages API call. A missing
// model is filled from CHIMII_LLM_DEFAULT_MODEL (or FallbackModel).
func (c *Client) Message(ctx context.Context, req api.MessagesRequest) (*api.StreamMessage, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.defaultModel
	}

	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	return c.sdk.CreateMessage(ctx, req)
}

// GenerateText sends an optional system prompt and one user message, then
// concatenates the text blocks in the Anthropic response. These bounded utility
// calls explicitly disable extended thinking: compatible providers may enable
// it by default. Use GenerateReasonedText for geometry or other planning work.
// An empty model uses the configured default.
func (c *Client) GenerateText(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	return c.generateText(ctx, model, systemPrompt, userPrompt, &api.ThinkingConfig{Type: "disabled"}, nil)
}

// GenerateReasonedText keeps a small reasoning budget for constraint-heavy
// utility work. Set effort explicitly as well: compatible providers such as
// DeepSeek ignore budget_tokens and otherwise default to high effort.
func (c *Client) GenerateReasonedText(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	return c.generateText(ctx, model, systemPrompt, userPrompt, &api.ThinkingConfig{Type: "enabled", BudgetTokens: 2048}, &api.OutputConfig{Effort: "low"})
}

func (c *Client) generateText(ctx context.Context, model, systemPrompt, userPrompt string, thinking *api.ThinkingConfig, outputConfig *api.OutputConfig) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}

	req := api.MessagesRequest{
		Model:        strings.TrimSpace(model),
		Thinking:     thinking,
		OutputConfig: outputConfig,
		Messages: []api.APIMessage{{
			Role: "user",
			Content: []types.ContentBlock{{
				Type: types.ContentBlockText,
				Text: userPrompt,
			}},
		}},
	}
	if strings.TrimSpace(systemPrompt) != "" {
		req.System = []api.SystemBlock{{Type: "text", Text: systemPrompt}}
	}

	message, err := c.Message(ctx, req)
	if err != nil {
		return "", err
	}
	if message == nil {
		return "", errors.New("llm: upstream returned no message")
	}

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

func withDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, defaultRequestTimeout)
}
