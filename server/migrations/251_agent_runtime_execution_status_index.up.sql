CREATE INDEX CONCURRENTLY agent_runtime_execution_status_idx ON agent_runtime (execution_type, status, workspace_id);
