CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS child_session_token_hash_idx ON child_session (token_hash);
