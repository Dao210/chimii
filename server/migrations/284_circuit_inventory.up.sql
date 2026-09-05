CREATE TABLE circuit_inventory (
    workspace_id UUID NOT NULL,
    parent_user_id UUID NOT NULL,
    kit_id TEXT NOT NULL,
    catalog_version TEXT NOT NULL,
    quantities JSONB NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
