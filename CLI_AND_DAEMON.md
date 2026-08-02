# CLI and Agent Daemon Guide

The `chimii` CLI connects your local machine to Chimii. It handles authentication, workspace management, issue tracking, and runs the agent daemon that executes AI tasks locally.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install chimii-ai/tap/chimii
```

### Build from Source

```bash
git clone https://github.com/chimii-ai/chimii.git
cd chimii
make build
cp server/bin/chimii /usr/local/bin/chimii
```

### Update

```bash
brew upgrade chimii-ai/tap/chimii
```

For install script or manual installs, use:

```bash
chimii update
```

`chimii update` auto-detects your installation method and upgrades accordingly.

## Quick Start

```bash
# One-command setup: configure, authenticate, and start the daemon
chimii setup

# For self-hosted (local) deployments:
chimii setup self-host
```

Or step by step:

```bash
# 1. Authenticate (opens browser for login)
chimii login

# 2. Start the agent daemon
chimii daemon start

# 3. Done — agents in your watched workspaces can now execute tasks on your machine
```

`chimii login` automatically discovers all workspaces you belong to and adds them to the daemon watch list.

## Authentication

### Browser Login

```bash
chimii login
```

Opens your browser for OAuth authentication, creates a 90-day personal access token, and auto-configures your workspaces.

### Token Login

```bash
chimii login --token <mul_...>
```

Authenticate using a personal access token directly. Useful for headless environments. Pass `--token=` with an empty value to be prompted interactively (so the token never lands in shell history).

### Check Status

```bash
chimii auth status
```

Shows your current server, user, and token validity.

### Logout

```bash
chimii auth logout
```

Removes the stored authentication token.

## Agent Daemon

The daemon is the local agent runtime. It detects available AI CLIs on your machine, registers them with the Chimii server, and executes tasks when agents are assigned work.

### Start

```bash
chimii daemon start
```

By default, the daemon runs in the background and logs to `~/.chimii/daemon.log`.

To run in the foreground (useful for debugging):

```bash
chimii daemon start --foreground
```

### Stop

```bash
chimii daemon stop
```

### Status

```bash
chimii daemon status
chimii daemon status --output json
```

Shows PID, uptime, detected agents, and watched workspaces.

### Logs

```bash
chimii daemon logs              # Last 50 lines
chimii daemon logs -f           # Follow (tail -f)
chimii daemon logs -n 100       # Last 100 lines
```

### Supported Agents

The daemon auto-detects these AI CLIs on your PATH:

| CLI | Command | Description |
|-----|---------|-------------|
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code) | `claude` | Anthropic's coding agent |
| [Codex](https://github.com/openai/codex) | `codex` | OpenAI's coding agent |
| [GitHub Copilot CLI](https://docs.github.com/en/copilot) | `copilot` | GitHub's coding agent (model routed by your GitHub entitlement) |
| OpenCode | `opencode` | Open-source coding agent |
| OpenClaw | `openclaw` | Open-source coding agent |
| Hermes | `hermes` | Nous Research coding agent |
| Gemini | `gemini` | Google's coding agent |
| [Pi](https://pi.dev/) | `pi` | Pi coding agent |
| [Cursor Agent](https://cursor.com/) | `cursor-agent` | Cursor's headless coding agent |
| Kimi | `kimi` | Moonshot coding agent |
| Kiro CLI | `kiro-cli` | Kiro ACP coding agent |
| [Qoder CLI](https://docs.qoder.com/) | `qodercli` | Qoder ACP coding agent |
| [Trae](https://docs.trae.cn/cli) | `traecli` | ByteDance TRAE CLI (ACP via `traecli acp serve`) |
| [Grok Build CLI](https://docs.x.ai/) | `grok` | xAI Grok Build CLI (ACP via `grok agent stdio`) |
| [Qwen Code](https://github.com/QwenLM/qwen-code) | `qwen` | Alibaba Qwen Code (`qwen -p` with stream-json) |

You need at least one installed. The daemon registers each detected CLI as an available runtime.

### How It Works

1. On start, the daemon detects installed agent CLIs and registers a runtime for each agent in each watched workspace
2. It polls the server at a configurable interval (default: 3s) for claimed tasks
3. When a task arrives, it creates an isolated workspace directory, spawns the agent CLI, and streams results back
4. Heartbeats are sent periodically (default: 15s) so the server knows the daemon is alive
5. On shutdown, all runtimes are deregistered

### Configuration

Daemon behavior is configured via flags or environment variables:

| Setting | Flag | Env Variable | Default |
|---------|------|--------------|---------|
| Poll interval | `--poll-interval` | `CHIMII_DAEMON_POLL_INTERVAL` | `3s` |
| Heartbeat interval | `--heartbeat-interval` | `CHIMII_DAEMON_HEARTBEAT_INTERVAL` | `15s` |
| Agent timeout | `--agent-timeout` | `CHIMII_AGENT_TIMEOUT` | `0` (no cap; bounded by the watchdogs) |
| Codex semantic inactivity timeout | `--codex-semantic-inactivity-timeout` | `CHIMII_CODEX_SEMANTIC_INACTIVITY_TIMEOUT` | `10m` |
| OpenCode idle watchdog | — | `CHIMII_OPENCODE_IDLE_WATCHDOG` | `10m` (`0` falls back to the generic idle watchdog; cannot extend it) |
| Max concurrent tasks | `--max-concurrent-tasks` | `CHIMII_DAEMON_MAX_CONCURRENT_TASKS` | `20` |
| Daemon ID | `--daemon-id` | `CHIMII_DAEMON_ID` | hostname |
| Device name | `--device-name` | `CHIMII_DAEMON_DEVICE_NAME` | hostname |
| Runtime name | `--runtime-name` | `CHIMII_AGENT_RUNTIME_NAME` | `Local Agent` |
| Workspaces root | — | `CHIMII_WORKSPACES_ROOT` | `~/chimii_workspaces` |
| GC enabled | — | `CHIMII_GC_ENABLED` | `true` (set `false`/`0` to disable) |
| GC scan interval | — | `CHIMII_GC_INTERVAL` | `2h` |
| GC TTL (done/cancelled issues) | — | `CHIMII_GC_TTL` | `24h` |
| GC orphan TTL (no `.gc_meta.json`) | — | `CHIMII_GC_ORPHAN_TTL` | `72h` |
| GC artifact TTL (open issues) | — | `CHIMII_GC_ARTIFACT_TTL` | `12h` (set `0` to disable) |
| GC artifact patterns | — | `CHIMII_GC_ARTIFACT_PATTERNS` | `node_modules,.next,.turbo` |

#### Workspace garbage collection

The daemon periodically scans `CHIMII_WORKSPACES_ROOT` and reclaims disk space in three modes:

- **Full task cleanup** — when an issue's status is `done` or `cancelled` and has been idle for `CHIMII_GC_TTL`, the entire task directory is removed.
- **Orphan cleanup** — task directories with no `.gc_meta.json` (e.g. left over from a daemon crash) are removed once they exceed `CHIMII_GC_ORPHAN_TTL`.
- **Artifact-only cleanup** — when a task has been completed for at least `CHIMII_GC_ARTIFACT_TTL` but the issue is still open, regenerable build outputs whose directory basename matches `CHIMII_GC_ARTIFACT_PATTERNS` are removed. The daemon also reclaims the exact managed path `codex-home/.sandbox-bin`; old task metadata without `completed_at` becomes eligible for this managed-only cleanup after its `.gc_meta.json` file has been idle for `CHIMII_GC_ORPHAN_TTL`. The rest of the task (source, `.git`, `output/`, `logs/`, `.gc_meta.json`, Codex auth/config/session state) is preserved so the agent can resume it.

Configured patterns are basename-only — entries containing `/` or `\` are silently dropped — and `.git` subtrees are never descended into. The managed Codex cache is matched by its exact relative path, so a repository's own `.sandbox-bin` is not removed unless an operator explicitly adds that basename to `CHIMII_GC_ARTIFACT_PATTERNS`. The default list (`node_modules`, `.next`, `.turbo`) is intentionally narrow; extend it per deployment if your repos consistently produce other regenerable directories (for example, `CHIMII_GC_ARTIFACT_PATTERNS=node_modules,.next,.turbo,target,__pycache__`). To disable artifact cleanup entirely, including the managed Codex cache, set `CHIMII_GC_ARTIFACT_TTL=0`.

Agent-specific overrides:

| Variable | Description |
|----------|-------------|
| `CHIMII_CLAUDE_PATH` | Custom path to the `claude` binary |
| `CHIMII_CLAUDE_MODEL` | Override the Claude model used |
| `CHIMII_CLAUDE_ARGS` | Default extra arguments for Claude Code runs |
| `CHIMII_CODEX_PATH` | Custom path to the `codex` binary |
| `CHIMII_CODEX_MODEL` | Override the Codex model used |
| `CHIMII_CODEX_ARGS` | Default extra arguments for Codex runs |
| `CHIMII_COPILOT_PATH` | Custom path to the `copilot` binary |
| `CHIMII_COPILOT_MODEL` | Override the Copilot model used (note: GitHub Copilot routes models through your account entitlement, so this may not be honoured) |
| `CHIMII_OPENCODE_PATH` | Custom path to the `opencode` binary |
| `CHIMII_OPENCODE_MODEL` | Override the OpenCode model used |
| `CHIMII_OPENCLAW_PATH` | Custom path to the `openclaw` binary |
| `CHIMII_OPENCLAW_MODEL` | Override the OpenClaw model used |
| `CHIMII_HERMES_PATH` | Custom path to the `hermes` binary |
| `CHIMII_HERMES_MODEL` | Override the Hermes model used |
| `CHIMII_GEMINI_PATH` | Custom path to the `gemini` binary |
| `CHIMII_GEMINI_MODEL` | Override the Gemini model used |
| `CHIMII_PI_PATH` | Custom path to the `pi` binary |
| `CHIMII_PI_MODEL` | Override the Pi model used |
| `CHIMII_CURSOR_PATH` | Custom path to the `cursor-agent` binary |
| `CHIMII_CURSOR_MODEL` | Override the Cursor Agent model used |
| `CHIMII_KIMI_PATH` | Custom path to the `kimi` binary |
| `CHIMII_KIMI_MODEL` | Override the Kimi model used |
| `CHIMII_KIRO_PATH` | Custom path to the `kiro-cli` binary |
| `CHIMII_KIRO_MODEL` | Override the Kiro model used |
| `CHIMII_QODER_PATH` | Custom path to the `qodercli` binary |
| `CHIMII_QODER_MODEL` | Override the Qoder model used |
| `CHIMII_TRAECLI_PATH` | Custom path to the `traecli` binary |
| `CHIMII_TRAECLI_MODEL` | Override the Trae model used (a model id from your logged-in traecli catalog, e.g. `Doubao-Seed-2.1-Pro`) |
| `CHIMII_GROK_PATH` | Custom path to the `grok` binary (defaults to `grok` on PATH; often `~/.grok/bin/grok`) |
| `CHIMII_GROK_MODEL` | Override the Grok model used (e.g. `grok-4.5`) |
| `CHIMII_QWEN_PATH` | Custom path to the `qwen` binary |
| `CHIMII_QWEN_MODEL` | Override the Qwen Code model used |
| `CHIMII_QWEN_ARGS` | Daemon-wide extra Qwen arguments (POSIX shellword parsing; managed protocol flags are filtered) |

If a previously generated `~/.chimii/hooks` wrapper is first on `PATH` and calls the same command name again, the daemon skips that hooks directory during built-in agent discovery and records the real binary path behind it. If your interactive shell still recurses when you run `claude`, `codex`, or `hermes` manually, remove the hooks entry from your shell startup file or replace the wrapper body with an absolute `exec /path/to/real-binary "$@"`.

The daemon launches Qoder as `qodercli --yolo --acp`, matching Qoder’s ACP “bypass permissions” mode so tool runs do not block on interactive approval in headless runs.
The daemon launches Qwen Code as `qwen -p <prompt> --output-format stream-json`. It writes the task brief to `QWEN.md`; when an agent has managed `mcp_config`, the daemon writes a 0600 per-run JSON file and passes it through `--mcp-config <path>`, then removes it after the process exits. A null config preserves Qwen Code native MCP settings.


`CHIMII_CLAUDE_ARGS`, `CHIMII_CODEX_ARGS`, and `CHIMII_QWEN_ARGS` are parsed with POSIX shellword quoting, so values such as `--model "gpt-5.1 codex" --sandbox read-only` are split like a shell command line. Agent arguments are applied in this order: hardcoded Chimii defaults, daemon-wide env defaults, then per-agent `custom_args` from the task.

### Self-Hosted Server

When connecting to a self-hosted Chimii instance, the easiest approach is:

```bash
# One command — configures for localhost, authenticates, starts daemon
chimii setup self-host

