package agent

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	sdkagent "github.com/codeany-ai/open-agent-sdk-go/agent"
	"github.com/codeany-ai/open-agent-sdk-go/types"
)

func TestNewReturnsOpenagentBackend(t *testing.T) {
	t.Parallel()
	// openagent 不 spawn 外部 CLI，不需要 ExecutablePath。
	b, err := New("openagent", Config{Logger: slog.Default()})
	if err != nil {
		t.Fatalf("New(openagent) error: %v", err)
	}
	if _, ok := b.(*openagentBackend); !ok {
		t.Fatalf("expected *openagentBackend, got %T", b)
	}
}

func TestIsSupportedTypeOpenagent(t *testing.T) {
	t.Parallel()
	if !IsSupportedType("openagent") {
		t.Fatal("expected openagent in SupportedTypes whitelist")
	}
}

func TestLaunchHeaderOpenagent(t *testing.T) {
	t.Parallel()
	if h := LaunchHeader("openagent"); h == "" {
		t.Fatal("expected non-empty launch header for openagent")
	}
}

func TestMapExecOptionsToSDKDefaults(t *testing.T) {
	t.Parallel()
	opts := ExecOptions{Cwd: "/tmp/work", Model: "sonnet-4-6"}
	cfg := Config{Env: map[string]string{"CODEANY_API_KEY": "test-key"}}

	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("mapExecOptionsToSDK error: %v", err)
	}
	if sdkOpts.Model != "sonnet-4-6" {
		t.Errorf("Model = %q, want %q", sdkOpts.Model, "sonnet-4-6")
	}
	if sdkOpts.CWD != "/tmp/work" {
		t.Errorf("CWD = %q, want %q", sdkOpts.CWD, "/tmp/work")
	}
	if sdkOpts.PermissionMode != types.PermissionModeBypassPermissions {
		t.Errorf("PermissionMode = %q, want bypass", sdkOpts.PermissionMode)
	}
	if sdkOpts.Env["CODEANY_API_KEY"] != "test-key" {
		t.Errorf("Env not propagated: %v", sdkOpts.Env)
	}
	if len(sdkOpts.MCPServers) != 0 {
		t.Errorf("expected no MCP servers, got %d", len(sdkOpts.MCPServers))
	}
}

func TestMapExecOptionsToSDKMCPConfig(t *testing.T) {
	t.Parallel()
	mcpJSON := json.RawMessage(`{"mcpServers":{"fetch":{"command":"uvx","args":["mcp-server-fetch"]},"api":{"type":"http","url":"http://localhost:3000/mcp"}}}`)
	opts := ExecOptions{McpConfig: mcpJSON}
	cfg := Config{}

	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("mapExecOptionsToSDK error: %v", err)
	}
	if len(sdkOpts.MCPServers) != 2 {
		t.Fatalf("expected 2 MCP servers, got %d", len(sdkOpts.MCPServers))
	}
	fetch, ok := sdkOpts.MCPServers["fetch"]
	if !ok {
		t.Fatal("expected 'fetch' MCP server")
	}
	if fetch.Command != "uvx" {
		t.Errorf("fetch.Command = %q, want %q", fetch.Command, "uvx")
	}
	if fetch.Type != "" && fetch.Type != types.MCPTransportStdio {
		t.Errorf("fetch.Type = %q, want stdio or empty", fetch.Type)
	}
	api, ok := sdkOpts.MCPServers["api"]
	if !ok {
		t.Fatal("expected 'api' MCP server")
	}
	if api.URL != "http://localhost:3000/mcp" {
		t.Errorf("api.URL = %q, want %q", api.URL, "http://localhost:3000/mcp")
	}
}

func TestMapExecOptionsToSDKMCPConfigNull(t *testing.T) {
	t.Parallel()
	// JSON null 表示"继承 runtime 配置"，不应产生 MCP servers。
	opts := ExecOptions{McpConfig: json.RawMessage(`null`)}
	cfg := Config{}

	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("mapExecOptionsToSDK error: %v", err)
	}
	if len(sdkOpts.MCPServers) != 0 {
		t.Errorf("expected 0 MCP servers for null config, got %d", len(sdkOpts.MCPServers))
	}
}

func TestMapExecOptionsToSDKMCPConfigInvalid(t *testing.T) {
	t.Parallel()
	opts := ExecOptions{McpConfig: json.RawMessage(`{invalid json`)}
	cfg := Config{}

	if _, err := mapExecOptionsToSDK(opts, cfg); err == nil {
		t.Fatal("expected error for invalid MCP config JSON")
	}
}

func TestMapExecOptionsToSDKThinkingLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		level string
		want  sdkagent.Effort
	}{
		{"low", sdkagent.EffortLow},
		{"medium", sdkagent.EffortMedium},
		{"high", sdkagent.EffortHigh},
		{"max", sdkagent.EffortMax},
		{"xhigh", sdkagent.EffortMax},
	}
	for _, tc := range cases {
		opts := ExecOptions{ThinkingLevel: tc.level}
		cfg := Config{}
		sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
		if err != nil {
			t.Fatalf("level=%q: error: %v", tc.level, err)
		}
		if sdkOpts.Effort != tc.want {
			t.Errorf("level=%q: Effort = %q, want %q", tc.level, sdkOpts.Effort, tc.want)
		}
	}
}

