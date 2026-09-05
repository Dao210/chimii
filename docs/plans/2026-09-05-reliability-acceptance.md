# 2026-09-05 可靠性与作品入口验收

本记录对应当前本地工作区。README 中的“已有代码、已测试、已部署、实物验证”分别判断，不能互相替代。数据库验收使用独立 PostgreSQL 17 测试实例，迁移至 289；未读取或改写生产数据。

## 实现范围

| 项目 | 本次实现 | 保持简单的方式 |
| --- | --- | --- |
| 消息缓存一致性 | HTTP 回填保留期间收到的实时帧；按 seq 去重；发送响应和广播按消息 ID 合并，保留附件等已有字段 | 共用一个缓存合并函数；重复事件保持引用稳定；单条顺序追加不排序；未打开会话不因广播创建消息缓存 |
| 删除并发与清理 | 明确搭建、库存、儿童及电路数据清理清单；验证事务回滚、工作区隔离、删除与写入互斥、已删工作区拒收 worker 结果 | 复用现有工作区锁及应用层事务；共享零件目录不属于工作区清理范围 |
| 搭建进度 | 预览、开始、继续、明确完成；刷新恢复；过期版本返回 409 | `build_creation` 增加三个标量字段，保存进度不改 recipe、plan、validation、MPD 等作品快照 |
| 电路闭环 | 验证真实库存、保存、步骤恢复、BOSON 连接、导出和家庭反馈隔离 | 保存进度后更新已有缓存行；本次保留现有详情响应协议 |
| 移动端作品入口 | More → Creations，积木/电路最近列表及原生摘要详情，从详情打开网页继续搭建 | 积木最近 60 项、电路最近 50 项，与服务端排序和权限一致；复用网页搭建和登录，不在 URL 中携带令牌 |
| 文档状态 | 中英文 README 状态表、CLAUDE 定位及两处现有模块 README 校准 | 沿用现有文件和 plans 目录 |

## 已执行检查

| 检查 | 结果和覆盖 |
| --- | --- |
| Core 专项 Vitest | 7 个文件、74 项通过：消息缓存、实时完成事件、进度 API 异常响应、Build/Circuit schema 与模块协议 |
| Views 专项 Vitest | 既有 4 个文件、53 项通过；新增 BuildResult 流程 1 项通过，验证预览不保存、逐步保存、明确点击完成 |
| Mobile 专项 Vitest | 2 项通过：步骤显示、未知观察值、网页地址配置及不携带认证参数 |
| Go handler + `-race` | 20 个顶层测试通过，含真实数据库删除/写入两种顺序、数据库阻塞屏障、清理回滚、隔离、worker 过期租约和删除后完成拒绝、Build 版本冲突和 Circuit 场景 |
| Go Build / Circuit / Middleware | 三个包测试通过；相关四个后端包 `go vet` 通过 |
| TypeScript | Core、Views、Web、Desktop 和 Mobile 类型检查通过 |
| 静态检查 | 本次涉及的消息、进度、作品 UI 与新增移动端文件 ESLint 通过；`git diff --check` 通过 |
| Chromium 端到端 | 3 条通过：积木澄清/生成/进度恢复；收音机库存/步骤/下载；BOSON 库存/连接/家庭反馈。覆盖 390px 窄屏和权限边界 |
| Mobile iOS JavaScript 导出 | Expo 导出成功；仅证明 JS 模块及资源能打包，不等同于原生安装验收 |

## 性能证据

- Build 摘要查询只选取列表所需字段，不读取 `recipe`、`build_plan` 或 `ldraw_mpd`。数据库测试中的作品含 100,000 字节 MPD，摘要响应为 331 字节。此数值是该夹具的响应大小，不是线上延迟或吞吐提升比例。
- Build 列表卡片使用轻量摘要，详情才加载完整作品。保存进度后取消同键在途读取，再更新进度和已有摘要缓存，避免旧响应覆盖新步骤。
- 浏览器测试断言 Build 进度流程未请求旧的完整作品列表；Circuit 单次步骤保存没有追加列表 GET。Circuit 进度接口仍返回现有详情，尚未做响应大小专项改造。
- 消息重复事件不产生新的等值列表；顺序追加避开全量排序。未进行长期会话压力测试或线上性能测量。

## 复跑入口

数据库测试使用一次性测试库，先应用 `server/migrations`。两个环境变量应指向同一个测试实例：

```bash
# 在 server/ 运行；DATABASE_URL 和 CHIMII_BUILD_TEST_DATABASE_URL 仅设置为测试库。
go test -race ./internal/handler -run 'Test(BuildProgress|BuildWorkerDB|WorkspaceBuild|Circuit)' -count=1 -v
go test ./internal/build ./internal/circuit ./internal/middleware
go vet ./internal/build ./internal/circuit ./internal/handler ./internal/middleware
```

```bash
# 在仓库根目录运行。
pnpm --filter @chimii/core exec vitest run chat/message-cache.test.ts chat/queries.test.ts realtime/use-realtime-sync.test.ts build/schemas.test.ts build/progress-api.test.ts circuit
pnpm --filter @chimii/views exec vitest run build/components/build-result.test.tsx
pnpm --filter @chimii/mobile exec vitest run lib/creations.test.ts
pnpm --filter @chimii/core --filter @chimii/views --filter @chimii/web --filter @chimii/desktop typecheck
pnpm --filter @chimii/mobile typecheck
```

浏览器测试需本地 API、Web、独立测试数据库及 `scripts/build-e2e-llm-fixture.mjs`。将 API 的 LLM 地址指向该夹具，设置 `CHIMII_BUILD_E2E_STUB=1` 后执行 `pnpm exec playwright test e2e/build.spec.ts e2e/circuit.spec.ts --workers=1`。夹具验证 HTTP、队列、数据库、编译器与 UI 的连接，不证明真实模型规划质量。

## 发布及实物边界

- 本次没有部署后端/Web，没有发布桌面或移动安装包，也没有重新核实线上运行版本。上线应先应用迁移 289，再发布依赖它的服务；回退前保留进度数据，down migration 会删除三个进度字段。
- 移动端需配置对应环境的 `EXPO_PUBLIC_WEB_URL`；完整搭建在浏览器中继续，可能需要同一账号重新登录。本次未执行模拟器界面对照或真机安装、登录及网页跳转验收。
- 积木模型检查和电路约束验证均不等同于实物验证。测试中的家庭反馈为夹具数据；参考电路的 `not_tested` 状态未被改为通过。需实际完成积木搭建、通电及家庭测试后记录新的独立证据。
- 本次是上述范围的专项验收，未执行全仓 `make check`、生产模型质量评估、桌面安装包验收或生产压力测试。
