package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdkagent "github.com/codeany-ai/open-agent-sdk-go/agent"
	"github.com/codeany-ai/open-agent-sdk-go/types"
)

// openagentBackend 实现 Backend interface，但与其他 17 个 backend 不同：
// 它不 spawn 外部 CLI 子进程，而是在 daemon 进程内直接跑 Open Agent SDK 的
// agent loop（32 个内置工具 + MCP + permissions + hooks + cost tracking）。
//
// 这是"agent runtime 内嵌"路径：daemon 不再需要用户机器上预装 claude/codex
// 等 CLI binary，SDK 自给自足。模型 API 仍由 SDK 的 api.Client 远程调用。
//
// daemon 调度链（agent.New → backend.Execute → drain loop）完全 provider-agnostic，
// 只要本 backend 正确产出 Message/Result 流，idle watchdog、task usage 上报、
// 取消语义全部复用，无需 daemon 侧特殊适配。
type openagentBackend struct{ cfg Config }

// Execute 启动一次 in-process agent 执行。
//
// 与 subprocess backend 的关键差异：
//   - 取消：直接靠 ctx.Cancel()，无需 SIGTERM→SIGKILL 进程组
//   - stderr：SDK 错误走 errCh，不扫描子进程 stderr
//   - MCP：opts.McpConfig JSON 解析为 map[string]types.MCPServerConfig，SDK 在
//     Init() 阶段连接，无需写临时文件传 --mcp-config
//   - resume：第一版不持久化 session，ResumeSessionID 忽略；后续可接 SDK
//     session.Manager
func (b *openagentBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	runCtx, cancel := runContext(ctx, opts.Timeout)

	sdkOpts, err := mapExecOptionsToSDK(opts, b.cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("openagent: map options: %w", err)
	}

	a := sdkagent.New(sdkOpts)
	if err := a.Init(runCtx); err != nil {
		cancel()
		a.Close()
		return nil, fmt.Errorf("openagent: init (MCP connect): %w", err)
	}

	b.cfg.Logger.Info("openagent started",
		"model", sdkOpts.Model, "cwd", opts.Cwd, "mcp_servers", len(sdkOpts.MCPServers))

	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)

	go func() {
		defer cancel()
		defer close(msgCh)
		defer close(resCh)
		defer a.Close()

		startTime := time.Now()
		events, errCh := a.Query(runCtx, prompt)

		var (
			finalResult  Result
			gotResult    bool
			finalStatus  = "completed"
			finalError   string
			sessionID    = a.SessionID()
			configuredModel = strings.TrimSpace(opts.Model)
			resultUsage  = make(map[string]TokenUsage)
		)

		for ev := range events {
			switch ev.Type {
			case types.MessageTypeAssistant:
				if ev.Message != nil {
					for _, blk := range ev.Message.Content {
						switch blk.Type {
						case types.ContentBlockText:
							if blk.Text != "" {
								trySend(msgCh, Message{Type: MessageText, Content: blk.Text})
							}
						case types.ContentBlockThinking:
							if blk.Thinking != "" {
								trySend(msgCh, Message{Type: MessageThinking, Content: blk.Thinking})
							}
						case types.ContentBlockToolUse:
							trySend(msgCh, Message{
								Type:   MessageToolUse,
								Tool:   blk.Name,
								CallID: blk.ID,
								Input:  blk.Input,
							})
						}
					}
				}
			case "tool_result":
				// SDK 在 loop.go 里发出的 tool_result 事件，携带 tool_use_id 与
				// content。映射到 daemon 的 MessageToolResult。
				var outputText string
				if ev.Message != nil {
					for _, blk := range ev.Message.Content {
						if blk.Type == types.ContentBlockText {
							outputText += blk.Text
						}
					}
				}
				trySend(msgCh, Message{
					Type:   MessageToolResult,
					CallID: extractToolUseID(ev.Message),
					Output: outputText,
				})
			case types.MessageTypeResult:
				gotResult = true
				finalResult = buildResultFromSDK(ev, sessionID, configuredModel, resultUsage, startTime)
			}
		}

		// SDK 的 errCh 在 events channel 关闭后才有值。
		if err := <-errCh; err != nil {
			finalStatus = "failed"
			finalError = err.Error()
		}

		if !gotResult {
			// loop 结束但没有 result 事件 —— 只有 errCh 报错时会发生。
			// 若 finalError 为空，给一个可诊断的兜底。
			if finalError == "" {
				finalError = "openagent: agent loop ended without result"
			}
			finalResult = Result{
				Status:     finalStatus,
				Error:      sanitizeAgentDiagnostic(finalError),
				DurationMs: time.Since(startTime).Milliseconds(),
				SessionID:  sessionID,
				Usage:      resultUsage,
			}
		}

		resCh <- finalResult
	}()

	return &Session{Messages: msgCh, Result: resCh}, nil
}

