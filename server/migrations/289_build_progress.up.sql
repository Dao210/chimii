ALTER TABLE build_creation
    ADD COLUMN current_step INTEGER NOT NULL DEFAULT 0 CHECK (current_step >= 0),
    ADD COLUMN completed_at TIMESTAMPTZ,
    ADD COLUMN progress_revision INTEGER NOT NULL DEFAULT 0;
