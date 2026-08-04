CREATE UNIQUE INDEX CONCURRENTLY cloud_runtime_session_message_key ON cloud_runtime_session_message (session_id, seq);
