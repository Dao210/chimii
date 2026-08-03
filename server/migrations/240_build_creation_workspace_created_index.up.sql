CREATE INDEX CONCURRENTLY IF NOT EXISTS build_creation_workspace_created_idx ON build_creation (workspace_id, created_at DESC);