// extractToolUseID 从 SDK tool_result 事件的 message content 里提取 tool_use_id。
// SDK loop.go 把 tool_result 包成 ContentBlockToolResult，ID 在 ToolUseID 字段。
func extractToolUseID(msg *types.Message) string {
	if msg == nil {
		return ""
	}
	for _, blk := range msg.Content {
		if blk.Type == types.ContentBlockToolResult && blk.ToolUseID != "" {
			return blk.ToolUseID
		}
	}
	return ""
}

// buildResultFromSDK 把 SDK 的 result 事件转为 daemon 的 Result。
// SDK 的 Usage 是单 model 的累积值；daemon 的 Result.Usage 是 map[model]TokenUsage。
func buildResultFromSDK(ev types.SDKMessage, sessionID, configuredModel string, usage map[string]TokenUsage, startTime time.Time) Result {
	status := "completed"
	if ev.Text == "" && ev.Usage == nil {
		status = "failed"
	}

	// 把 SDK 的 Usage 累积到 usage map。SDK 的 QueryResult.Usage 是单次执行的
	// 总量，model 从 ev.Message.Model 或 configuredModel 取。
	model := configuredModel
	if ev.Message != nil && ev.Message.Model != "" {
		model = ev.Message.Model
	}
	if model == "" {
		model = "openagent"
	}

	if ev.Usage != nil {
		u := usage[model]
		u.InputTokens += int64(ev.Usage.InputTokens)
		u.OutputTokens += int64(ev.Usage.OutputTokens)
		u.CacheReadTokens += int64(ev.Usage.CacheReadInputTokens)
		u.CacheWriteTokens += int64(ev.Usage.CacheCreationInputTokens)
		usage[model] = u
	}

	durationMs := ev.Duration
	if durationMs <= 0 {
		durationMs = time.Since(startTime).Milliseconds()
	}

	return Result{
		Status:     status,
		Output:     ev.Text,
		DurationMs: durationMs,
		SessionID:  sessionID,
		Usage:      usage,
	}
}

// mapExecOptionsToSDK 把 daemon 的 ExecOptions + Config 映射为 SDK 的 Options。
//
// SDK 的 resolveEnvOptions 会自动从 Env 读 CODEANY_API_KEY / ANTHROPIC_API_KEY /
// CODEANY_MODEL 等，所以 daemon 注入的 Config.Env 直接透传即可。
// CWD/Model/SystemPrompt/MaxTurns/Timeout 直接字段映射。
// McpConfig JSON 解析为 map[string]types.MCPServerConfig。
// ThinkingLevel 映射为 SDK 的 Effort（语义对齐：low/medium/high/max）。
func mapExecOptionsToSDK(opts ExecOptions, cfg Config) (sdkagent.Options, error) {
	sdkOpts := sdkagent.Options{
		Model:        opts.Model,
		CWD:          opts.Cwd,
		SystemPrompt: opts.SystemPrompt,
		MaxTurns:     opts.MaxTurns,
		TimeoutMs:    int(opts.Timeout.Milliseconds()),
		Env:          cfg.Env,
		// daemon 层已做权限控制，SDK 内 bypass permissions 避免双重审批。
		PermissionMode: types.PermissionModeBypassPermissions,
	}

	// MCP config: {"mcpServers":{"<name>":{...}}} → map[string]types.MCPServerConfig
	if hasManagedMcpConfig(opts.McpConfig) {
		var raw struct {
			MCPServers map[string]types.MCPServerConfig `json:"mcpServers"`
		}
		if err := json.Unmarshal(opts.McpConfig, &raw); err != nil {
			return sdkOpts, fmt.Errorf("parse mcp config: %w", err)
		}
		sdkOpts.MCPServers = raw.MCPServers
	}

	// ThinkingLevel: runtime-native reasoning effort。
	// 不同 provider 的值域不同（Claude: low|medium|high|xhigh|max；
	// Codex: none|minimal|low|medium|high|xhigh）。SDK 的 Effort 值域是
	// low|medium|high|max，语义对齐。空值不设置，SDK 用模型默认。
	if lvl := strings.TrimSpace(opts.ThinkingLevel); lvl != "" {
		switch strings.ToLower(lvl) {
		case "none", "minimal":
			// 显式禁用思考 —— SDK 不设置 Effort 也不设置 Thinking，
			// 走模型默认（多数模型默认不开 thinking）。
		case "low":
			sdkOpts.Effort = sdkagent.EffortLow
		case "medium":
			sdkOpts.Effort = sdkagent.EffortMedium
		case "high":
			sdkOpts.Effort = sdkagent.EffortHigh
		case "xhigh", "max":
			sdkOpts.Effort = sdkagent.EffortMax
		default:
			// 未知值不阻断执行，按 medium 兜底。
			sdkOpts.Effort = sdkagent.EffortMedium
		}
	}

	return sdkOpts, nil
}
