CREATE TABLE cloud_runtime_session (
    id TEXT NOT NULL,
    workspace_id UUID NOT NULL,
    runtime_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('anthropic', 'openai')),
    model TEXT NOT NULL DEFAULT '',
    context_type TEXT NOT NULL CHECK (context_type IN ('issue', 'chat', 'autopilot', 'quick_create', 'direct')),
    context_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'failed', 'cancelled')),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (id LIKE 'crs_%')
);
