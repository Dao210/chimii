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

-- name: CountLDrawPartRevisionsByVersion :one
SELECT COUNT(*)
FROM ldraw_part_revision
WHERE catalog_version = @catalog_version;

-- name: UpsertPartDefinition :exec
INSERT INTO part_definition (
    part_key,
    name,
    category,
    popularity_rank,
    certification_level,
    auto_build_eligible,
    geometry_profile,
    studs_x,
    studs_z,
    plates_y,
    default_quantity,
    has_top_studs,
    has_bottom_receptors
) VALUES (
    @part_key,
    @name,
    @category,
    @popularity_rank,
    @certification_level,
    @auto_build_eligible,
    @geometry_profile,
    @studs_x,
    @studs_z,
    @plates_y,
    @default_quantity,
    @has_top_studs,
    @has_bottom_receptors
)
ON CONFLICT (part_key) DO UPDATE
SET name = EXCLUDED.name,
    category = EXCLUDED.category,
    popularity_rank = EXCLUDED.popularity_rank,
    certification_level = EXCLUDED.certification_level,
    auto_build_eligible = EXCLUDED.auto_build_eligible,
    geometry_profile = EXCLUDED.geometry_profile,
    studs_x = EXCLUDED.studs_x,
    studs_z = EXCLUDED.studs_z,
    plates_y = EXCLUDED.plates_y,
    default_quantity = EXCLUDED.default_quantity,
    has_top_studs = EXCLUDED.has_top_studs,
    has_bottom_receptors = EXCLUDED.has_bottom_receptors,
    updated_at = now();

-- name: UpsertPartCatalogRevision :exec
INSERT INTO part_catalog_revision (
    catalog_version,
    part_key,
    ldraw_part_id,
    ldraw_sha256,
    content_sha256,
    semantic_version,
    origin_y_offset_ldu,
    origin_center_z_offset_ldu,
    bounds_json,
    connections_json,
    occupancy_json
) VALUES (
    @catalog_version,
    @part_key,
    @ldraw_part_id,
    @ldraw_sha256,
    @content_sha256,
    @semantic_version,
    @origin_y_offset_ldu,
    @origin_center_z_offset_ldu,
    @bounds_json,
    @connections_json,
    @occupancy_json
)
ON CONFLICT (catalog_version, part_key) DO UPDATE
SET ldraw_part_id = EXCLUDED.ldraw_part_id,
    ldraw_sha256 = EXCLUDED.ldraw_sha256,
    content_sha256 = EXCLUDED.content_sha256,
    semantic_version = EXCLUDED.semantic_version,
    origin_y_offset_ldu = EXCLUDED.origin_y_offset_ldu,
    origin_center_z_offset_ldu = EXCLUDED.origin_center_z_offset_ldu,
    bounds_json = EXCLUDED.bounds_json,
    connections_json = EXCLUDED.connections_json,
    occupancy_json = EXCLUDED.occupancy_json,
    updated_at = now();

-- name: ListCreativeCatalogPartsByVersion :many
SELECT
    d.part_key,
    d.name,
    d.category,
    d.popularity_rank,
    d.certification_level,
    d.auto_build_eligible,
    d.geometry_profile,
    d.studs_x,
    d.studs_z,
    d.plates_y,
    d.default_quantity,
    d.has_top_studs,
    d.has_bottom_receptors,
    r.ldraw_part_id,
    r.origin_y_offset_ldu,
    r.origin_center_z_offset_ldu
FROM part_definition AS d
JOIN part_catalog_revision AS r ON r.part_key = d.part_key
WHERE r.catalog_version = @catalog_version
  AND d.certification_level IN ('basic', 'advanced', 'certified')
ORDER BY d.popularity_rank, d.part_key;

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
