CREATE INDEX CONCURRENTLY circuit_creation_list_idx ON circuit_creation (workspace_id, creator_user_id, actor_key, created_at DESC, id DESC);