# Or for on-premise with custom domains:
chimii setup self-host --server-url https://api.example.com --app-url https://app.example.com
```

Or configure manually:

```bash
# Set URLs individually
chimii config set server_url http://localhost:8080
chimii config set app_url http://localhost:3000

# For production with TLS:
# chimii config set server_url https://api.example.com
# chimii config set app_url https://app.example.com

chimii login
chimii daemon start
```

### Profiles

Profiles let you run multiple daemons on the same machine — for example, one for production and one for a staging server.

```bash
# Set up a staging profile
chimii setup self-host --profile staging --server-url https://api-staging.example.com --app-url https://staging.example.com

# Start its daemon
chimii daemon start --profile staging

# Default profile runs separately
chimii daemon start
```

Each profile gets its own config directory (`~/.chimii/profiles/<name>/`), daemon state, health port, and workspace root.

## Workspaces

### Working with multiple workspaces

Every command runs against a single workspace. The CLI resolves which one in this order (highest priority first):

1. `--workspace-id <id>` flag on the command
2. `CHIMII_WORKSPACE_ID` environment variable
3. The default workspace stored in your current profile (set by `chimii workspace switch` or `chimii login`)

`chimii workspace switch <id|slug>` is the day-to-day way to change the default workspace. For scripting and headless setups where you don't want any stored state, prefer the `--workspace-id` flag or the env variable. `chimii config set workspace_id <id>` is the low-level equivalent of `switch` (it writes the same setting but skips the access check).

If you need full isolation between organizations or accounts — separate tokens, separate daemons, separate config dirs — use `--profile <name>` instead. Each profile keeps its own default workspace.

### List Workspaces

```bash
chimii workspace list
chimii workspace list --full-id
chimii workspace list --output json
```

The current default workspace is marked with `*`. Table output shows short UUID prefixes — pass `--full-id` when you need the canonical UUIDs.

### Switch Default Workspace

```bash
chimii workspace switch <workspace-id>
chimii workspace switch <slug>
```

Verifies you have access to the workspace, then sets it as the default for the current profile. Subsequent commands without `--workspace-id` and `CHIMII_WORKSPACE_ID` target this workspace. Pair `--profile` if you want to change a non-default profile's workspace.

### Get Details

```bash
chimii workspace get <workspace-id>
chimii workspace get <workspace-id> --output json
```

Passing no `<workspace-id>` resolves to the current default workspace, so `chimii workspace get` doubles as "what workspace am I on?".

### List Members

```bash
chimii workspace member list <workspace-id>
```

## Issues

### List Issues

```bash
chimii issue list
chimii issue list --status in_progress
chimii issue list --priority urgent --assignee "Agent Name"
chimii issue list --assignee-id 5fb87ac7-23b5-4a7a-81fa-ed295a54545d
chimii issue list --full-id
chimii issue list --limit 20 --output json
chimii issue list --status todo --sort position       # board order (the default)
chimii issue list --sort created_at --direction desc  # newest first
```

Table output shows a routable issue `KEY` such as `MUL-123`; copy that key into follow-up commands like `issue get`, `issue comment list`, `issue status`, or `--parent`. Add `--full-id` when you need canonical UUIDs. Available filters: `--status`, `--priority`, `--assignee` / `--assignee-id`, `--project`, `--metadata`, `--limit`. Use `--assignee-id <uuid>` for unambiguous filtering when names overlap.

Results come back in board order (`position`, ascending) by default. Pass `--sort` to change the column (`position`, `title`, `created_at`, `start_date`, `due_date`, `priority`) and `--direction asc|desc` to flip the order. `position` is always ascending (it is the manual drag order), so `--direction` is rejected when `--sort` is `position` or omitted — use it only with `title`, `created_at`, `start_date`, `due_date`, or `priority`.

Use `--metadata key=value` (repeatable; combined with AND) to filter by per-issue metadata. The value is JSON-parsed: `true`/`false` become bool, numbers become numbers, anything else is a string. Wrap as `'"42"'` to force a string when the value would otherwise sniff as a number:

```bash
chimii issue list --metadata pipeline_status=waiting_review
chimii issue list --metadata pr_number=482 --metadata is_blocked=true
```

### Get Issue

```bash
chimii issue get <id>
chimii issue get <id> --output json
```

### Create Issue

```bash
chimii issue create --title "Fix login bug" --description "..." --priority high --assignee "Lambda"
chimii issue create --title "Fix login bug" --assignee-id 5fb87ac7-23b5-4a7a-81fa-ed295a54545d
```

Flags: `--title` (required), `--description`, `--status`, `--priority`, `--assignee` / `--assignee-id`, `--parent`, `--project`, `--due-date`. Pass `--assignee-id <uuid>` (mutually exclusive with `--assignee`) when scripting against the IDs returned by `chimii workspace member list --output json` / `chimii agent list --output json`.

### Update Issue

```bash
chimii issue update <id> --title "New title" --priority urgent
chimii issue update <id> --position 4.5
```

`--position` sets the raw ordering value within the board column (lower sorts first). For relative moves, `issue reorder` is easier because it works out the value for you.

### Reorder Issue

Move an issue within its current status column. The new ordering value is computed the same way the board's drag-and-drop computes it, so the CLI and UI agree on where the issue lands.

```bash
chimii issue reorder <id> --top              # top of its status column
chimii issue reorder <id> --bottom           # bottom of its status column
chimii issue reorder <id> --before <other>   # directly above another issue in the same column
chimii issue reorder <id> --after  <other>   # directly below another issue in the same column
```

Pick exactly one of `--top`, `--bottom`, `--before`, or `--after`. Reorder stays inside the issue's current column, so `--before` / `--after` must name an issue in that same column. To move an issue to a different column, change its status first with `issue status`, then reorder within the new column.

### Assign Issue

```bash
chimii issue assign <id> --to "Lambda"
chimii issue assign <id> --to-id 5fb87ac7-23b5-4a7a-81fa-ed295a54545d
chimii issue assign <id> --unassign
```

Pass `--to-id <uuid>` to assign by canonical UUID (mutually exclusive with `--to`); useful when names overlap across members and agents.

### Change Status

```bash
chimii issue status <id> in_progress
```

Valid statuses: `backlog`, `todo`, `in_progress`, `in_review`, `done`, `blocked`, `cancelled`.

### Comments

```bash
# List comments — flat timeline, chronological. Hard cap of 2000 rows; on
# long-running issues prefer one of the thread-aware reads below to keep
# context windows tight.
chimii issue comment list <issue-id>

