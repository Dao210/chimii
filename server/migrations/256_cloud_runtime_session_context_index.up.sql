CREATE INDEX CONCURRENTLY cloud_runtime_session_context_idx ON cloud_runtime_session (runtime_id, agent_id, context_type, context_id, updated_at DESC);
