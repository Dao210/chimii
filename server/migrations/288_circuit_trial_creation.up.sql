CREATE INDEX CONCURRENTLY circuit_trial_creation_idx ON circuit_trial (workspace_id, parent_user_id, actor_key, creation_id, created_at DESC, id DESC);
