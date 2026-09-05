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
    r.semantic_version,
    r.origin_y_offset_ldu,
    r.origin_center_z_offset_ldu,
    r.bounds_json,
    r.connections_json,
    r.occupancy_json
FROM part_definition AS d
JOIN part_catalog_revision AS r ON r.part_key = d.part_key
WHERE r.catalog_version = @catalog_version
  AND d.certification_level IN ('basic', 'advanced', 'certified')
ORDER BY d.popularity_rank, d.part_key;

-- name: GetActiveKitProfileForCatalog :one
SELECT *
FROM kit_profile
WHERE kit_id = @kit_id
  AND catalog_version = @catalog_version
  AND status = 'active'
ORDER BY version DESC
LIMIT 1;

-- name: DeprecateKitProfilesExcept :exec
UPDATE kit_profile
SET status = 'deprecated', updated_at = now()
WHERE kit_id = @kit_id
  AND status = 'active'
  AND catalog_version <> @catalog_version;

-- name: UpsertKitProfile :exec
INSERT INTO kit_profile (
    kit_id,
    version,
    name,
    description,
    source_manifest_kit_id,
    catalog_version,
    part_count,
    status
) VALUES (
    @kit_id,
    @version,
    @name,
    @description,
    @source_manifest_kit_id,
    @catalog_version,
    @part_count,
    @status
)
ON CONFLICT (kit_id, version, catalog_version) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    source_manifest_kit_id = EXCLUDED.source_manifest_kit_id,
    part_count = EXCLUDED.part_count,
    status = EXCLUDED.status,
    updated_at = now();

-- name: DeleteKitProfileParts :exec
DELETE FROM kit_profile_part
WHERE kit_id = @kit_id
  AND kit_version = @kit_version
  AND catalog_version = @catalog_version;

-- name: UpsertKitProfilePart :exec
INSERT INTO kit_profile_part (
    kit_id,
    kit_version,
    catalog_version,
    part_key,
    popularity_rank,
    default_quantity,
    enabled
) VALUES (
    @kit_id,
    @kit_version,
    @catalog_version,
    @part_key,
    @popularity_rank,
    @default_quantity,
    @enabled
)
ON CONFLICT (kit_id, kit_version, catalog_version, part_key) DO UPDATE
SET popularity_rank = EXCLUDED.popularity_rank,
    default_quantity = EXCLUDED.default_quantity,
    enabled = EXCLUDED.enabled,
    updated_at = now();

-- name: CountKitCatalogParts :one
SELECT COUNT(*)
FROM kit_profile_part AS kit_part
JOIN part_definition AS definition
    ON definition.part_key = kit_part.part_key
JOIN part_catalog_revision AS revision
    ON revision.catalog_version = kit_part.catalog_version
   AND revision.part_key = kit_part.part_key
WHERE kit_part.kit_id = @kit_id
  AND kit_part.kit_version = @kit_version
  AND kit_part.catalog_version = @catalog_version
  AND kit_part.enabled
  AND (
    sqlc.arg(search_query)::TEXT = ''
    OR definition.name ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
    OR definition.part_key ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
    OR revision.ldraw_part_id ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
  )
  AND (sqlc.arg(category_filter)::TEXT = '' OR definition.category = sqlc.arg(category_filter)::TEXT)
  AND CASE sqlc.arg(capability_filter)::TEXT
    WHEN 'auto_build' THEN definition.auto_build_eligible
    WHEN 'inventory' THEN definition.certification_level <> 'asset_only'
    WHEN 'preview' THEN definition.certification_level = 'asset_only'
    ELSE TRUE
  END;

-- name: ListKitCatalogCategories :many
SELECT DISTINCT definition.category
FROM kit_profile_part AS kit_part
JOIN part_definition AS definition
    ON definition.part_key = kit_part.part_key
WHERE kit_part.kit_id = @kit_id
  AND kit_part.kit_version = @kit_version
  AND kit_part.catalog_version = @catalog_version
  AND kit_part.enabled
ORDER BY definition.category;

-- name: ListKitCatalogParts :many
SELECT
    definition.part_key,
    definition.name,
    definition.category,
    kit_part.popularity_rank,
    definition.certification_level,
    definition.auto_build_eligible,
    definition.geometry_profile,
    definition.studs_x,
    definition.studs_z,
    definition.plates_y,
    kit_part.default_quantity,
    definition.has_top_studs,
    definition.has_bottom_receptors,
    revision.ldraw_part_id,
    revision.origin_y_offset_ldu,
    revision.origin_center_z_offset_ldu
FROM kit_profile_part AS kit_part
JOIN part_definition AS definition
    ON definition.part_key = kit_part.part_key
JOIN part_catalog_revision AS revision
    ON revision.catalog_version = kit_part.catalog_version
   AND revision.part_key = kit_part.part_key
WHERE kit_part.kit_id = @kit_id
  AND kit_part.kit_version = @kit_version
  AND kit_part.catalog_version = @catalog_version
  AND kit_part.enabled
  AND (
    sqlc.arg(search_query)::TEXT = ''
    OR definition.name ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
    OR definition.part_key ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
    OR revision.ldraw_part_id ILIKE '%' || sqlc.arg(search_query)::TEXT || '%'
  )
  AND (sqlc.arg(category_filter)::TEXT = '' OR definition.category = sqlc.arg(category_filter)::TEXT)
  AND CASE sqlc.arg(capability_filter)::TEXT
    WHEN 'auto_build' THEN definition.auto_build_eligible
    WHEN 'inventory' THEN definition.certification_level <> 'asset_only'
    WHEN 'preview' THEN definition.certification_level = 'asset_only'
    ELSE TRUE
  END
  AND (
    kit_part.popularity_rank > sqlc.arg(after_rank)::INTEGER
    OR (kit_part.popularity_rank = sqlc.arg(after_rank)::INTEGER AND definition.part_key > sqlc.arg(after_part_key)::TEXT)
  )
ORDER BY kit_part.popularity_rank, definition.part_key
LIMIT sqlc.arg(page_limit)::INTEGER;

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
