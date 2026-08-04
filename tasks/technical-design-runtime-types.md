# CLI / Cloud Runtime 可切换执行方案

> 状态：Proposed
>
> 决策：采用“方案 A”——CLI runtime 与 Cloud runtime 是两种并列、显式选择的执行类型，绝不在两者之间自动 fallback。
>
> 默认：CLI runtime。Cloud runtime 需由运维配置显式启用，首期仅支持 `anthropic` / `openai`。

## 1. 目标与边界

目标执行拓扑：

```text
runtime 执行类型
├── CLI runtime    本地或云主机 daemon + server/pkg/agent
│                  默认；保留当前全部 CLI provider 覆盖
└── Cloud runtime  API server worker + server/runtime
                   可选；仅 anthropic/openai；Chimii 托管 API 密钥与费用
```

必须满足的不变量：

1. 一个 `agent_runtime` 只属于一种执行类型。
2. 一个 agent 通过 `runtime_id` 显式绑定执行类型。
3. 一个 task 从入队到终态不允许在 CLI / Cloud 之间切换。
4. Cloud 失败只能在同一 Cloud runtime 上按平台重试规则重试，不能改由 CLI 重跑；CLI 亦然。
5. CLI session 与 Cloud session 不转换、不共享、不互相恢复。
6. 配置修改只改变“新建时的可选项和默认项”，不静默迁移已有 agent/task。

不在本方案中实现：

- SDK 失败后 fallback 到 CLI。
- CLI session 与 Cloud session 的格式转换。
- 让 `server/runtime` 成为 daemon 的一个 provider backend。
- 根据 model/provider 在运行中隐式切换 runtime 类型。
- 首期开放 Anthropic/OpenAI 之外的 OpenAI-compatible gateway。

## 2. 当前代码事实与必须修正的偏差

### 2.1 可复用的基础

- `agent_runtime.runtime_mode` 已有 `local | cloud`。
- agent 已通过 `runtime_id` 绑定 runtime，task 入队时会固化 `runtime_id`。
- `TaskService.ClaimTaskForRuntime` / batch claim 已有原子 claim、agent 并发限制、丢失 claim 恢复与重试机制。
- task start/progress/message/usage/complete/fail/cancel/session pin 的服务端数据模型已经完整。
- claim 响应已覆盖 issue/chat/autopilot/quick-create、skill、MCP、repo、workspace context 与任务 token。
- prior session 查询已要求 prior task 与 current task 的 `runtime_id` 一致，可防止 runtime 切换后误恢复会话。
- 前端 runtime picker 和 API 类型已能识别 local/cloud，可在此基础上增加执行类型。

### 2.2 “cloud”已有两种含义，不能直接复用 `runtime_mode`

现有 `CHIMII_CLOUD_FLEET_URL` 与 `/api/cloud-runtime/nodes` 管理的是云主机节点。这些节点仍运行 daemon + `pkg/agent` + CLI，只是部署位置在云上，不是本方案的 SDK Cloud runtime。

因此需要把两个维度分开：

| 维度 | 字段 | 取值 | 含义 |
| --- | --- | --- | --- |
| 部署位置 | 现有 `runtime_mode` | `local | cloud` | daemon/工作机在用户本地还是云主机 |
| 执行引擎 | 新增 `execution_type` | `cli | cloud` | 使用 `pkg/agent` CLI，还是 server-side `server/runtime` SDK |

合法组合：

| `runtime_mode` | `execution_type` | 执行方式 | 状态 |
| --- | --- | --- | --- |
| `local` | `cli` | 用户本地 daemon + CLI | 现有、默认 |
| `cloud` | `cli` | 现有 Fleet 云节点 daemon + CLI | 现有、保留 |
| `cloud` | `cloud` | API server worker + Open Agent SDK | 本方案新增 |
| `local` | `cloud` | 无意义 | API/DB 禁止 |

API 和 UI 面向用户使用“CLI runtime / Cloud runtime”。`runtime_mode` 保留为兼容现有 Fleet 和客户端的 placement 字段，不再用它决定执行引擎。

### 2.3 当前 `openagent` 接入位置与方案 A 相反

当前代码已存在：

- `server/pkg/agent/openagent.go`：在 daemon 进程内运行 SDK。
- `agent.SupportedTypes` / `agent.New()` 中的 `openagent`。
- migration 233：把 `openagent` 加入 CLI custom runtime profile 协议族。

该路径还有三个生产阻断问题：

