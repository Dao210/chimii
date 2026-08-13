INSERT INTO kit_profile (
    kit_id,
    version,
    name,
    description,
    source_manifest_kit_id,
    catalog_version,
    part_count,
    status
)
SELECT
    'chimii-starter-100',
    1,
    'CHIMII Starter Kit 100',
    'The 100 most common parts in the pinned Starter Kit manifest.',
    'chimii-starter-1000-v1',
    release.catalog_version,
    100,
    'active'
FROM ldraw_catalog_release AS release
WHERE release.status = 'active'
  AND (
      SELECT COUNT(*)
      FROM part_catalog_revision AS revision
      JOIN part_definition AS definition
          ON definition.part_key = revision.part_key
      WHERE revision.catalog_version = release.catalog_version
        AND definition.popularity_rank <= 100
  ) = 100
ON CONFLICT (kit_id, version, catalog_version) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    source_manifest_kit_id = EXCLUDED.source_manifest_kit_id,
    part_count = EXCLUDED.part_count,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO kit_profile_part (
    kit_id,
    kit_version,
    catalog_version,
    part_key,
    popularity_rank,
    default_quantity,
    enabled
)
SELECT
    'chimii-starter-100',
    1,
    release.catalog_version,
    definition.part_key,
    definition.popularity_rank,
    definition.default_quantity,
    TRUE
FROM ldraw_catalog_release AS release
JOIN kit_profile AS profile
    ON profile.kit_id = 'chimii-starter-100'
   AND profile.version = 1
   AND profile.catalog_version = release.catalog_version
   AND profile.status = 'active'
JOIN part_catalog_revision AS revision
    ON revision.catalog_version = release.catalog_version
JOIN part_definition AS definition
    ON definition.part_key = revision.part_key
WHERE release.status = 'active'
  AND definition.popularity_rank <= 100
ON CONFLICT (kit_id, kit_version, catalog_version, part_key) DO UPDATE
SET popularity_rank = EXCLUDED.popularity_rank,
    default_quantity = EXCLUDED.default_quantity,
    enabled = EXCLUDED.enabled,
    updated_at = now();