func TestMapExecOptionsToSDKThinkingLevelEmpty(t *testing.T) {
	t.Parallel()
	opts := ExecOptions{ThinkingLevel: ""}
	cfg := Config{}
	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if sdkOpts.Effort != "" {
		t.Errorf("expected empty Effort for empty ThinkingLevel, got %q", sdkOpts.Effort)
	}
}

func TestMapExecOptionsToSDKThinkingLevelDisabled(t *testing.T) {
	t.Parallel()
	// "none"/"minimal" 显式禁用思考 —— 不设置 Effort 也不设置 Thinking。
	for _, lvl := range []string{"none", "minimal", "NONE", "Minimal"} {
		opts := ExecOptions{ThinkingLevel: lvl}
		cfg := Config{}
		sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
		if err != nil {
			t.Fatalf("level=%q: error: %v", lvl, err)
		}
		if sdkOpts.Effort != "" {
			t.Errorf("level=%q: expected empty Effort (disabled), got %q", lvl, sdkOpts.Effort)
		}
		if sdkOpts.Thinking != nil {
			t.Errorf("level=%q: expected nil Thinking (disabled), got %+v", lvl, sdkOpts.Thinking)
		}
	}
}

func TestMapExecOptionsToSDKThinkingLevelUnknown(t *testing.T) {
	t.Parallel()
	opts := ExecOptions{ThinkingLevel: "bogus"}
	cfg := Config{}
	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// 未知值不阻断执行，按 medium 兜底。
	if sdkOpts.Effort != sdkagent.EffortMedium {
		t.Errorf("expected EffortMedium fallback for unknown level, got %q", sdkOpts.Effort)
	}
}

func TestMapExecOptionsToSDKSystemPrompt(t *testing.T) {
	t.Parallel()
	opts := ExecOptions{SystemPrompt: "You are a test agent."}
	cfg := Config{}
	sdkOpts, err := mapExecOptionsToSDK(opts, cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if sdkOpts.SystemPrompt != "You are a test agent." {
		t.Errorf("SystemPrompt = %q, want %q", sdkOpts.SystemPrompt, "You are a test agent.")
	}
}

func TestBuildResultFromSDK(t *testing.T) {
	t.Parallel()
	usage := make(map[string]TokenUsage)
	ev := types.SDKMessage{
		Type:     types.MessageTypeResult,
		Text:     "task completed",
		Usage:    &types.Usage{InputTokens: 100, OutputTokens: 50},
		NumTurns: 3,
		Duration: 5000,
	}
	result := buildResultFromSDK(ev, "session-123", "sonnet-4-6", usage, time.Now())
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if result.Output != "task completed" {
		t.Errorf("Output = %q, want %q", result.Output, "task completed")
	}
	if result.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", result.SessionID, "session-123")
	}
	if result.DurationMs != 5000 {
		t.Errorf("DurationMs = %d, want 5000", result.DurationMs)
	}
	tu, ok := result.Usage["sonnet-4-6"]
	if !ok {
		t.Fatal("expected usage entry for sonnet-4-6")
	}
	if tu.InputTokens != 100 || tu.OutputTokens != 50 {
		t.Errorf("Usage = %+v, want Input=100 Output=50", tu)
	}
}

func TestBuildResultFromSDKEmptyResultIsFailed(t *testing.T) {
	t.Parallel()
	usage := make(map[string]TokenUsage)
	// 空 Text + nil Usage → 失败
	ev := types.SDKMessage{
		Type: types.MessageTypeResult,
		Text: "",
	}
	result := buildResultFromSDK(ev, "s1", "m1", usage, time.Now())
	if result.Status != "failed" {
		t.Errorf("Status = %q, want failed for empty result", result.Status)
	}
}

func TestExtractToolUseID(t *testing.T) {
	t.Parallel()
	// nil message → 空
	if got := extractToolUseID(nil); got != "" {
		t.Errorf("extractToolUseID(nil) = %q, want empty", got)
	}
	// 无 tool_result block → 空
	msg := &types.Message{Content: []types.ContentBlock{{Type: types.ContentBlockText, Text: "hi"}}}
	if got := extractToolUseID(msg); got != "" {
		t.Errorf("extractToolUseID = %q, want empty", got)
	}
	// 有 tool_result block
	msg2 := &types.Message{Content: []types.ContentBlock{{
		Type:      types.ContentBlockToolResult,
		ToolUseID: "call-abc",
	}}}
	if got := extractToolUseID(msg2); got != "call-abc" {
		t.Errorf("extractToolUseID = %q, want %q", got, "call-abc")
	}
}

func TestOpenagentInSupportedTypesSlice(t *testing.T) {
	t.Parallel()
	// 确保 SupportedTypes 切片确实包含 openagent，与 IsSupportedType 保持一致。
	found := false
	for _, t := range SupportedTypes {
		if t == "openagent" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("openagent not found in SupportedTypes slice")
	}
}

func TestOpenagentErrorUnknownTypeMentionsOpenagent(t *testing.T) {
	t.Parallel()
	_, err := New("bogus", Config{})
	if err == nil {
		t.Fatal("expected error for bogus agent type")
	}
	if !strings.Contains(err.Error(), "openagent") {
		t.Errorf("error message should mention openagent as supported, got: %v", err)
	}
}
