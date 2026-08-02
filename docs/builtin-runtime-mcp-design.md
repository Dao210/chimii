# Builtin In-Process Runtime with Full MCP Tool Support (L3) — 复用 @codeany/open-agent-sdk

## 核心发现

`server/src/` 已经是一个完整的 in-process TS agent SDK —— [`@codeany/open-agent-sdk`](https://github.com/codeany-ai/open-agent-sdk-typescript) v0.2.4。它完整覆盖了 L3 builtin runtime 所需的全部能力：

| 能力 | 现有实现 | 位置 |
|---|---|---|
| Agent loop（in-process） | ✅ `QueryEngine` 完整循环：prompt → LLM → tool_call → result → 重复 | [server/src/engine.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/engine.ts) |
| LLM provider 抽象 | ✅ Anthropic Messages + OpenAI Chat Completions 双后端 | [server/src/providers/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/providers/index.ts) |
| MCP client（stdio/SSE/HTTP） | ✅ 基于官方 `@modelcontextprotocol/sdk` | [server/src/mcp/client.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/mcp/client.ts) |
| 30+ 内置工具 | ✅ bash/edit/read/write/glob/grep/web-fetch/web-search/task/agent/team/todo/cron/lsp/... | [server/src/tools/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/tools/index.ts) |
| 技能系统（含 5 个 bundled） | ✅ commit / debug / review / simplify / test | [server/src/skills/bundled/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/skills/bundled/index.ts) |
| Session 持久化与 resume | ✅ JSON 文件存储，支持 list/fork/tag | [server/src/session.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/session.ts) |
| Hook 系统 | ✅ pre/post tool use、session、compact 生命周期 | [server/src/hooks.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/hooks.ts) |
| In-process SDK MCP server | ✅ `createSdkMcpServer()` 把 `tool()` 定义包成 MCP server | [server/src/sdk-mcp-server.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/sdk-mcp-server.ts) |
| 上下文压缩 | ✅ auto-compact + micro-compact | [server/src/utils/compact.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/utils/compact.ts) |
| 重试与错误分类 | ✅ 指数退避 + prompt-too-long / rate-limit / auth 错误识别 | [server/src/utils/retry.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/utils/retry.ts) |
| Token 估算与成本统计 | ✅ 按模型单价计算 USD | [server/src/utils/tokens.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/utils/tokens.ts) |
| 权限模式 | ✅ allow/deny/bypassPermissions | [server/src/types.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/types.ts) |
| HTTP/SSE 服务端示例 | ✅ 已有 web chat server | [server/examples/web/server.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/examples/web/server.ts) |

**结论**：原方案中需要从零编写的 `server/pkg/mcp/`、agent loop、工具集、skill 系统全部已有现成实现。工作量从"大"降到"中"。

## 修订后的目标

新增 `provider = "codeany"` 的 agent runtime，daemon 通过 Node 子进程跑 `@codeany/open-agent-sdk`，复用其完整 agent loop + MCP + 工具生态，**不依赖任何外部 CLI**（Claude/Hermes/Codex）。

## 非目标

- 不替换现有 17 个子进程 backend；`codeany` 是并列新 provider
- 不把 Go daemon 改成 Node daemon（与"server/daemon TS 重写"方案解耦，可独立落地）
- 不在 Go 进程内跑 TS（不做 WASM 桥接）

## 集成架构

```
┌─ daemon (Go) ──────────────────────────────────────────────────────────┐
│                                                                         │
│  runTask(provider="codeany")                                            │
│   │                                                                     │
│   ▼                                                                     │
│  agent.New("codeany", Config{WorkDir, McpConfig, Env, Model, ...})      │
│   │                                                                     │
│   ▼                                                                     │
│  codeanyBackend.Execute(ctx, prompt, opts)   ← 实现 agent.Backend      │
│   │                                                                     │
│   │  exec.CommandContext("node", "runner.mjs", "--stdio-json")          │
│   │  stdin: {prompt, opts, mcpConfig, env, model, sessionId, ...}       │
│   │  stdout: NDJSON stream of SDKMessage events                         │
│   │                                                                     │
│   ▼                                                                     │
│  ┌─ node runner.mjs (子进程) ───────────────────────────────────────┐  │
│  │                                                                  │  │
│  │  import { createAgent } from '@codeany/open-agent-sdk'           │  │
│  │                                                                  │  │
│  │  const agent = createAgent({                                     │  │
│  │    model: opts.model,                                            │  │
│  │    apiType: opts.apiType,           // auto-detect if absent     │  │
│  │    apiKey: opts.apiKey,             // 从 daemon 注入            │  │
│  │    baseURL: opts.baseURL,                                        │  │
│  │    cwd: opts.cwd,                                                │  │
│  │    maxTurns: opts.maxTurns,                                      │  │
│  │    mcpServers: opts.mcpServers,    // Claude-style mcpServers    │  │
│  │    permissionMode: 'bypassPermissions',                          │  │
│  │    sessionId: opts.resumeSessionId,  // resume                   │  │
│  │    tools: ['bash','edit','read','write','glob','grep',           │  │
│  │            'web-fetch','web-search','task-create', ...],         │  │
│  │    allowedTools: [...],              // 白名单（含 mcp__* 前缀） │  │
│  │    hooks: opts.hooks,                // 可选                      │  │
│  │  })                                                             │  │
│  │                                                                  │  │
│  │  for await (const ev of agent.query(opts.prompt)) {              │  │
│  │    process.stdout.write(JSON.stringify(ev) + '\n')               │  │
│  │  }                                                               │  │
│  │                                                                  │  │
│  │  // 进程退出 = 终态                                              │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│   │                                                                     │
│   ▼                                                                     │
│  stream-json parser (复用 openclaw 的 processOutput 模式)               │
│   │                                                                     │
│   ▼                                                                     │
│  Message{Type, Content, Tool, CallID, Input, Output}  →  msgCh          │
│  Result{Status, Output, Usage, SessionID, ResumeRejected} → resCh       │
│                                                                         │
│  Session{Messages, Result}  →  executeAndDrain (unchanged)              │
└─────────────────────────────────────────────────────────────────────────┘
```

**关键洞察**：daemon 已经有完整的 stream-json subprocess 模式（[openclaw.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/openclaw.go)、[qwen.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/qwen.go)），codeany backend 复用同一模式，只是子进程从"openclaw CLI"换成"node runner.mjs"。子进程内才是真正的 in-process agent loop。

## 与之前 Go 重写方案的对比

| 维度 | 旧方案（Go 重写） | 新方案（复用 TS SDK） |
|---|---|---|
| MCP client | 从零写 `server/pkg/mcp/`（5+ 文件） | 直接用 `@modelcontextprotocol/sdk` |
| Agent loop | 从零写 builtin.go | 复用 `QueryEngine` |
| 内置工具 | 第二期才加 | 立即拥有 30+ 工具 |
| LLM provider 抽象 | 从零写 | 复用 `providers/`（Anthropic + OpenAI） |
| Skill 系统 | 从零写 | 复用 `skills/`（含 5 个 bundled） |
| Session resume | 从零写文件存储 | 复用 `session.ts` |
| 上下文压缩 | 不在范围 | 复用 `utils/compact.ts` |
| 重试/错误分类 | 不在范围 | 复用 `utils/retry.ts` |
| 工作量 | 大（新写 10+ 文件） | 中（新写 1 Go 文件 + 1 TS runner） |
| 维护 | 与现有 backend 同语言 | TS SDK 独立升级 |
| 子进程开销 | 无（真 in-process） | 有（Node 子进程，但子进程内是完整 loop） |

新方案的"子进程"与 Claude/Hermes 的子进程本质不同：Claude/Hermes 子进程是第三方 CLI（外部依赖），codeany 子进程是项目自己的 TS SDK（可控）。

## 详细设计

### 1. 新 Go 文件：`server/pkg/agent/codeany.go`

参照 [openclaw.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/openclaw.go) 的模式，spawn `node` 跑 runner 脚本，stdin/stdout 用 NDJSON。

```go
// codeanyBackend implements Backend by spawning a Node subprocess that runs
// the @codeany/open-agent-sdk in-process agent loop. Communication is NDJSON
// over stdin/stdout, mirroring the openclaw stream-json pattern.
type codeanyBackend struct {
    cfg Config
}

func (b *codeanyBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
    runnerPath := b.cfg.ExecutablePath  // path to runner.mjs
    if runnerPath == "" {
        runnerPath = defaultCodeanyRunnerPath()
    }
    if _, err := exec.LookPath("node"); err != nil {
        return nil, fmt.Errorf("node executable not found: %w", err)
    }

    runCtx, cancel := runContext(ctx, opts.Timeout)

    args := []string{runnerPath, "--stdio-json"}
    cmd := exec.CommandContext(runCtx, "node", args...)
    hideAgentWindow(cmd)
    cmd.WaitDelay = 10 * time.Second
    if opts.Cwd != "" {
        cmd.Dir = opts.Cwd
    }
    cmd.Env = buildEnv(b.cfg.Env)

    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = newLogWriter(b.cfg.Logger, "[codeany:stderr] ")

    if err := cmd.Start(); err != nil {
        cancel()
        return nil, fmt.Errorf("start codeany runner: %w", err)
    }

    // 发送初始配置（一次写入，后续如需多 turn 可复用 stdin）
    initMsg := codeanyInitRequest{
        Prompt:          prompt,
        Model:           opts.Model,
        SystemPrompt:    opts.SystemPrompt,
        Cwd:             opts.Cwd,
        MaxTurns:        opts.MaxTurns,
        McpConfig:       opts.McpConfig,
        ResumeSessionID: opts.ResumeSessionID,
        ApiKey:          b.cfg.LLMAPIKey,
        BaseURL:         b.cfg.LLMBaseURL,
        ApiType:         b.cfg.LLMApiType,
    }
    if err := json.NewEncoder(stdin).Encode(initMsg); err != nil {
        cancel()
        return nil, fmt.Errorf("write codeany init: %w", err)
    }
    stdin.Close()  // 单 turn 模式：发完即关

    msgCh := make(chan Message, 256)
    resCh := make(chan Result, 1)

    go func() {
        defer cancel()
        defer close(msgCh)
        defer close(resCh)

        startTime := time.Now()
        scanResult := b.processOutput(stdout, msgCh)

        exitErr := cmd.Wait()
        duration := time.Since(startTime)

        if runCtx.Err() == context.DeadlineExceeded {
            scanResult.status = "timeout"
        } else if runCtx.Err() == context.Canceled {
            scanResult.status = "aborted"
        } else if exitErr != nil && scanResult.status == "completed" {
            scanResult.status = "failed"
            scanResult.errMsg = fmt.Sprintf("node runner exited: %v", exitErr)
        }

        resCh <- Result{
            Status:     scanResult.status,
            Output:     scanResult.output,
            Error:      scanResult.errMsg,
            DurationMs: duration.Milliseconds(),
            SessionID:  scanResult.sessionID,
            Usage:      scanResult.usage,
        }
    }()

    return &Session{Messages: msgCh, Result: resCh}, nil
}
```

`processOutput` 直接复用 openclaw 的 NDJSON 解析模式：每行一个 JSON 事件，按 `type` 字段分发到 `MessageText` / `MessageToolUse` / `MessageToolResult` / `MessageError`。

SDK 的 [SDKMessage](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/types.ts#L55-L100) 类型与 chimii `Message` 的映射：

| SDK 事件 | chimii Message |
|---|---|
| `{type:"assistant", message:{content:[{type:"text", text}]}}` | `Message{Type: MessageText, Content: text}` |
| `{type:"assistant", message:{content:[{type:"thinking", thinking}]}}` | `Message{Type: MessageThinking, Content: thinking}` |
| `{type:"assistant", message:{content:[{type:"tool_use", id, name, input}]}}` | `Message{Type: MessageToolUse, Tool: name, CallID: id, Input: input}` |
| `{type:"tool_result", result:{tool_use_id, tool_name, output}}` | `Message{Type: MessageToolResult, Tool: tool_name, CallID: tool_use_id, Output: output}` |
| `{type:"result", subtype, session_id, usage, total_cost_usd}` | 终态：`Result{Status, Output, Usage, SessionID}` |
| `{type:"result", subtype:"error_max_turns"}` | `Result{Status: "failed"}` |
| `{type:"result", subtype:"error_during_execution"}` | `Result{Status: "failed"}` |

### 2. 新 TS runner：`server/src/runner-stdio.ts`

一个极薄的入口脚本（约 60 行），把 stdin 的 JSON 配置喂给 `createAgent()`，把 `agent.query()` 的 async iterator 逐条写到 stdout。

```typescript
#!/usr/bin/env node
/**
 * Stdio runner for @codeany/open-agent-sdk.
 *
 * Protocol:
 *   stdin:  one JSON line with {prompt, model, mcpServers, ...}
 *   stdout: NDJSON stream of SDKMessage events (assistant / tool_result / result)
 *   stderr: human-readable logs (daemon captures via log writer)
 *
 * Exit codes:
 *   0 — run completed (terminal "result" event emitted)
 *   1 — startup or runtime error
 */
import { createAgent } from './index.js'
import { readBody } from './utils/messages.js'  // or inline read

async function main() {
  const raw = await readBody(process.stdin)
  const opts = JSON.parse(raw)

  const agent = createAgent({
    model: opts.model || process.env.CODEANY_MODEL,
    apiType: opts.apiType,
    apiKey: opts.apiKey || process.env.CODEANY_API_KEY,
    baseURL: opts.baseURL || process.env.CODEANY_BASE_URL,
    cwd: opts.cwd,
    maxTurns: opts.maxTurns || 20,
    mcpServers: opts.mcpConfig,
    permissionMode: 'bypassPermissions',
    sessionId: opts.resumeSessionId,
    systemPrompt: opts.systemPrompt,
    tools: ['bash','edit','read','write','glob','grep',
            'web-fetch','web-search','task-create','task-list',
            'task-update','agent','todo','skill'],
    // allowedTools 可由 daemon 注入白名单
  })

  try {
    for await (const ev of agent.query(opts.prompt)) {
      process.stdout.write(JSON.stringify(ev) + '\n')
    }
  } catch (err) {
    process.stdout.write(JSON.stringify({
      type: 'result',
      subtype: 'error_during_execution',
      is_error: true,
      result: err.message,
    }) + '\n')
    process.exit(1)
  } finally {
    await agent.close()
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
```

构建：在 `server/package.json` 的 `scripts` 加一行 `"build:runner": "tsc && esbuild src/runner-stdio.ts --bundle --platform=node --format=esm --outfile=dist/runner-stdio.mjs"`，产出一个自包含的 `dist/runner-stdio.mjs`，daemon 通过 `node dist/runner-stdio.mjs --stdio-json` 启动。

### 3. 注册到 agent 工厂

[agent.go:295-338](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go#L295-L338) 的 `New` switch 加 case：

```go
case "codeany":
    return &codeanyBackend{cfg: cfg}, nil
```

同步更新（与旧方案一致）：
- [SupportedTypes](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go#L236-L254) 加 `"codeany"`
- [launchHeaders](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go#L351-L369) 加 `"codeany": "open-agent-sdk (Node + MCP)"`
- 新 migration 放宽 `runtime_profile.protocol_family` CHECK 约束加 `'codeany'`（参照 migrations/134/136/175/179/202 模板）

### 4. 扩展 `agent.Config`

[agent.go:208-217](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go#L208-L217) 加入 daemon 注入的 LLM 凭证：

```go
type Config struct {
    // ... 现有字段保留
    LLMAPIKey  string  // codeany 专用：CODEANY_API_KEY
    LLMBaseURL string  // codeany 专用：CODEANY_BASE_URL
    LLMApiType string  // codeany 专用：'anthropic-messages' | 'openai-completions'（空则按 model 名自动探测）
}
```

Model 字段已存在，codeany 复用为 LLM model 名。

### 5. Daemon 端接入

#### 5.1 LLM 凭证注入

- [daemon/config.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/config.go) 加 `LLMAPIKey` / `LLMBaseURL` / `LLMApiType` 字段
- [cmd_daemon.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/cmd/chimii/cmd_daemon.go) 启动时从环境变量读取（推荐复用现有 `CHIMII_LLM_API_KEY` / `CHIMII_LLM_BASE_URL`，再加 `CHIMII_LLM_API_TYPE`）
- [daemon.go:4843](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go#L4843) 构造 `agent.Config` 时填入这三个字段

#### 5.2 探测路径

[probeBuiltinRuntime](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go#L1269) 对 `provider == "codeany"` 改为：
- 检查 `node` 在 PATH 中
- 检查 runner 脚本存在（`node -e "require('fs').existsSync('<runnerPath>')"` 或直接 `os.Stat`）
- 返回 `("open-agent-sdk <version>", true)`，version 可从 `server/package.json` 读

[detectBuiltinRuntimes](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go#L1353) 在 `node` 可用时产出 codeany runtime。

#### 5.3 `d.cfg.Agents["codeany"]` 合成 entry

```go
AgentEntry{
    Path:    runnerPath,   // dist/runner-stdio.mjs 的绝对路径
    Command: "node",
    Model:   d.cfg.LLMDynamicModel,
}
```

[resolveAgentEntry](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go#L487) 与 `layerCustomEnvAndHermesHome` / `configureCodexTaskShellEnvironment` 对 codeany 跳过。

#### 5.4 系统 prompt 注入

[providerNeedsInlineSystemPrompt](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go#L3908) 对 `codeany` 返回 `true`（与 openclaw 一致：runner 不读 cwd 的 AGENTS.md，runtime brief 必须进 prompt）。

#### 5.5 MCP 配置合并

直接复用 [mergeRuntimeAndAgentMcpConfig](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/runtime_mcp.go#L35)。codeany 的 runtime 级 MCP 配置文件读 `~/.chimii/mcp.json`（与旧方案一致）：

在 [runtime_mcp.go:84](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/runtime_mcp.go#L84) 加 `case "codeany"` 分支，读 `~/.chimii/mcp.json` 的 `mcpServers` 字段（与 Claude 的 `~/.claude.json` 对称）。

### 6. Skill 加载

[execenv/context.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/execenv/context.go) 的 `writeContextFiles` 对 `codeany` 写到默认路径 `{workDir}/.agent_context/skills/{name}/SKILL.md`。

但 **TS SDK 自带 5 个 bundled skills**（commit/debug/review/simplify/test）——这些 skills 不需要 daemon 写文件，runner 启动时 `initBundledSkills()` 自动注册。daemon 写入的是 workspace 自定义 skills，两者在 `SkillTool` 中合并可见。

### 7. Session resume

**完全复用 TS SDK 的 session 持久化**，无需 Go 侧自管：

- `Result.SessionID` 来自 SDK 的 `result.session_id`（`agent.query()` 的终态事件携带）
- 下次 task 带 `ResumeSessionID` 时，runner 通过 `createAgent({sessionId: ...})` 传入
- SDK 的 [session.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/session.ts) 自动从 `~/.open-agent-sdk/sessions/<id>/` 读取历史
- resume 失败（session 文件不存在/损坏）→ SDK 退化为 fresh session，runner 在 result 事件中带 `subtype: "success"` 但 `session_id` 变化 → daemon 侧检测 session_id 变化设 `ResumeRejected = true`

**注意**：SDK 默认 session 目录是 `~/.open-agent-sdk/sessions/`。若要按 workspace 隔离，可在 runner 里设置 `process.env.HOME = opts.cwd` 或加一个 `CODEANY_SESSION_DIR` 环境变量（需要 SDK 支持，是上游 PR 候选）。

### 8. 权限模型

runner 固定用 `permissionMode: 'bypassPermissions'`，与 Claude 的 `--permission-mode bypassPermissions` 和 Hermes 的 `HERMES_YOLO_MODE=1` 对齐。

SDK 的权限系统（[types.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/types.ts) 的 `PermissionMode`）支持更细粒度的 `allow/deny` 模式，第二期可接入 per-tool 白名单。

### 9. 成本与 usage 统计

SDK 的 `result` 事件携带：

```typescript
{
  type: 'result',
  subtype: 'success',
  usage: { input_tokens, output_tokens, cache_read_input_tokens?, cache_creation_input_tokens? },
  total_cost_usd: number,
  model_usage?: Record<string, { input_tokens, output_tokens }>,  // 多模型场景
}
```

Go 侧 `processOutput` 把 `usage` + `model_usage` 转为 `map[string]TokenUsage`（key 是 model 名，单模型场景用 `opts.Model`）。`total_cost_usd` 可记到 `Result.CostUSDTicks`（若 [agent.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go) 的 Result 结构支持，否则记到日志）。

### 10. 工具白名单与命名空间

SDK 的工具命名空间：
- 内置工具：`bash`、`edit`、`read`、`write`、`glob`、`grep`、`web_fetch`、`web_search`、`task_create`、`agent`、`todo`、`skill`、...
- MCP 工具：`mcp__<server_name>__<tool_name>`（见 [mcp/client.ts:94](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/mcp/client.ts#L94)）
- SDK MCP server 工具：`mcp__<server_name>__<tool_name>`（同上，in-process）

daemon 侧通过 `allowedTools` 字段注入白名单，例：
```json
{
  "allowedTools": ["bash", "edit", "read", "write", "mcp__github__*"]
}
```

`*` 通配符由 SDK 的 `filterTools` 支持。

## 实施阶段

### Phase 1: TS runner 与构建（独立，不依赖 daemon）
- 新建 [server/src/runner-stdio.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/runner-stdio.ts)
- 在 [server/package.json](file:///Users/chunjun/Desktop/Build/Games/chimii/server/package.json) 加 `build:runner` 脚本
- 单元测试：mock stdin，验证 stdout NDJSON 流
- 验收：`echo '{"prompt":"hi","model":"gpt-4o","apiKey":"..."}' | node dist/runner-stdio.mjs` 输出合法 NDJSON

### Phase 2: Go codeany backend
- 新建 [server/pkg/agent/codeany.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/codeany.go)
- 实现 `Backend` 接口 + NDJSON 解析（参照 [openclaw.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/openclaw.go)）
- 单元测试：mock node 子进程（用 echo 脚本回放 fixture NDJSON）
- 验收：`agent.New("codeany", ...)` 返回非 nil，单元测试全绿

### Phase 3: agent 工厂与白名单
- [agent.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go) 注册 case + SupportedTypes + launchHeaders
- 新 migration 放宽 `protocol_family` CHECK 加 `'codeany'`
- 验收：`agent.IsSupportedType("codeany") == true`

### Phase 4: daemon 集成
- daemon Config 注入 LLM 凭证
- `detectBuiltinRuntimes` / `probeBuiltinRuntime` 处理 codeany
- `runTask` 合成 AgentEntry + 跳过子进程无关步骤
- `providerNeedsInlineSystemPrompt` 加 codeany
- runtime_mcp 加 `case "codeany"`
- 验收：daemon 启动后自动注册 codeany runtime

### Phase 5: e2e 验证
- 配置真实 LLM（OpenAI 或 Anthropic）+ 一个 stdio MCP server（如 `@modelcontextprotocol/server-filesystem`）
- 创建 agent，`provider = "codeany"`，挂载 MCP config
- 创建 issue assign 给该 agent
- 验证：task 派发 → runner 启动 → LLM 调用 → MCP 工具调用 → 回复写入 issue
- 验证：session resume（同 issue 第二次 task 延续上下文）
- 验证：usage 与 cost 统计正确
- 验收：完整 agent loop 端到端跑通

## 复用清单

### 直接复用（无修改）
- [server/src/engine.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/engine.ts) — agent loop
- [server/src/mcp/client.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/mcp/client.ts) — MCP client
- [server/src/providers/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/providers/index.ts) — LLM provider
- [server/src/tools/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/tools/index.ts) — 30+ 内置工具
- [server/src/skills/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/skills/index.ts) — skill 系统
- [server/src/session.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/session.ts) — session 持久化
- [server/src/hooks.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/hooks.ts) — hook 系统
- [server/src/utils/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/utils/compact.ts) — 压缩/重试/token
- [server/pkg/agent/openclaw.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/openclaw.go) — subprocess 模式参考
- [server/internal/daemon/runtime_mcp.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/runtime_mcp.go) — MCP config 合并

### 新增
- `server/src/runner-stdio.ts` — TS runner 入口（~60 行）
- `server/pkg/agent/codeany.go` — Go backend（~300 行，参照 openclaw.go 700 行精简）
- `server/pkg/agent/codeany_test.go` — 单元测试
- `server/migrations/XXX_runtime_profile_codeany.up.sql` — CHECK 约束放宽
- `server/migrations/XXX_runtime_profile_codeany.down.sql`

### 修改
- [server/package.json](file:///Users/chunjun/Desktop/Build/Games/chimii/server/package.json) — 加 `build:runner` 脚本
- [server/pkg/agent/agent.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go) — `New` switch、`SupportedTypes`、`launchHeaders`、`Config` 加 LLM 字段
- [server/internal/daemon/config.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/config.go) — 加 LLM 凭证字段
- [server/cmd/chimii/cmd_daemon.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/cmd/chimii/cmd_daemon.go) — 构造 LLM 凭证
- [server/internal/daemon/daemon.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/daemon.go) — `detectBuiltinRuntimes`/`probeBuiltinRuntime`/`runTask`/`providerNeedsInlineSystemPrompt` 特殊处理
- [server/internal/daemon/runtime_mcp.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/runtime_mcp.go) — 加 `case "codeany"` 读 `~/.chimii/mcp.json`

## 风险与缓解

| 风险 | 严重度 | 缓解 |
|---|---|---|
| Node 运行时未安装 | 高 | daemon 探测时检查 `node` 在 PATH；用户引导安装 Node 18+ |
| runner 脚本路径发现 | 中 | daemon 启动时定位 `server/dist/runner-stdio.mjs`；支持 `CHIMII_CODEANY_RUNNER_PATH` 环境变量覆盖 |
| Node 子进程冷启动慢 | 中 | Node 18+ 启动 ~50ms；runner 体积小（esbuild bundle 后 < 500KB）；可接受 |
| MCP server 子进程泄漏 | 高 | runner 在 `finally` 中 `agent.close()`，SDK 内部会关闭 MCP 连接；daemon 侧进程组终止兜底 |
| Session 目录污染 | 中 | 第一期用默认 `~/.open-agent-sdk/sessions/`；后续给 SDK 加 `CODEANY_SESSION_DIR` 环境变量支持 |
| TS SDK 版本漂移 | 低 | `server/package.json` 锁定版本；runner 是 thin wrapper，API 变化影响小 |
| 与 TS 重写方案的关系 | 低 | 完全兼容：TS 重写后 daemon 本身就是 Node，可去掉子进程直接 in-process 调用 SDK |
| 工具权限粒度 | 中 | 第一期 `bypassPermissions`；第二期接 `allowedTools` 白名单 + per-tool deny |

## 与 TS 重写方案的衔接

codeany backend 的 Go 子进程模式是**过渡方案**：

1. **当前**：Go daemon + Node 子进程跑 TS SDK（本方案）
2. **TS 重写后**：daemon 本身是 Node，直接 `import { createAgent } from '@codeany/open-agent-sdk'`，去掉子进程边界，真正 in-process

过渡方案的代码资产全部可复用：
- `runner-stdio.ts` → 演化为 daemon 的内部模块
- `codeany.go` → 弃用（Go daemon 退役）
- MCP config 合并、skill 写入、探测逻辑 → 移植到 TS

因此本方案不会产生"废弃代码"，是 TS 重写前的合理里程碑。

## 开放问题

1. **runner 分发方式**：随 daemon 一起发布（`server/dist/` 打包到 release），还是要求用户先 `npm install`？建议前者：CI 构建 runner.mjs 并打入 Go release artifact
2. **多 LLM 凭证**：当前方案假设全局一组 `CHIMII_LLM_*`。是否需要 per-agent 凭证（agent A 用 OpenAI，agent B 用 Anthropic）？建议第二期加 agent-level `llm_config` 字段
3. **SDK 版本升级策略**：`@codeany/open-agent-sdk` 是独立开源项目，升级时如何测试兼容性？建议加 e2e 测试覆盖关键路径（chat/tool-call/mcp/resume）
4. **工具集裁剪**：默认启用全部 30+ 工具可能过重（bash 在某些环境不安全）。是否需要 per-workspace 工具白名单？建议第一期用 `allowedTools` env var，第二期接 UI
5. **上下文压缩阈值**：SDK 默认按模型 context window 自动压缩。是否需要 daemon 侧覆盖？建议第一期用 SDK 默认值

## 验收标准

1. `agent.New("codeany", ...)` 返回非 nil backend
2. `agent.IsSupportedType("codeany") == true`
3. `runtime_profile` 表 `protocol_family = 'codeany'` 可写入
4. daemon 启动后（node 在 PATH 中）自动注册 codeany runtime
5. codeany backend 单元测试覆盖：
   - 纯对话（无工具）
   - 单工具调用（bash）
   - MCP 工具调用（mock stdio MCP server）
   - session resume 成功
   - runner 崩溃后状态正确（failed + 错误消息）
   - ctx 取消（SIGTERM 透传到 node 子进程）
6. runner 单元测试覆盖：
   - stdin JSON 解析
   - stdout NDJSON 流式输出
   - 错误场景输出 `result.subtype: "error_during_execution"`
   - MCP server 连接失败不阻塞启动（warn 日志 + 跳过该 server）
7. e2e：真实 LLM + 真实 MCP server，task 端到端跑通
8. 成本统计：`Result.Usage` 非空且与 SDK `result.usage` 一致
9. 不影响其他 17 个 backend 的现有行为（回归测试全绿）

## 参考

- [`@codeany/open-agent-sdk` GitHub](https://github.com/codeany-ai/open-agent-sdk-typescript)
- [MCP 2025-03-26 spec](https://modelcontextprotocol.io/specification/2025-03-26)
- [server/src/index.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/index.ts) — SDK 入口
- [server/src/engine.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/engine.ts) — agent loop
- [server/src/mcp/client.ts](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/mcp/client.ts) — MCP client
- [server/src/providers/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/providers/index.ts) — LLM provider
- [server/src/tools/](file:///Users/chunjun/Desktop/Build/Games/chimii/server/src/tools/index.ts) — 内置工具
- [server/pkg/agent/openclaw.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/openclaw.go) — subprocess backend 参考
- [server/pkg/agent/agent.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/pkg/agent/agent.go) — Backend 契约
- [server/internal/daemon/runtime_mcp.go](file:///Users/chunjun/Desktop/Build/Games/chimii/server/internal/daemon/runtime_mcp.go) — MCP config 合并
