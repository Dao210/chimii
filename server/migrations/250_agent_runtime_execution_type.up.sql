-- Separate where a runtime is hosted (runtime_mode) from which execution
-- engine it uses. Every historical runtime, including Fleet-hosted daemons,
-- is CLI-backed and therefore keeps the default value.
ALTER TABLE agent_runtime
    ADD COLUMN execution_type TEXT NOT NULL DEFAULT 'cli'
        CHECK (execution_type IN ('cli', 'cloud')),
    ADD CONSTRAINT agent_runtime_execution_placement_check
        CHECK (execution_type = 'cli' OR runtime_mode = 'cloud');