# Single thread (root + every descendant). Anchor may be the root itself
# or any reply inside the thread — the server walks up to the root.
chimii issue comment list <issue-id> --thread <comment-id>

# Single thread, capped to the N most recent replies. The thread root is
# always included (even with --tail 0), so an agent landing on a long
# thread keeps the "what is this about" context without dragging hundreds
# of replies into its prompt.
chimii issue comment list <issue-id> --thread <comment-id> --tail 30

# Scroll older replies inside the same thread. --before / --before-id are
# the reply cursor that the previous response emitted on stderr as
# `Next reply cursor: --before <ts> --before-id <reply-id>`.
chimii issue comment list <issue-id> --thread <comment-id> --tail 30 \
    --before <ts> --before-id <reply-id>

# Most recently active threads (root + every descendant), grouped by
# thread. Returns N complete conversational arcs, oldest-active first so
# the freshest thread sits closest to "now" in an agent prompt.
chimii issue comment list <issue-id> --recent 10

# Scroll older threads. Under --recent, --before / --before-id are a
# THREAD cursor (thread last_activity_at + root id), emitted on stderr as
# `Next thread cursor: --before <ts> --before-id <root-id>`.
chimii issue comment list <issue-id> --recent 10 \
    --before <ts> --before-id <root-id>

