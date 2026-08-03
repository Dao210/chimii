CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS build_session_client_request_idx
ON build_session (
    workspace_id,
    creator_user_id,
    client_request_id,
    (COALESCE(child_profile_id, '00000000-0000-0000-0000-000000000000'::uuid))
);