1. 忽略 `ResumeSessionID`，没有持久化 SDK 会话。
2. 使用 `PermissionModeBypassPermissions`。
3. SDK `Bash` 直接在宿主进程启动 shell；当前 sandbox/permissions 实现不是 fail-closed。

方案实施的第一步必须是移除这条 daemon SDK 路径，而不是在其上继续扩展。

## 3. 配置合同

### 3.1 服务端配置

```dotenv
# 新建 runtime / onboarding 的默认执行类型。
# 仅影响新选择，不迁移已有 agent。
CHIMII_RUNTIME_DEFAULT=cli

# Cloud runtime 默认关闭。
CHIMII_CLOUD_RUNTIME_ENABLED=false

# 启用后允许创建的 provider 白名单；首期只允许这两个枚举。
CHIMII_CLOUD_RUNTIME_PROVIDERS=anthropic,openai

# Cloud worker 资源与安全边界。
CHIMII_CLOUD_RUNTIME_MAX_CONCURRENT_TASKS=8
CHIMII_CLOUD_RUNTIME_WORK_ROOT=/var/lib/chimii/cloud-runtime
CHIMII_CLOUD_RUNTIME_TASK_TIMEOUT=2h
CHIMII_CLOUD_RUNTIME_IDLE_TIMEOUT=30m

# Anthropic。未列入 providers 时即使配了 key 也不启用。
CHIMII_CLOUD_RUNTIME_ANTHROPIC_API_KEY=
CHIMII_CLOUD_RUNTIME_ANTHROPIC_BASE_URL=https://api.anthropic.com
CHIMII_CLOUD_RUNTIME_ANTHROPIC_DEFAULT_MODEL=

# OpenAI。
CHIMII_CLOUD_RUNTIME_OPENAI_API_KEY=
CHIMII_CLOUD_RUNTIME_OPENAI_BASE_URL=https://api.openai.com/v1
CHIMII_CLOUD_RUNTIME_OPENAI_DEFAULT_MODEL=
```

配置解析规则：

- `CHIMII_RUNTIME_DEFAULT` 仅接受 `cli | cloud`，默认 `cli`。
- `default=cloud` 时，Cloud runtime 必须已启用且至少有一个 provider 密钥，否则 server 启动失败。
- Cloud 开关关闭时，server 不启动 Cloud worker，API 不允许新建/绑定 Cloud runtime，但保留已有数据供管理员处理。
- Cloud 开关不作为运行中 kill switch；优雅关机先取消/结束本进程的正在执行任务，不将它们转给 CLI。
- 密钥只来自 server secret/environment，不进入 `agent_runtime.metadata`、`agent.runtime_config`、API 响应或日志。
- 不复用 `CHIMII_LLM_*`。该配置属于 server 内部轻量 LLM helper；Cloud runtime 需独立的费用、并发、模型和安全边界。
- 现有 `CHIMII_CLOUD_FLEET_URL` 保持不变，它管理的是 Cloud-hosted CLI 节点，不是 SDK runtime。

切换到 Cloud 作为新建默认值：

```dotenv
CHIMII_CLOUD_RUNTIME_ENABLED=true
CHIMII_RUNTIME_DEFAULT=cloud
CHIMII_CLOUD_RUNTIME_PROVIDERS=anthropic,openai
```

这个切换不会修改现有 agent 的 `runtime_id`。要切换已有 agent，用户必须在 UI/API 中选择新 runtime；切换后的第一个 task 强制开新 session。

### 3.2 前端公开配置

`GET /api/config` 新增：

```json
{
  "runtime": {
    "default_type": "cli",
    "enabled_types": ["cli", "cloud"],
    "cloud_providers": ["anthropic", "openai"]
  }
}
```

这些字段只表达能力，不泄露 key、base URL 或计费配置。前端用 `parseWithFallback` + zod schema 解析；旧 server 缺少字段时必须回退为 CLI-only。

## 4. 数据模型

### 4.1 `agent_runtime`

新增：

```sql
execution_type TEXT NOT NULL DEFAULT 'cli'
  CHECK (execution_type IN ('cli', 'cloud'))
```

数据迁移：

- 所有现有 runtime，包括 `runtime_mode='cloud'` 的 Fleet 节点，全部 backfill 为 `execution_type='cli'`。
- Cloud SDK runtime 由新 API 创建：`runtime_mode='cloud'` + `execution_type='cloud'` + `daemon_id=NULL`。
- Cloud runtime provider 只能为 `anthropic | openai`，由 handler/service 应用层校验。
- 不添加外键或 cascade。
- 新索引使用 `CREATE [UNIQUE] INDEX CONCURRENTLY`，且每个索引单独一个 migration 文件。