# Incremental polling. Combines with --thread or --recent; filters out
# replies created on or before <ts> from the page (the thread root is
# exempt so the agent always gets context).
chimii issue comment list <issue-id> --thread <comment-id> --tail 30 \
    --since <RFC3339-timestamp>

# Add a comment
chimii issue comment add <issue-id> --content "Looks good, merging now"

# Reply to a specific comment
chimii issue comment add <issue-id> --parent <comment-id> --content "Thanks!"

# Delete a comment
chimii issue comment delete <comment-id>
```

**`--before` / `--before-id` semantics depend on the paging mode**, by
design — same flag, different scope:

| Mode | What the cursor walks | stderr label |
| --- | --- | --- |
| `--recent N` | Older *threads* (last_activity_at, root_id) | `Next thread cursor` |
| `--thread <id> --tail N` | Older *replies* inside that thread (created_at, id) | `Next reply cursor` |

Outside those two modes (`--thread` without `--tail`, or no `--thread`
and no `--recent`) the cursor flags are rejected so they cannot silently
no-op. The server emits the cursor headers (`X-Chimii-Next-Before` /
`X-Chimii-Next-Before-Id`) only when an older page actually exists —
exact-boundary pages (e.g. `--tail 3` on a thread with exactly 3
replies) intentionally return no cursor so callers stop paginating.

When `--since` is combined with `--recent` or `--thread --tail`, the
server additionally suppresses the cursor once the cursor target itself
is older than `since`. Older pages walk strictly older rows, so they
cannot satisfy `> since` either — emitting a cursor there would just
hand back root-only pages until the caller reaches the start of the
thread / issue. Incremental polling stops at the first page whose
cursor target falls before the watermark.

### Metadata

Per-issue metadata is a small KV map agents use to track pipeline state (PR number, pipeline status, waiting_on, ...). Keys match `^[a-zA-Z_][a-zA-Z0-9_.-]{0,63}$`, values are primitives (string / number / bool), max 50 keys per issue, blob capped at 8KB.

The bar for writing is high: pin a value only when it is materially important to the issue AND likely to be re-read by future runs on this same issue (the PR URL, the deploy URL, what we're blocked on). Most runs write zero new keys — that's the expected case. Don't pin runtime bookkeeping like `attempts`, single-run investigation notes, large logs, secrets/tokens, or description/comment copies — see the agent runtime prompt for the full anti-pattern list.

```bash
# List every key on an issue
chimii issue metadata list <issue-id>

