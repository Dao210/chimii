<div align="center">

# CHIMII 奇觅

### 让孩子从 AI 内容消费者，变成真实世界的创造者。

**孩子从任意想法出发，AI 理解现有零件，生成可执行方案，并陪伴孩子完成和改造。**

[![CI](https://github.com/chimii-ai/chimii/actions/workflows/ci.yml/badge.svg)](https://github.com/chimii-ai/chimii/actions/workflows/ci.yml)
[![GitHub stars](https://img.shields.io/github/stars/chimii-ai/chimii?style=flat)](https://github.com/chimii-ai/chimii/stargazers)
[![License](https://img.shields.io/badge/license-Modified%20Apache%202.0-2f6f5e)](LICENSE)

[官网](https://chimii.ai) · [产品愿景](docs/plans/%E5%A5%87%E8%A7%85%E5%8F%91%E6%98%8E%E5%AE%B6prd%2020260728.md) · [本地开发](#本地开发) · [X](https://x.com/ChimiiAI)

**[English](README.md) | 简体中文**

</div>

## 从消费 AI 内容，到在真实世界创造

大多数面向孩子的 AI 体验，都在把孩子留在屏幕里：再生成一张图片、再看一段动画、再问一个问题。

**CHIMII 奇觅发明家**选择了相反的方向。孩子从任意想法出发，奇觅理解他们手边已有的零件，把想法转化为真正能搭建的方案，并在孩子动手、测试和持续改造的过程中提供陪伴。

AI 是向导，孩子才是发明家。

> [!IMPORTANT]
> **CHIMII 正处于产品方向升级阶段。** 当前营销体验与产品叙事已经切换到本文描述的儿童 AI 发明套件。仓库中仍保留上一阶段已经成熟的 AI Agent 协作基础设施。奇觅发明家专属能力正在这套基础上开发，尚未完成的部分会在下方状态表中明确标记。

## 奇觅如何帮助孩子创造

| 阶段 | 孩子的体验 | 奇觅的作用 |
| --- | --- | --- |
| **1. Imagine it · 奇思妙想** | 说出想法、用语音描述，或随手画下来。 | 理解创意意图，把它转化为实体设计目标。 |
| **2. Build it · 动手建造** | 使用家中已有零件，跟随清楚的步骤完成搭建。 | 检查零件约束，生成稳固、可执行的搭建方案。 |
| **3. Bring it to life · 赋予生命** | 加入动作或表情，反复测试并继续改造。 | 引导运动、互动、故障排查与持续迭代。 |

奇觅的目标不是替孩子生产更多内容，而是帮助孩子亲手创造真实存在的作品。

## 孩子可以创造什么

- **机器宠物**：会移动，也会对周围作出回应
- **秘密基地**：拥有舱门、藏宝空间和机械结构
- **陷阱机关**：使用齿轮、杠杆与重力触发
- **自动赛车**：在测试、调校和重建中持续进化
- **情绪玩偶**：会表达情绪，也能感知外部世界

这些只是起点，不是固定模板。故事由孩子决定，作品也可以一直改变。

## 创造过程中自然生长的能力

| 孩子获得 | 家长看到 |
| --- | --- |
| 把脑洞变成现实的兴奋感 | 在开放式设计中培养**创造力** |
| 对设计和故事的真正主导权 | 在测试与修复中提升**解决问题能力** |
| 可以触摸、运动和继续改造的作品 | 在结构、运动与因果中形成**工程思维** |
| 通过亲手完成获得的创造自信 | 把 AI 当工具而非思考替代品的**负责任 AI 素养** |
| 更多在真实世界动手的时间 | **更少被动屏幕时间**，更多专注与行动 |

## 产品原则

1. **孩子始终拥有主导权。** 奇觅负责建议、解释和鼓励，但不替代孩子的选择与努力。
2. **方案必须真的能造出来。** 现有零件、结构稳定、重力和装配顺序都是产品的核心约束。
3. **屏幕最终要把注意力带回真实世界。** 给出下一条有效提示，然后让孩子继续观察和操作眼前的作品。
4. **失败也是发明的一部分。** 机关没有运行时，先观察、定位原因，再尝试新的方案。
5. **隐私从设备端开始。** 在条件允许时，目标架构优先在本地处理孩子的语音、图片与发明数据。

## 项目状态

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| 多语言营销体验 | **仓库已实现** | 支持英文、简体中文、日文和韩文的响应式奇觅发明家首页。 |
| Web、桌面端与移动端基础 | **仓库已实现** | 已具备共享业务页面、身份认证、工作区、实时更新和多平台外壳。 |
| Go 服务与本地运行时基础 | **仓库已实现** | 已具备 API、WebSocket、PostgreSQL、daemon 与 Agent 执行基础能力。 |
| 奇思妙想编译器 | **路线图** | 多模态创意输入与受实体零件约束的设计生成。 |
| 物理结构与装配求解器 | **路线图** | 零件审计、稳定性检查与儿童友好的装配步骤。 |
| 赋予生命体验 | **路线图** | 屏幕表情、动力组件、传感器与极简行为逻辑。 |
| 家长成长观测站 | **路线图** | 非排名式成长反馈、屏幕时间引导与负责任 AI 护栏。 |

完整产品方向请参阅[《奇觅发明家 PRD》](docs/plans/%E5%A5%87%E8%A7%85%E5%8F%91%E6%98%8E%E5%AE%B6prd%2020260728.md)。路线图描述的是产品意图，不代表相关功能已经上线。

## 仓库架构

```text
apps/web/        Next.js Web 应用与多语言营销体验
apps/desktop/    Electron 桌面应用
apps/mobile/     Expo / React Native 移动应用
server/          Go API、实时服务、CLI、daemon 与数据库迁移
packages/core/   无界面的业务逻辑、API 客户端、查询与状态
packages/ui/     共享 UI 原语与设计令牌
packages/views/  Web 与桌面端共享的业务页面
```

| 层级 | 技术栈 |
| --- | --- |
| Web | Next.js 16、React 19、Tailwind CSS 4 |
| 桌面端 | Electron |
| 移动端 | Expo / React Native |
| 后端 | Go、Chi、sqlc、gorilla/websocket |
| 数据 | PostgreSQL 17 与 pgvector |
| Monorepo | pnpm workspaces 与 Turborepo |

当前应用基础使用 React Query 管理服务端状态，使用 Zustand 管理客户端与视图状态。Web 与桌面端共享无界面逻辑和业务页面；移动端拥有独立的平台 UI 与运行时。

## 本地开发

### 环境要求

- [Node.js](https://nodejs.org/) 22
- [pnpm](https://pnpm.io/) 10.28+
- [Go](https://go.dev/) 1.26+
- [Docker](https://www.docker.com/)

### 一键启动

```bash
make dev
```

`make dev` 会准备环境、安装依赖、启动 PostgreSQL、执行数据库迁移，并启动后端与 Web 应用。运行 `make help` 可以查看完整命令列表。

### 常用命令

| 命令 | 用途 |
| --- | --- |
| `make dev` | 初始化并启动完整本地环境 |
| `make start` | 使用当前环境文件启动前后端 |
| `make stop` | 停止当前检出目录的应用进程 |
| `pnpm dev:web` | 仅运行 Next.js Web 应用 |
| `pnpm dev:desktop` | 运行 Electron 桌面应用 |
| `pnpm typecheck` | 检查全部 TypeScript workspace |
| `pnpm test` | 运行 TypeScript 单元测试 |
| `make test` | 运行 Go 测试 |
| `make check` | 运行完整的本地验证流程 |

架构与贡献规则请阅读 [AGENTS.md](AGENTS.md) 和 [CLAUDE.md](CLAUDE.md)。移动端的额外说明位于 [`apps/mobile/README.md`](apps/mobile/README.md)。

## 参与贡献

CHIMII 正在快速演进。我们欢迎与实体设计生成、儿童友好交互、负责任 AI、隐私、无障碍和多语言体验相关的贡献。

提交改动前，请确保：

1. 已阅读 [AGENTS.md](AGENTS.md) 与 [CLAUDE.md](CLAUDE.md) 中的仓库规范。
2. 产品能力描述与上方状态表保持一致，不把路线图功能写成已上线功能。
3. 先运行与改动最相关的检查；改动范围较大时，再运行 `make check`。
4. 修改产品级信息时，同步维护英文与简体中文文档。

## 开源协议

CHIMII 使用带附加条件的[修改版 Apache License 2.0](LICENSE)。托管服务、嵌入式使用和商业分发可能需要额外授权，请在商业使用前完整阅读协议。
