DROP TABLE build_message;
ALTER TABLE build_session DROP COLUMN conversation_id, DROP COLUMN kind, DROP COLUMN circuit_creation_id, DROP COLUMN request_hash;
