CREATE TABLE kit_profile (
    kit_id TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version > 0),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_manifest_kit_id TEXT NOT NULL,
    catalog_version TEXT NOT NULL,
    part_count INTEGER NOT NULL CHECK (part_count > 0),
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'deprecated')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
