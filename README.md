<div align="center">

# CHIMII 奇觅

### Imagine it. Build it. Bring it to life.

**An AI invention kit that helps kids turn ideas into real moving creations.**

[![CI](https://github.com/Dao210/chimii/actions/workflows/ci.yml/badge.svg)](https://github.com/Dao210/chimii/actions/workflows/ci.yml)
[![GitHub stars](https://img.shields.io/github/stars/Dao210/chimii?style=flat)](https://github.com/Dao210/chimii/stargazers)
[![License](https://img.shields.io/badge/license-Modified%20Apache%202.0-2f6f5e)](LICENSE)

[Website](https://chimii.ai) · [Product vision](docs/plans/%E5%A5%87%E8%A7%85%E5%8F%91%E6%98%8E%E5%AE%B6prd%2020260728.md) · [Development](#development) · [X](https://x.com/ChimiiAI)

**English | [简体中文](README.zh-CN.md)**

</div>

## From consuming AI to creating in the real world

Most AI experiences keep children inside a screen: generate another image, watch another animation, ask another question.

**CHIMII 奇觅发明家** takes the opposite approach. A child starts with any idea. CHIMII understands the parts they already have, turns the idea into a buildable plan, and guides them as they build, test, and reinvent it in the physical world.

AI is the guide. The child is the inventor.

> [!IMPORTANT]
> **CHIMII is in an active product transition.** The marketing experience and product direction now reflect the children’s invention kit described here. This repository still contains the mature agent-workspace foundation from the previous product direction. Invention-specific capabilities are being developed on top of that foundation and are marked clearly in the status table below.

## How CHIMII works

| Step | Child experience | What CHIMII does |
| --- | --- | --- |
| **1. Imagine it** | Say an idea, describe it by voice, or sketch it. | Understand the intent and turn it into a physical design goal. |
| **2. Build it** | Follow clear steps using parts already available. | Check part constraints and create a stable, executable build plan. |
| **3. Bring it to life** | Add movement or expression, then test and modify it. | Guide motion, interaction, troubleshooting, and iteration. |

The goal is not to make AI produce more content for a child. It is to help the child produce something real.

## What children can create

- **Robot pets** that move and react
- **Secret bases** with doors, hiding places, and mechanisms
- **Clever traps** powered by gears, levers, and gravity
- **Self-driving racers** built to test, tune, and rebuild
- **Expressive dolls** that show emotion and respond to the world

The examples are starting points, not templates. Children choose the story and keep changing the result.

## What grows along the way

| For children | For parents |
| --- | --- |
| The excitement of making an idea real | **Creativity** through open-ended design |
| Ownership over the design and story | **Problem solving** through testing and repair |
| A creation they can touch, move, and modify | **Engineering thinking** through structure, motion, and cause |
| Confidence earned through building | **Responsible AI literacy** by using AI as a tool, not a substitute for thought |
| More time making in the physical world | **Less passive screen time** and more hands-on attention |

## Product principles

1. **The child stays in charge.** CHIMII suggests, explains, and encourages; it does not replace the child’s decisions or effort.
2. **Plans must be physically buildable.** Available parts, stability, gravity, and assembly order are product constraints, not afterthoughts.
3. **Screens should lead back to the real world.** Give the next useful cue, then return attention to the object being built.
4. **Iteration is part of the invention.** A failed mechanism is an invitation to observe, diagnose, and try again.
5. **Privacy starts on the device.** The target architecture favors local processing for children’s voice, images, and invention data whenever practical.

## Project status

| Area | Code | Executed tests | Deployment | Physical verification |
| --- | --- | --- | --- | --- |
| Workspace, chat and runtime foundation | Implemented | Focused cache and PostgreSQL cleanup/concurrency tests; see acceptance record | Not rechecked in this change | Not applicable |
| Brick planning and compilation | Generic target shapes, clarification, per-model inventory constraints, reviewed modules, validation and MPD export | Local compiler/handler checks and fixture-based browser flow | Not rechecked in this change | Not performed; geometric validation is not a physical test |
| Brick building progress | Start/continue/complete, revision checks and summary lists | Local API and UI tests; see acceptance record | Requires migration 289; not deployed by this change | Not performed |
| Circuit building | Saved inventory, step progress, reference circuits, BOSON module connections and family trial reports | Local compiler/API tests and radio/BOSON browser flows | Not rechecked in this change | Reference designs remain `not_tested`; family reports are separate evidence |
| Mobile creation entry | Native recent lists and summary details; full building opens the web | See acceptance record for mobile checks and remaining limits | No mobile release performed | Not performed |
| Multimodal ideas and parent growth portal | Voice/sketch planning and growth insights remain roadmap items | Not established | Not established | Not performed |

Status reflects the local checkout on 2026-09-05, including work awaiting commit or release. Test details and limits are in the [acceptance record](docs/plans/2026-09-05-reliability-acceptance.md). A local test does not establish deployment or hardware acceptance. The [Inventor PRD](docs/plans/%E5%A5%87%E8%A7%85%E5%8F%91%E6%98%8E%E5%AE%B6prd%2020260728.md) describes the broader direction.

The 2026-09-08 [BrickGPT comparison experiment](tasks/build-eval-first-batch.md)
adds an offline 100-case baseline, pinned force-analysis bridge and dataset
candidate import. Exact reproduction of the optional BrickGPT model needs
Llama base weights; Chimii continues to use DeepSeek. Most force comparisons
hit the current solver license size limit.

The 2026-09-09 [algorithm integration](tasks/brickgpt-integration.md)
adds cumulative seam priorities and bounded neighborhood retiling to the Go
compiler for configured inventory. The original 100-case baseline is unchanged; a separate 80-case
inventory corpus improved from 68 to 71 accepted layouts. This is local
compiler evidence, not deployment, live-model or physical verification.

## Repository architecture

```text
apps/web/        Next.js web app and multilingual marketing experience
apps/desktop/    Electron desktop app
apps/mobile/     Expo / React Native mobile app
server/          Go API, realtime services, CLI, daemon, and migrations
packages/core/   Headless business logic, API client, queries, and stores
packages/ui/     Shared UI primitives and design tokens
packages/views/  Shared web and desktop product views
```

| Layer | Technology |
| --- | --- |
| Web | Next.js 16, React 19, Tailwind CSS 4 |
| Desktop | Electron |
| Mobile | Expo / React Native |
| Backend | Go, Chi, sqlc, gorilla/websocket |
| Data | PostgreSQL 17 with pgvector |
| Monorepo | pnpm workspaces and Turborepo |

The current application foundation uses React Query for server state and Zustand for client/view state. Web and desktop share headless logic and product views; mobile owns its platform-specific UI and runtime.

## Development

### Prerequisites

- [Node.js](https://nodejs.org/) 22
- [pnpm](https://pnpm.io/) 10.28+
- [Go](https://go.dev/) 1.26+
- [Docker](https://www.docker.com/)

### Start locally

```bash
make dev
```

`make dev` prepares the environment, installs dependencies, starts PostgreSQL, runs migrations, and launches the backend and web app. Run `make help` to see the full command list.

### Common commands

| Command | Purpose |
| --- | --- |
| `make dev` | Set up and start the complete local stack |
| `make start` | Start backend and frontend using the current env file |
| `make stop` | Stop the app processes for this checkout |
| `pnpm dev:web` | Run only the Next.js web app |
| `pnpm dev:desktop` | Run the Electron desktop app |
| `pnpm typecheck` | Type-check all TypeScript workspaces |
| `pnpm test` | Run TypeScript unit tests |
| `make test` | Run Go tests |
| `make check` | Run the full local verification pipeline |

For architecture and contribution rules, read [AGENTS.md](AGENTS.md) and [CLAUDE.md](CLAUDE.md). The mobile app has additional instructions in [`apps/mobile/README.md`](apps/mobile/README.md).

## Contributing

CHIMII is evolving quickly. Contributions are welcome, especially around physical-design generation, child-friendly interaction, responsible AI, privacy, accessibility, and multilingual experience.

Before opening a change:

1. Read the repository guidelines in [AGENTS.md](AGENTS.md) and [CLAUDE.md](CLAUDE.md).
2. Keep product claims aligned with the status table above.
3. Run the narrowest relevant checks, then `make check` when the change warrants the full pipeline.
4. Keep English and Simplified Chinese documentation aligned when changing product-level information.

## License

CHIMII is released under a [modified Apache License 2.0](LICENSE) with additional conditions for hosted, embedded, and commercially distributed offerings. Review the license before commercial use.
