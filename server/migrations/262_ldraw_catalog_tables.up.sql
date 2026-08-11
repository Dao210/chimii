-- Build catalog delivery now has explicit release tracking and optional per-part
-- persistence rows for canonicalized LDraw metadata and generated assets.
-- No foreign keys or cascade rules are used; application-layer lookups are
-- responsible for relation integrity and cleanup.
CREATE TABLE ldraw_catalog_release (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    catalog_version TEXT NOT NULL,
    release TEXT NOT NULL,
    source_url TEXT NOT NULL,
    archive_sha256 TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'error')),
    release_json TEXT,
    part_count INTEGER NOT NULL DEFAULT 0 CHECK (part_count >= 0),
    downloaded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ldraw_part_revision (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    catalog_version TEXT NOT NULL,
    part_id TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    kind TEXT NOT NULL DEFAULT 'official',
    ldraw_sha256 TEXT NOT NULL,
    storage_backend TEXT NOT NULL DEFAULT 'db',
    content_sha256 TEXT,
    content_type TEXT NOT NULL DEFAULT 'model/gltf-binary',
    storage_key TEXT NOT NULL,
    payload BYTEA,
    payload_size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (payload_size_bytes >= 0),
    payload_format INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
