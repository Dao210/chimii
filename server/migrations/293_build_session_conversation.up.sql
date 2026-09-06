CREATE INDEX CONCURRENTLY build_session_conversation_idx ON build_session (conversation_id, created_at DESC, id DESC);
