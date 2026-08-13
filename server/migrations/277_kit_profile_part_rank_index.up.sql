CREATE INDEX CONCURRENTLY kit_profile_part_rank_idx ON kit_profile_part (kit_id, kit_version, catalog_version, popularity_rank, part_key) WHERE enabled;
