CREATE UNIQUE INDEX CONCURRENTLY build_message_event_idx ON build_message (conversation_id, event_key);
