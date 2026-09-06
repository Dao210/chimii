-- name: LatestBuildConversationSession :one
SELECT * FROM build_session WHERE conversation_id = @conversation_id
ORDER BY created_at DESC, id DESC LIMIT 1;

-- name: AppendBuildMessage :one
INSERT INTO build_message (conversation_id, session_id, role, kind, content, event_key, request_hash, metadata)
VALUES (@conversation_id, @session_id, @role, @kind, @content, @event_key, @request_hash, @metadata)
ON CONFLICT (conversation_id, event_key) DO UPDATE SET event_key = build_message.event_key
RETURNING *;

-- name: GetBuildMessageByEvent :one
SELECT * FROM build_message WHERE conversation_id = @conversation_id AND event_key = @event_key;

-- name: ListBuildMessages :many
SELECT * FROM build_message WHERE conversation_id = @conversation_id
AND (@before_sequence::bigint = 0 OR sequence < @before_sequence)
ORDER BY sequence DESC LIMIT @page_size;

-- name: SetBuildSessionKind :one
UPDATE build_session SET kind = @kind, recipe = @recipe, updated_at = now() WHERE id = @id RETURNING *;

-- name: CompleteBuildConversationReply :one
UPDATE build_session SET status = 'completed', recipe = @recipe, circuit_creation_id = sqlc.narg(circuit_creation_id), error = NULL, updated_at = now()
WHERE id = @id RETURNING *;

-- name: ListInventionSummaries :many
WITH creations AS (
    SELECT id, title, prompt, 'brick'::text AS kind, created_at, current_step, progress_revision,
        (validation->>'step_count')::integer AS step_count, completed_at IS NOT NULL AS completed
    FROM build_creation b WHERE b.workspace_id = @workspace_id AND b.creator_user_id = @creator_user_id
        AND (sqlc.narg(child_profile_id)::uuid IS NULL OR b.child_profile_id = sqlc.narg(child_profile_id))
    UNION ALL
    SELECT id, document->>'title' AS title, document->>'prompt' AS prompt, 'circuit'::text AS kind, created_at,
        current_step, progress_revision, jsonb_array_length(document->'project'->'steps') AS step_count, observation = 'worked' AS completed
    FROM circuit_creation c WHERE c.workspace_id = @workspace_id AND c.creator_user_id = @creator_user_id
        AND actor_key = @actor_key
)
SELECT * FROM creations WHERE (sqlc.narg(before_time)::timestamptz IS NULL OR (created_at, id, kind) < (sqlc.narg(before_time), sqlc.narg(before_id)::uuid, @before_kind::text))
ORDER BY created_at DESC, id DESC, kind DESC LIMIT @page_size;

-- name: LatestBuildConversationResult :one
SELECT * FROM build_session WHERE conversation_id = @conversation_id AND (creation_id IS NOT NULL OR circuit_creation_id IS NOT NULL)
ORDER BY created_at DESC, id DESC LIMIT 1;
