-- name: CreateCircuitCreation :one
INSERT INTO circuit_creation (workspace_id, creator_user_id, child_profile_id, actor_key, client_request_id, request_hash, document)
VALUES (@workspace_id, @creator_user_id, sqlc.narg(child_profile_id), @actor_key, @client_request_id, @request_hash, @document)
ON CONFLICT (workspace_id, creator_user_id, actor_key, client_request_id) DO NOTHING
RETURNING *;

-- name: GetCircuitCreation :one
SELECT * FROM circuit_creation
WHERE id = @id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id AND actor_key = @actor_key;

-- name: GetCircuitCreationByRequest :one
SELECT * FROM circuit_creation
WHERE client_request_id = @client_request_id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id AND actor_key = @actor_key;

-- name: ListCircuitCreations :many
SELECT id, document->>'title' AS title, document->'project'->>'id' AS project_id, observation, current_step, created_at
FROM circuit_creation
WHERE workspace_id = @workspace_id AND creator_user_id = @creator_user_id AND actor_key = @actor_key
ORDER BY created_at DESC, id DESC
LIMIT 50;

-- name: UpdateCircuitProgress :one
UPDATE circuit_creation
SET current_step = @current_step, observation = @observation, progress_revision = progress_revision + 1, updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id AND actor_key = @actor_key
  AND progress_revision = @expected_revision
  AND @current_step::integer >= 0
  AND @current_step::integer < jsonb_array_length(document->'project'->'steps')
RETURNING *;
