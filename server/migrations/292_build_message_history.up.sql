CREATE INDEX CONCURRENTLY build_message_history_idx ON build_message (conversation_id, sequence DESC);
