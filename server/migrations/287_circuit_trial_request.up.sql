CREATE UNIQUE INDEX CONCURRENTLY circuit_trial_request_idx ON circuit_trial (workspace_id, parent_user_id, actor_key, id);