建议索引：

```sql
CREATE INDEX CONCURRENTLY agent_runtime_execution_status_idx
  ON agent_runtime (execution_type, status, workspace_id);
```

### 4.2 Cloud session

不把 SDK JSONL 会话存在 server 本地磁盘，因为 API server 可以是多副本且可以随时替换。新增持久化表：

```text
cloud_runtime_session
  id, runtime_id, agent_id, provider, model,
  context_type(issue|chat|autopilot|quick_create), context_id,
  status, created_at, updated_at

cloud_runtime_session_message
  session_id, seq, role, payload(jsonb), created_at
```

约束：

- 不使用 DB foreign key；删除 runtime/agent/workspace 时由应用服务在同一事务中清理。
- session id 使用 `crs_<uuid>` 前缀，防止与 CLI 原生 session id 混淆。
- `server/runtime/agent` 增加可注入的 `SessionStore`，每轮 API/tool 完成后增量持久化，不等 task 终态才一次性写入。
- 在第一次 LLM API 请求前创建 session 并调用共享 task lifecycle 服务 pin 到 `agent_task_queue.session_id`。
- resume 只加载同一 `runtime_id` 下的 `crs_` session；对 CLI session id 或其他 runtime 的 session 直接拒绝，不做格式猜测。

## 5. 服务端组件设计

### 5.1 目标目录

```text
server/internal/runtimeconfig/
  config.go                 # env 解析、严格校验、公开 capability 投影

server/internal/cloudfleet/
  client.go                 # 由现 internal/cloudruntime 改名；仅表示现有 Fleet 代理

server/internal/cloudruntime/
  manager.go                # worker 启停、并发、优雅关机
  registry.go               # Cloud runtime 创建/状态/provider 校验
  executor.go               # TaskRun -> server/runtime agent.Options
  messages.go               # SDKMessage -> task message
  sessions.go               # PostgreSQL SessionStore adapter
  workdir.go                # 任务工作目录与 repo 准备
  errors.go                 # 稳定 failure_reason 分类

server/internal/taskexecution/
  coordinator.go            # claim/finalize/start/complete/fail 共享业务编排
  reporter.go               # progress/message/usage/session/cancel 内部接口
```

`internal/cloudruntime` 当前名称已被 Fleet HTTP client 占用。先将现有 client/handler 语义重命名为 `cloudfleet`，再把 `cloudruntime` 专用于 SDK 执行，避免新旧“cloud runtime”在同一 package 中继续混用。

### 5.2 共享 task 编排，不内部伪造 HTTP

现有 claim payload 构建与 lifecycle 大量实现在 `internal/handler/daemon.go`。Cloud worker 不应使用 loopback HTTP 、伪造 daemon token 或直接调 handler。

将业务逻辑下沉为可内部调用的服务：

```go
type ClaimedRun struct {
    Task       db.AgentTaskQueue
    Runtime    db.AgentRuntime
    Input      TaskRunInput
    AuthToken  string
}

type Coordinator interface {
    Claim(ctx context.Context, runtimeID pgtype.UUID) (*ClaimedRun, error)
    Start(ctx context.Context, run ClaimedRun) error
    PinSession(ctx context.Context, taskID, sessionID, workDir string) error
    AppendMessages(ctx context.Context, taskID string, messages []TaskMessage) error
    RecordUsage(ctx context.Context, taskID string, usage []TaskUsage) error
    Complete(ctx context.Context, taskID string, result TaskResult) error
    Fail(ctx context.Context, taskID string, failure TaskFailure) error
    Status(ctx context.Context, taskID string) (string, error)
}
```

- daemon HTTP handlers 改为“鉴权 + decode/encode + 调用 Coordinator”。
- Cloud worker 直接调 Coordinator。
- claim token 生成、comment receipt、workspace isolation、session resume 判断只保留一份实现。
- CLI daemon 线上协议与行为不变。

### 5.3 Cloud worker 生命周期

