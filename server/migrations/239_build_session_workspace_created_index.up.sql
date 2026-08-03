CREATE INDEX CONCURRENTLY IF NOT EXISTS build_session_actor_active_idx
ON build_session (workspace_id, creator_user_id, child_profile_id, expires_at)
WHERE status IN ('clarifying', 'queued', 'generating');
