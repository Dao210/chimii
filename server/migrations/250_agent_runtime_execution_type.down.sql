ALTER TABLE agent_runtime
    DROP CONSTRAINT IF EXISTS agent_runtime_execution_placement_check,
    DROP COLUMN IF EXISTS execution_type;
