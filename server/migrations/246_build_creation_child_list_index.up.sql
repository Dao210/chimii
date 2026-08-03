CREATE INDEX CONCURRENTLY IF NOT EXISTS build_creation_child_list_idx ON build_creation (workspace_id, creator_user_id, child_profile_id, created_at DESC);