1. server 启动时解析 runtime 配置。
2. Cloud 关闭：不启动 worker，将 SDK runtime 视为不可选；CLI/Fleet 不受影响。
3. Cloud 开启：每个 API server 副本启动 worker pool。
4. worker 通过 DB/event bus 获知 `execution_type='cloud'` 的 queued task，并使用现有原子 claim 服务抢占。
5. 多副本可同时 claim；DB 的 `SKIP LOCKED`/状态条件是单点真相，不使用进程内锁做分布式互斥。
6. 先获取 global worker semaphore，再 claim，避免无空闲 slot 却把 task 变为 dispatched。
7. 准备工作目录与 sandbox，pin session/workdir，然后将 task 改为 running。
8. SDK event 增量转换为 task messages，使用与 daemon 相同的脱敏、持久化和 realtime 广播。
9. 执行期间监听 task cancel/terminal 事件，并保留低频 DB poll 作为丢消息保底。
10. 先 flush message/session/usage，再 complete/fail。
11. server 关机时 cancel 本副本执行中的 SDK context，将未进入稳定终态的 task 按同 Cloud runtime 的基础设施失败重试，绝不改投 CLI。

### 5.4 Provider 和 model

- runtime 级 provider 只允许 `anthropic | openai`。
- API key/base URL 由 server 运维配置提供，不允许 workspace/agent 覆盖 key。
- agent 保留 model/thinking 选择，但必须通过 provider 对应的服务端白名单验证。
- `server/runtime/api` 不再依赖 key 前缀/model 名称猜 provider；Cloud executor 必须显式传 `Provider`。
- 任何 provider/model 错误都只在当前 Cloud runtime 上失败。不设置 SDK `FallbackModel`，以免引入另一层隐式降级语义。

## 6. 安全与工作目录

Cloud runtime 不得在当前 SDK 的宿主直执行模式下上线。生产启用前必须完成：

1. 每个 task 独立工作目录，路径必须位于 `CHIMII_CLOUD_RUNTIME_WORK_ROOT/<workspace>/<task>`。
2. 工具执行始终经过可注入 `SandboxExecutor`，不得由 `tools.Bash` 直接 `exec.Command("bash", "-c", ...)`。
3. 文件系统校验 fail-closed：路径不在 allowlist 内时返回拒绝，不能像当前 `ValidateWrite` 那样对非 allowlist 路径放行。
4. 防 symlink escape、`..` escape、绝对路径 escape、硬链接越界和 TOCTOU。
5. 默认不挂载 server 源码、server `.env`、Docker socket、SSH/AWS 用户目录或其他 task 工作目录。
6. 传给 tool 的 env 使用 allowlist 组装，不继承 API server 的完整 `os.Environ()`。LLM provider key 不进入 tool env。
7. 默认禁止宿主网络/云 metadata/loopback；仅通过明确 egress 策略访问允许的公网地址。
8. 限制 CPU、内存、进程数、磁盘、单命令超时、task 总超时和输出字节数。
9. `PermissionModeBypassPermissions` 不得用于 Cloud production executor；即使上层已校验，底层工具仍需实施沙箱约束。
10. `local_directory` 资源只能绑定 CLI runtime。Cloud runtime 仅处理 server 能够在沙箱中拉取的 repo/project 资源。

上述任一门禁未完成时，`CHIMII_CLOUD_RUNTIME_ENABLED=true` 必须启动失败，而不是记录 warning 后继续。

## 7. API 与前端

### 7.1 Cloud runtime 管理 API

新增用户 API：

```text
POST   /api/runtimes/cloud
GET    /api/runtimes/{runtimeId}
PATCH  /api/runtimes/{runtimeId}
DELETE /api/runtimes/{runtimeId}
```

`POST /api/runtimes/cloud` 请求示例：

```json
{
  "name": "Chimii Cloud - OpenAI",
  "provider": "openai",
  "visibility": "private"
}
```

服务端固定写入 `runtime_mode=cloud` / `execution_type=cloud` / `daemon_id=NULL`。客户端不能直接提交这三个基础设施字段。

现有 daemon register/upsert 固定写入 `execution_type=cli`。Cloud worker 永远不通过 daemon register/heartbeat API 注册自己。

### 7.2 UI

- Runtime 创建入口显示两个显式选项：“Connect CLI runtime”和“Create Cloud runtime”。
- Cloud 关闭时不显示 Cloud 创建入口，但已存在的 Cloud runtime 仍可显示为 unavailable 并可删除/管理。
- 现有“Cloud worker/node”对话框改名为“Cloud CLI machine”，与 SDK Cloud runtime 区分。
- Agent runtime picker 按 `execution_type` 分 CLI/Cloud，再在 CLI 内按 local/Fleet machine 分组。
- onboarding 不再硬编码“cloud 优先”，而是读取 `/api/config.runtime.default_type`。默认仍为 CLI。
- 把 agent 从 CLI 改绑到 Cloud，或从 Cloud 改绑到 CLI 时，UI 必须提示“将开始新会话，原 runtime 的上下文不会迁移”。

