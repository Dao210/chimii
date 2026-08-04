-- name: CreateCloudRuntimeSession :one
WITH locked_workspace AS MATERIALIZED (
    SELECT workspace.id
    FROM workspace
    WHERE workspace.id = @workspace_id
    FOR KEY SHARE
), locked_runtime AS MATERIALIZED (
    SELECT runtime.id
    FROM agent_runtime AS runtime, locked_workspace
    WHERE runtime.id = @runtime_id
      AND runtime.workspace_id = locked_workspace.id
    FOR KEY SHARE OF runtime
), locked_agent AS MATERIALIZED (
    SELECT agent.id
    FROM agent, locked_runtime
    WHERE agent.id = @agent_id
      AND agent.workspace_id = @workspace_id
    FOR KEY SHARE OF agent
), locked_chat_context AS MATERIALIZED (
    SELECT chat.id
    FROM chat_session AS chat, locked_agent
    WHERE @context_type = 'chat'
      AND chat.id::text = @context_id
      AND chat.workspace_id = @workspace_id
    FOR KEY SHARE OF chat
), valid_context AS MATERIALIZED (
    SELECT true AS valid WHERE @context_type <> 'chat'
    UNION ALL
    SELECT true AS valid FROM locked_chat_context
)
INSERT INTO cloud_runtime_session (
    id, workspace_id, runtime_id, agent_id, provider, model,
    context_type, context_id, status
) SELECT
    @id, @workspace_id, @runtime_id, @agent_id, @provider, @model,
    @context_type, @context_id, 'active'
FROM locked_workspace, locked_runtime, locked_agent, valid_context
RETURNING *;

-- name: GetCloudRuntimeSessionForRuntime :one
SELECT *
FROM cloud_runtime_session
WHERE id = @id AND runtime_id = @runtime_id;

-- name: GetLatestCloudRuntimeSessionForContext :one
SELECT *
FROM cloud_runtime_session
WHERE runtime_id = @runtime_id
  AND agent_id = @agent_id
  AND context_type = @context_type
  AND context_id = @context_id
ORDER BY updated_at DESC
LIMIT 1;

-- name: AppendCloudRuntimeSessionMessage :one
WITH locked_session AS MATERIALIZED (
    SELECT crs.id
    FROM cloud_runtime_session AS crs
    WHERE crs.id = @session_id
    FOR UPDATE
), next_message AS (
    SELECT COALESCE(MAX(message.seq), 0) + 1 AS seq
    FROM locked_session
    LEFT JOIN cloud_runtime_session_message AS message
      ON message.session_id = locked_session.id
), inserted_message AS (
    INSERT INTO cloud_runtime_session_message (session_id, seq, role, payload)
    SELECT @session_id, next_message.seq, @role, @payload::jsonb
    FROM locked_session, next_message
    RETURNING *
), touched_session AS (
    UPDATE cloud_runtime_session AS crs
    SET updated_at = now()
    WHERE crs.id = (SELECT inserted_message.session_id FROM inserted_message)
    RETURNING crs.id
)
SELECT inserted_message.*
FROM inserted_message, touched_session;

-- name: TouchCloudRuntimeSession :execrows
UPDATE cloud_runtime_session
SET updated_at = now()
WHERE id = $1;

-- name: ListCloudRuntimeSessionMessages :many
SELECT *
FROM cloud_runtime_session_message
WHERE session_id = $1
ORDER BY seq ASC;

-- name: SetCloudRuntimeSessionStatus :one
UPDATE cloud_runtime_session
SET status = @status,
    last_error = sqlc.narg('last_error'),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteCloudRuntimeSessionMessagesByRuntime :exec
DELETE FROM cloud_runtime_session_message
WHERE session_id IN (
    SELECT id FROM cloud_runtime_session WHERE runtime_id = $1
);

-- name: DeleteCloudRuntimeSessionsByRuntime :exec
DELETE FROM cloud_runtime_session
WHERE runtime_id = $1;

-- name: DeleteCloudRuntimeSessionMessagesByAgents :exec
DELETE FROM cloud_runtime_session_message
WHERE session_id IN (
    SELECT id FROM cloud_runtime_session WHERE agent_id = ANY(@agent_ids::uuid[])
);

-- name: DeleteCloudRuntimeSessionsByAgents :exec
DELETE FROM cloud_runtime_session
WHERE agent_id = ANY(@agent_ids::uuid[]);

-- name: DeleteCloudRuntimeSessionMessagesByWorkspace :exec
DELETE FROM cloud_runtime_session_message
WHERE session_id IN (
    SELECT id FROM cloud_runtime_session WHERE workspace_id = $1
);

-- name: DeleteCloudRuntimeSessionsByWorkspace :exec
DELETE FROM cloud_runtime_session
WHERE workspace_id = $1;

-- name: DeleteCloudRuntimeSessionMessagesByContext :exec
DELETE FROM cloud_runtime_session_message
WHERE session_id IN (
    SELECT id
    FROM cloud_runtime_session
    WHERE context_type = @context_type AND context_id = @context_id
);

-- name: DeleteCloudRuntimeSessionsByContext :exec
DELETE FROM cloud_runtime_session
WHERE context_type = @context_type AND context_id = @context_id;
