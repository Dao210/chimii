CREATE TABLE circuit_trial (
    id UUID NOT NULL,
    workspace_id UUID NOT NULL,
    parent_user_id UUID NOT NULL,
    actor_key TEXT NOT NULL,
    creation_id UUID NOT NULL,
    document_hash TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    hardware_label TEXT NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('worked', 'needs_help')),
    notes TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
