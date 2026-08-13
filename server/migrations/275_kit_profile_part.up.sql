CREATE TABLE kit_profile_part (
    kit_id TEXT NOT NULL,
    kit_version INTEGER NOT NULL CHECK (kit_version > 0),
    catalog_version TEXT NOT NULL,
    part_key TEXT NOT NULL,
    popularity_rank INTEGER NOT NULL CHECK (popularity_rank > 0),
    default_quantity INTEGER NOT NULL DEFAULT 0 CHECK (default_quantity >= 0),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
