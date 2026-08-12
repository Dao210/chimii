CREATE TABLE part_catalog_revision (
    catalog_version TEXT NOT NULL,
    part_key TEXT NOT NULL,
    ldraw_part_id TEXT NOT NULL,
    ldraw_sha256 TEXT NOT NULL,
    content_sha256 TEXT,
    semantic_version INTEGER NOT NULL DEFAULT 1 CHECK (semantic_version > 0),
    origin_y_offset_ldu INTEGER NOT NULL DEFAULT 0,
    origin_center_z_offset_ldu INTEGER NOT NULL DEFAULT 0,
    bounds_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    connections_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    occupancy_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
