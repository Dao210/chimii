# Runtime execution types

Chimii supports two explicit execution engines. They can coexist in one
workspace, but a task never falls back from one engine to the other.

| User-facing type | Execution path | Providers |
| --- | --- | --- |
| CLI runtime (default) | local or Fleet-hosted daemon → `server/pkg/agent` | every installed CLI backend |
| Cloud runtime (optional) | API worker → `server/runtime` | Anthropic and OpenAI only |

`agent_runtime.runtime_mode` continues to describe placement (`local` or
`cloud`). `agent_runtime.execution_type` describes the engine (`cli` or
`cloud`). A Fleet machine therefore remains `runtime_mode=cloud` and
`execution_type=cli`; a server SDK runtime uses `cloud/cloud`.

## Enable Cloud runtime

Cloud execution is off by default. At minimum, enable it, select the providers,
and configure a key for every selected provider:

```dotenv
CHIMII_CLOUD_RUNTIME_ENABLED=true
CHIMII_CLOUD_RUNTIME_PROVIDERS=anthropic,openai
CHIMII_CLOUD_RUNTIME_ANTHROPIC_API_KEY=...
CHIMII_CLOUD_RUNTIME_ANTHROPIC_DEFAULT_MODEL=...
CHIMII_CLOUD_RUNTIME_OPENAI_API_KEY=...
CHIMII_CLOUD_RUNTIME_OPENAI_DEFAULT_MODEL=...
```

Keep `CHIMII_RUNTIME_DEFAULT=cli` unless new onboarding flows should prefer
Cloud runtime. Changing the default does not move existing agents or queued
tasks. Users must explicitly rebind an agent, and the first task on the new
runtime starts a new session.

## Current Cloud capability

The current Cloud runtime is deliberately text-only. The SDK worker restores
durable conversation context and streams/persists model messages, but it starts
with all host shell and filesystem tools disabled. It is suitable for chat,
analysis, and preparing proposed issue content; it must not claim that it
inspected a repository, ran commands, changed files, or created an issue.

Tool-capable Cloud execution remains a separate rollout gate. It may only be
enabled after command execution is moved out of the API server into a
fail-closed sandbox with filesystem, environment, network, and resource
isolation. This restriction does not affect CLI runtimes, whose tools continue
to run through the existing daemon/provider path.

Each enabled Cloud provider currently exposes only its configured
`DEFAULT_MODEL`. The server rejects arbitrary model IDs supplied outside the
UI; expanding the catalog requires an explicit deployment allowlist rather
than provider-side guessing or fallback.

See [the technical design](../tasks/technical-design-runtime-types.md) for the
full data, lifecycle, security, and rollout contract.
