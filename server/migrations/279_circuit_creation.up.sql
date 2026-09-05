-- Independent electronic construction documents. Relationships are managed by
-- application transactions; all indexes are separate concurrent migrations.
CREATE TABLE circuit_creation (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    creator_user_id UUID NOT NULL,
    child_profile_id UUID,
    actor_key TEXT NOT NULL,
    client_request_id UUID NOT NULL,
    request_hash TEXT NOT NULL,
    document JSONB NOT NULL,
    current_step INTEGER NOT NULL DEFAULT 0 CHECK (current_step >= 0),
    observation TEXT NOT NULL DEFAULT 'not_tried' CHECK (observation IN ('not_tried', 'worked', 'needs_help')),
    progress_revision INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
