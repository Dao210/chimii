-- Workspace-scoped physical brick inventory. The absence of a
-- brick_inventory row means the workspace has never customized inventory.
-- Persisted configured=false rows also mean unlimited mode while preserving
-- a monotonic revision for conflict detection.
-- Relationships are enforced by application transactions; no foreign keys.
CREATE TABLE brick_inventory (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    catalog_version TEXT NOT NULL,
    configured BOOLEAN NOT NULL DEFAULT TRUE,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE brick_inventory_item (
    inventory_id UUID NOT NULL,
    part_key TEXT NOT NULL,
    color_code INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
