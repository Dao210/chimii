-- Build Studio owns a small durable state machine: a child-language session,
-- a validated immutable creation, and a Postgres-backed compilation job. No
-- foreign keys are used; workspace/user cleanup is performed explicitly by
-- application services according to the repository migration rules.
--
-- Unique and lookup indexes are split into migrations 235-240 and 246-248 so every index
-- is created CONCURRENTLY in its own single-statement file.
CREATE TABLE build_session (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    creator_user_id UUID NOT NULL,
    child_profile_id UUID,
    client_request_id UUID NOT NULL,
    prompt TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'clarifying'
        CHECK (status IN ('clarifying', 'queued', 'generating', 'completed', 'failed')),
    question JSONB,
    answers JSONB NOT NULL DEFAULT '{}'::jsonb,
    creation_id UUID,
    error TEXT,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + interval '30 minutes',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE build_creation (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    creator_user_id UUID NOT NULL,
    child_profile_id UUID,
    session_id UUID NOT NULL,
    title TEXT NOT NULL,
    prompt TEXT NOT NULL,
    archetype TEXT NOT NULL,
    recipe JSONB NOT NULL,
    build_plan JSONB NOT NULL,
    validation JSONB NOT NULL,
    ldraw_mpd TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE build_job (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    session_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    leased_until TIMESTAMPTZ,
    lease_token UUID,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
