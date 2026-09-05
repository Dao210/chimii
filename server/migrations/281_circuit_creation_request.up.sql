CREATE UNIQUE INDEX CONCURRENTLY circuit_creation_request_idx ON circuit_creation (workspace_id, creator_user_id, actor_key, client_request_id);