## 8. 失败、重试与取消

Cloud runtime 新增稳定的 `failure_reason`：

```text
cloud_runtime_unavailable
cloud_provider_auth
cloud_provider_rate_limited
cloud_provider_error
cloud_sandbox_violation
cloud_session_corrupt
cloud_workdir_prepare
cloud_runtime_timeout
```

分类规则：

- auth、sandbox violation、session corrupt：不自动重试。
- rate limit、短暂 provider 5xx、server worker shutdown：可以按现有平台重试机制在同一 Cloud runtime 上重试。
- 已经开始工具执行的任务不做 executor fallback。重试是新 task attempt，需沿用现有幂等/会话安全规则，不是把原 task 换引擎继续。
- cancel 会取消 SDK context 和沙箱内进程组，然后 flush 已产生 transcript，最后 ACK。

代码和测试中需有一条显式断言：Cloud executor 的错误处理不得引用 `pkg/agent.New()`、daemon claim API 或任何 CLI executable path。

## 9. 观测与计费

- 现有 `runtime_mode` 指标不足以区分 Fleet CLI 与 SDK Cloud；所有 runtime/task 指标新增低基数 label `execution_type=cli|cloud`。
- Cloud usage 沿用 `task_usage` 持久化，provider/model/token/cost 从 SDK result 映射。
- 费用归属由 `execution_type` 决定：CLI 为用户 CLI 账号；Cloud 为 Chimii 托管账号。一个 task 只能出现一种归属。
- 日志包含 `runtime_id/task_id/execution_type/provider/model`，不包含 API key、完整 prompt 或未脱敏 tool 输入。
- 最低指标：cloud claim wait、run duration、active workers、provider requests/status、token/cost、sandbox denials、session load/save latency、cancel latency。

## 10. 实施分阶段

### Phase 0：纠正执行边界

- 删除 `server/pkg/agent/openagent.go` 及其 daemon backend 测试。
- 从 `agent.New()`、`SupportedTypes`、launch header、custom runtime 列表中移除 `openagent`。
- 新 migration 将已有 `runtime_profile.protocol_family='openagent'` 设为 `enabled=false`，然后用新的 NOT VALID CHECK 禁止新增 `openagent`；不修改已发布的 migration 233。
- 保留 `server/runtime` module，但在 Cloud production gate 完成前不接入任务执行。

### Phase 1：配置与控制面

- 新增严格 `runtimeconfig`、`.env.example` 文档和 `/api/config` capability。
- 新增 `execution_type`、backfill 和 concurrent index。
- daemon registration 固定写 `execution_type=cli`。
- Cloud runtime CRUD 和 provider 白名单。
- agent create/update 校验 runtime 是否被当前部署启用。

### Phase 2：SDK 生产化

- 实现 `SessionStore`、恢复、增量持久化与 crash test。
- 将 tool 执行抽象为 fail-closed `SandboxExecutor`。
- 补齐路径越界、symlink、env、network、resource limit 安全测试。
- 完成 Anthropic/OpenAI 显式 provider 和错误分类。

### Phase 3：任务执行面

- 从 daemon handler 抽取 `taskexecution.Coordinator`。
- 实现 Cloud worker pool、取消、优雅关机、usage/message adapter。
- 实现 repo/workdir 准备与跨租户隔离。
- 跑通 issue/chat/autopilot/quick-create 四类 task。

### Phase 4：前端与运维

- Runtime 创建类型选择、agent picker、切换会话提示。
- onboarding 使用 server default，CLI 保持默认。
- 指标、dashboard、容量告警、费用告警和 runbook。

### Phase 5：canary

- 先仅内部 workspace，再按 workspace allowlist 开放。
- 分 provider 设置任务/日费用上限。
- canary 期间可关闭 Cloud 新建/新 claim，但不允许改为 CLI fallback。

## 11. 测试矩阵

### 单元测试