# Read a single key
chimii issue metadata get <issue-id> --key pipeline_status

# Write a single key — value auto-typed (true/false → bool, numbers → number, else string)
chimii issue metadata set <issue-id> --key pipeline_status --value waiting_review
chimii issue metadata set <issue-id> --key pr_number --value 482
chimii issue metadata set <issue-id> --key is_blocked --value true

# Force a specific type when sniffing would pick the wrong one
chimii issue metadata set <issue-id> --key code --value 42 --type string

# Remove a key
chimii issue metadata delete <issue-id> --key pipeline_status
```

All writes are single-key atomic — concurrent agents writing different keys do not lose each other's updates. To query, use `chimii issue list --metadata key=value` (see *List Issues* above).

### Subscribers

```bash
# List subscribers of an issue
chimii issue subscriber list <issue-id>

# Subscribe yourself to an issue
chimii issue subscriber add <issue-id>

# Subscribe another member or agent by name
chimii issue subscriber add <issue-id> --user "Lambda"

# Unsubscribe yourself
chimii issue subscriber remove <issue-id>

# Unsubscribe another member or agent
chimii issue subscriber remove <issue-id> --user "Lambda"
```

Subscribers receive notifications about issue activity (new comments, status changes, etc.). Without `--user`, the command acts on the caller.

### Execution History

```bash
# List all execution runs for an issue
chimii issue runs <issue-id>
chimii issue runs <issue-id> --full-id
chimii issue runs <issue-id> --output json

