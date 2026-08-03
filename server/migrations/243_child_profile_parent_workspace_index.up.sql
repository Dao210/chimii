CREATE INDEX CONCURRENTLY IF NOT EXISTS child_profile_parent_workspace_idx ON child_profile (parent_user_id, workspace_id, created_at);
