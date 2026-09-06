ALTER TABLE build_session ADD COLUMN conversation_id uuid;
UPDATE build_session SET conversation_id = id;
ALTER TABLE build_session ALTER COLUMN conversation_id SET NOT NULL,
    ADD COLUMN kind text NOT NULL DEFAULT 'brick' CHECK (kind IN ('auto', 'brick', 'circuit')),
    ADD COLUMN circuit_creation_id uuid,
    ADD COLUMN request_hash text NOT NULL DEFAULT '';
CREATE TABLE build_message (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    sequence bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
    conversation_id uuid NOT NULL,
    session_id uuid NOT NULL,
    role text NOT NULL CHECK (role IN ('user', 'assistant')),
    kind text NOT NULL CHECK (kind IN ('message', 'question', 'result', 'error')),
    content text NOT NULL,
    event_key text NOT NULL,
    request_hash text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