# View messages for a specific execution run
chimii issue run-messages <task-id>
chimii issue run-messages <short-task-id> --issue <issue-id>
chimii issue run-messages <task-id> --output json

# Incremental fetch (only messages after a given sequence number)
chimii issue run-messages <task-id> --since 42 --output json

# Aggregated token usage for an issue (sum across all its task runs)
chimii issue usage <issue-id>
chimii issue usage <issue-id> --output json
```

The `usage` command returns the aggregated token usage for an issue, summed across all of its task runs: input tokens, output tokens, cache read/write tokens, and the run count (`task_count`). It wraps `GET /api/issues/<id>/usage` — the same figures the issue detail view shows. Use `--output json` to feed billing/cost tooling.

The `runs` command shows all past and current executions for an issue, including running tasks. Table output uses short task UUID prefixes by default; pass `--full-id` to print canonical task UUIDs. The `run-messages` command accepts full task UUIDs directly; copied short task prefixes must be scoped with `--issue <issue-id>` so the CLI only checks that issue's runs. It shows the detailed message log (tool calls, thinking, text, errors) for a single run. Use `--since` for efficient polling of in-progress runs.

## Projects

Projects group related issues (e.g. a sprint, an epic, a workstream). Every project
belongs to a workspace and can optionally have a lead (member or agent).

### List Projects

```bash
chimii project list
chimii project list --status in_progress
chimii project list --output json
```

Available filters: `--status`.

### Get Project

```bash
chimii project get <id>
chimii project get <id> --output json
```

### Create Project

```bash
chimii project create --title "2026 Week 16 Sprint" --icon "🏃" --lead "Lambda"
```

Flags: `--title` (required), `--description`, `--status`, `--icon`, `--lead`, `--start-date`, `--due-date`. Dates are calendar days (`YYYY-MM-DD`).

### Update Project

```bash
chimii project update <id> --title "New title" --status in_progress
chimii project update <id> --lead "Lambda"
chimii project update <id> --due-date 2026-04-15
```

Flags: `--title`, `--description`, `--status`, `--icon`, `--lead`, `--start-date`, `--due-date`. For the date flags, pass an empty string (e.g. `--start-date ""`) to clear the date.

### Change Status

```bash
chimii project status <id> in_progress
```

Valid statuses: `planned`, `in_progress`, `paused`, `completed`, `cancelled`.

### Delete Project

```bash
chimii project delete <id>
```

### Associating Issues with Projects

Use the `--project` flag on `issue create` / `issue update` to attach an issue to a
project, or on `issue list` to filter issues by project:

```bash
chimii issue create --title "Login bug" --project <project-id>
chimii issue update <issue-id> --project <project-id>
chimii issue list --project <project-id>
```

## Setup

```bash
# One-command setup for Chimii Cloud: configure, authenticate, and start the daemon
chimii setup