- config：默认 CLI、非法枚举、Cloud 缺 key、default/cloud 矛盾、secret 不进公开配置。
- provider：Anthropic/OpenAI 显式映射，拒绝其他 provider。
- message adapter：text/thinking/tool-use/tool-result/result/usage/error。
- session：create/pin/append/resume/crash recovery/concurrent append/wrong runtime rejection。
- sandbox：`..`、绝对路径、symlink、硬链接、敏感 env、metadata/loopback、fork bomb、超时、超限输出。
- failure classifier：auth/429/5xx/timeout/cancel/sandbox/session corruption。

### 后端集成测试

- 使用本地 fake Anthropic/OpenAI stream server，默认测试不访问真实账号。
- 两个 server worker 竞争同一 task，只有一个成功 claim。
- 全链路验证 queued -> dispatched -> running -> completed/failed/cancelled。
- session 在 worker A 崩溃后由 worker B 在同一 Cloud runtime 恢复。
- agent 切换 execution type 后新 task 不读取旧 session。
- Cloud 错误不会调用 CLI backend，CLI 错误不会调用 SDK。
- Cloud/Fleet 同时存在时不误路由：`runtime_mode=cloud, execution_type=cli` 仍由 daemon claim。
- task token、workspace isolation、comment receipt 和 realtime 与 CLI 路径一致。

### 前端测试

- 旧 server config 响应默认 CLI-only。
- Cloud 开关/默认类型的 UI 组合。
- Fleet CLI 节点和 SDK Cloud runtime 分组/文案不混淆。
- runtime 切换强制显示新会话提示。
- malformed config/API response 的 schema fallback。

### 发布前验证

```bash
(cd server && go test ./internal/runtimeconfig ./internal/cloudruntime ./internal/taskexecution)
(cd server/runtime && go test ./...)
(cd server && go test ./internal/handler ./internal/service ./pkg/agent)
pnpm typecheck
pnpm test
make check
```

真实 provider smoke test 单独使用 build tag + 显式环境开关，不进默认 CI，因为它会访问真实账号并产生费用。

## 12. 上线验收标准

1. 不配置任何新 env 时，系统行为与现在一致：CLI-only、daemon + `pkg/agent`。
2. 启用 Cloud 后，CLI runtime 和 Cloud runtime 可在同一 workspace 并存，agent 可显式选择任一类型。
3. 任何 task 的 `execution_type` 在整个生命周期内不变。
4. 代码库中不存在 Cloud -> CLI 或 CLI -> Cloud fallback 分支。
5. daemon 不链接/创建 `server/runtime/agent` 实例；API server Cloud worker 不创建 `pkg/agent` backend。
6. Anthropic/OpenAI 均完成 stream、tool、cancel、usage、session resume 集成测试。
7. Cloud session 能够跨 API server 副本恢复，不依赖本地 JSONL 文件。
8. sandbox escape 测试全部通过，tool 进程不可读取 server 密钥或其他 task 目录。
9. 现有 Fleet 云节点仍按 CLI runtime 执行，不被 SDK worker claim。
10. UI/API/metrics 都能区分 placement (`runtime_mode`) 和执行类型 (`execution_type`)。

## 13. 需要同步的主要文件

| 范围 | 文件/目录 |
| --- | --- |
| 配置 | `.env.example`, `server/cmd/server/router.go`, `server/internal/handler/config.go`, `packages/core/config/` |
| 数据库 | `server/migrations/`, `server/pkg/db/queries/runtime.sql`, `server/pkg/db/queries/agent.sql`, sqlc generated files |
| 边界纠正 | `server/pkg/agent/openagent.go`, `server/pkg/agent/agent.go`, migration 233 的后续迁移 |
| 任务编排 | `server/internal/handler/daemon.go`, `server/internal/service/task.go`, 新 `server/internal/taskexecution/` |
| Cloud SDK | `server/runtime/`, 新 `server/internal/cloudruntime/` |
| 现有 Fleet | `server/internal/cloudruntime/` -> `server/internal/cloudfleet/`, `server/internal/handler/cloud_runtime.go` |
| 前端 | `packages/core/types/agent.ts`, `packages/core/runtimes/`, `packages/views/runtimes/`, `packages/views/agents/`, onboarding |
| 文档/观测 | `docs/analytics.md`, runtime 安装/部署文档、runbook |

CLI provider 覆盖不在文档中手写固定数字作为代码真相。当前仓库的 `agent.SupportedTypes` 是协议族权威清单，与参考文档所述“27 个 provider/backend”口径不完全一致。实施时应从 `pkg/agent` 及 provider alias 生成能力清单，防止代码与产品文案再次漂移。
