CREATE TABLE cloud_runtime_session_message (
    session_id TEXT NOT NULL,
    seq BIGINT NOT NULL,
    role TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