# For local self-hosted deployments
chimii setup self-host

# Custom ports
chimii setup self-host --port 9090 --frontend-port 4000

# On-premise with custom domains
chimii setup self-host --server-url https://api.example.com --app-url https://app.example.com
```

`chimii setup` configures the CLI, opens your browser for authentication, and starts the daemon — all in one step. Use `chimii setup self-host` to connect to a self-hosted server instead of Chimii Cloud.

## Configuration

### View Config

```bash
chimii config show
```

Shows config file path, server URL, app URL, and default workspace.

### Set Values

```bash
chimii config set server_url https://api.example.com
chimii config set app_url https://app.example.com
chimii config set workspace_id <workspace-id>
```

`config set workspace_id <id>` is the low-level interface — it writes the value verbatim without checking that the workspace exists or that you have access. Prefer `chimii workspace switch <id|slug>` for day-to-day workspace changes; it does both checks before saving.

## Autopilot Commands

Autopilots are scheduled/triggered automations that dispatch agent tasks (either by creating an issue or by running an agent directly).

### List Autopilots

```bash
chimii autopilot list
chimii autopilot list --full-id
chimii autopilot list --status active --output json
```

Autopilot table IDs are short UUID prefixes; follow-up autopilot commands accept copied prefixes when they are unique in the current workspace. Use `--full-id` to print canonical UUIDs.

### Get Autopilot Details

```bash
chimii autopilot get <id>
chimii autopilot get <id> --output json   # includes triggers
```

### Create / Update / Delete

```bash
chimii autopilot create \
  --title "Nightly bug triage" \
  --description "Scan todo issues and prioritize." \
  --agent "Lambda" \
  --mode create_issue \
  --subscriber "Alice"

