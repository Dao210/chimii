-- name: GetLatestActiveLDrawCatalogRelease :one
SELECT *
FROM ldraw_catalog_release
WHERE status = 'active'
ORDER BY downloaded_at DESC
LIMIT 1;

-- name: GetLDrawPartRevisionByVersionAndPartID :one
SELECT *
FROM ldraw_part_revision
WHERE catalog_version = @catalog_version
  AND part_id = @part_id
ORDER BY revision DESC
LIMIT 1;

-- name: GetLDrawCatalogReleaseByVersion :one
SELECT *
FROM ldraw_catalog_release
WHERE catalog_version = @catalog_version;

-- name: DeprecateLDrawCatalogReleasesExcept :exec
UPDATE ldraw_catalog_release
SET status = 'deprecated', updated_at = now()
WHERE status = @status
  AND catalog_version <> @catalog_version;

-- name: UpsertLDrawCatalogRelease :one
INSERT INTO ldraw_catalog_release (
    catalog_version,
    release,
    source_url,
    archive_sha256,
    status,
    release_json,
    part_count,
    downloaded_at,
    updated_at
) VALUES (
    @catalog_version,
    @release,
    @source_url,
    @archive_sha256,
    @status,
    @release_json,
    @part_count,
    now(),
    now()
)
ON CONFLICT (catalog_version) DO UPDATE
SET release = EXCLUDED.release,
    source_url = EXCLUDED.source_url,
    archive_sha256 = EXCLUDED.archive_sha256,
    status = EXCLUDED.status,
    release_json = EXCLUDED.release_json,
    part_count = EXCLUDED.part_count,
    downloaded_at = EXCLUDED.downloaded_at,
    updated_at = now()
RETURNING *;

-- name: UpsertLDrawPartRevision :one
INSERT INTO ldraw_part_revision (
    catalog_version,
    part_id,
    revision,
    kind,
    ldraw_sha256,
    storage_backend,
    content_sha256,
    content_type,
    storage_key,
    payload,
    payload_size_bytes,
    payload_format
) VALUES (
    @catalog_version,
    @part_id,
    @revision,
    @kind,
    @ldraw_sha256,
    @storage_backend,
    @content_sha256,
    @content_type,
    @storage_key,
    @payload,
    @payload_size_bytes,
    @payload_format
)
ON CONFLICT (catalog_version, part_id, revision) DO UPDATE
SET kind = EXCLUDED.kind,
    ldraw_sha256 = EXCLUDED.ldraw_sha256,
    storage_backend = EXCLUDED.storage_backend,
    content_sha256 = EXCLUDED.content_sha256,
    content_type = EXCLUDED.content_type,
    storage_key = EXCLUDED.storage_key,
    payload = EXCLUDED.payload,
    payload_size_bytes = EXCLUDED.payload_size_bytes,
    payload_format = EXCLUDED.payload_format,
    updated_at = now()
RETURNING *;