chimii autopilot update <id> --status paused
chimii autopilot update <id> --description "New prompt"
chimii autopilot update <id> --subscriber "Alice" --subscriber "Bob"
chimii autopilot update <id> --clear-subscribers
chimii autopilot delete <id>
```

`--mode` accepts `create_issue` (creates a new issue on each run and assigns it to the agent) or `run_only` (enqueues a direct agent task without creating an issue). `--agent` accepts either a name or UUID.
`--subscriber` accepts a workspace member name or user ID and may be repeated; on update it replaces the autopilot's subscriber template. Subscribers receive inbox notifications for issues created by a `create_issue` autopilot. Use `--clear-subscribers` to remove all autopilot subscribers.

### Manual Trigger

```bash
chimii autopilot trigger <id>            # Fires the autopilot once, returns the run
```

### Run History

```bash
chimii autopilot runs <id>
chimii autopilot runs <id> --limit 50 --output json
```

### Schedule Triggers

```bash
chimii autopilot trigger-add <autopilot-id> --cron "0 9 * * 1-5" --timezone "America/New_York"
chimii autopilot trigger-update <autopilot-id> <trigger-id> --enabled=false
chimii autopilot trigger-delete <autopilot-id> <trigger-id>
```

Only cron-based `schedule` triggers are currently exposed via the CLI. The data model also defines `webhook` and `api` kinds, but there is no server endpoint that fires them yet, so they're not surfaced here.

## Other Commands

```bash
chimii version              # Show CLI version and commit hash
chimii update               # Update to latest version
chimii agent list           # List agents in the current workspace
```

## Output Formats

Most commands support `--output` with two formats:

- `table` — human-readable table (default for list commands)
- `json` — structured JSON (useful for scripting and automation)

```bash
chimii issue list --output json
chimii daemon status --output json
```

## Error Messages

The CLI funnels command errors returned to the top-level handler through a
single user-facing translation layer (`server/internal/cli/errors.go`) so that
what you see on the terminal is a short, actionable sentence rather than a raw
Go error, an HTTP status line, or an internal `resolve issue: ...` chain. (A
few commands print their own output or run deliberate fast probes — for example
`setup`'s short `/health` reachability check — and don't go through this
layer.) The underlying detail is still available on demand (see `--debug`).

### What you see

- **Friendly, single-line message.** Transport failures (timeout, DNS,
  connection refused, TLS) and HTTP status failures (401/403/404/409/400·422/
  429/5xx) are each rendered as one clear sentence with a next step — for
  example a timeout suggests checking the network or raising
  `CHIMII_HTTP_TIMEOUT`, and a 401 tells you to run `chimii login`.
- **Server-provided validation messages are preserved.** For a 400/422 that
  carries a message from the server, that message is shown verbatim
  (`Invalid request: <server message>`); only when there is none do you get the
  generic "check your values / run with --help" hint.
- **No leaked internals by default.** Raw URLs, status lines, JSON bodies, and
  the internal verb chain are hidden unless you ask for them.

### Language

Messages default to **English**, matching the rest of the CLI's help output.
If a Chinese locale is detected in `LC_ALL`, `LC_MESSAGES`, or `LANG` (in that
precedence order), messages switch to **Chinese**. No flag is needed; set the
locale as usual:

```bash
LANG=zh_CN.UTF-8 chimii issue get MUL-9999   # 错误信息显示为中文
```

### Exit codes

The process exit code is tiered so scripts can branch on the failure class:

| Exit code | Meaning |
| --- | --- |
| `0` | success |
| `1` | generic / unclassified error |
| `2` | network error (timeout, DNS, connection refused, TLS, offline) |
| `3` | authentication / authorization (HTTP 401, 403) |
| `4` | not found (HTTP 404) |
| `5` | validation (HTTP 400, 422) |

```bash
chimii issue get MUL-9999
if [ $? -eq 4 ]; then echo "no such issue"; fi
```

### Seeing the full detail (`--debug`)

Pass the global `--debug` flag (or set `CHIMII_DEBUG=1`) to print the complete
original error chain — the internal verb chain, the request method/path/status,
and the raw server body — underneath the friendly message. Use it when you need
to file a bug or understand exactly what the server returned:

```bash
chimii issue list --debug
CHIMII_DEBUG=1 chimii issue update MUL-1234 --title "x"
```

### Request timeout

API requests use a default timeout of 30 seconds. Override it with
`CHIMII_HTTP_TIMEOUT` when you are on a slow network; it accepts a Go duration
(`45s`, `2m`) or a plain number of seconds (`45`). Command-level deadlines are
always at least this value, so raising it takes effect across all commands.

```bash
CHIMII_HTTP_TIMEOUT=60s chimii issue list
```
