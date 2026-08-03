-- name: LockBuildActorCapacity :exec
-- Serialize the active-build count and insert for one parent/child actor. This
-- keeps the two-active-build limit exact under concurrent/retried requests
-- without blocking a sibling profile or another family.
SELECT pg_advisory_xact_lock(hashtextextended(@actor_key::text, 0));

-- name: CreateBuildSession :one
INSERT INTO build_session (
    workspace_id, creator_user_id, child_profile_id, client_request_id, prompt, status, question, answers
) VALUES (
    @workspace_id, @creator_user_id, sqlc.narg(child_profile_id), @client_request_id, @prompt, @status,
    sqlc.narg(question), COALESCE(sqlc.narg(answers), '{}'::jsonb)
)
ON CONFLICT (
    workspace_id,
    creator_user_id,
    client_request_id,
    (COALESCE(child_profile_id, '00000000-0000-0000-0000-000000000000'::uuid))
)
DO UPDATE SET updated_at = build_session.updated_at
RETURNING *;

-- name: GetBuildSessionByClientRequest :one
SELECT * FROM build_session
WHERE workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND client_request_id = @client_request_id
  AND (
    (sqlc.narg(child_profile_id)::uuid IS NULL AND child_profile_id IS NULL)
    OR child_profile_id = sqlc.narg(child_profile_id)
  );

-- name: CountActiveBuildSessions :one
SELECT COUNT(*) FROM build_session
WHERE workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND (
    (sqlc.narg(child_profile_id)::uuid IS NULL AND child_profile_id IS NULL)
    OR child_profile_id = sqlc.narg(child_profile_id)
  )
  AND status IN ('clarifying', 'queued', 'generating')
  AND expires_at > now();

-- name: GetBuildSessionInWorkspace :one
SELECT * FROM build_session
WHERE id = @id
  AND workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id));

-- name: GetBuildSessionForWorker :one
SELECT * FROM build_session WHERE id = @id;

-- name: SubmitBuildSessionAnswers :one
UPDATE build_session
SET answers = @answers,
    status = 'queued',
    question = NULL,
    error = NULL,
    updated_at = now()
WHERE id = @id
  AND workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id))
  AND expires_at > now()
  AND status = 'clarifying'
RETURNING *;

-- name: MarkBuildSessionGenerating :exec
UPDATE build_session
SET status = 'generating', updated_at = now()
WHERE id = @id AND status IN ('queued', 'generating');

-- name: CompleteBuildSession :one
UPDATE build_session
SET status = 'completed', creation_id = @creation_id, error = NULL, updated_at = now()
WHERE id = @id
RETURNING *;

-- name: FailBuildSession :exec
UPDATE build_session
SET status = 'failed', error = @error, updated_at = now()
WHERE id = @id;

-- name: EnqueueBuildJob :one
INSERT INTO build_job (workspace_id, session_id)
VALUES (@workspace_id, @session_id)
ON CONFLICT (session_id) DO UPDATE SET updated_at = build_job.updated_at
RETURNING *;

-- name: ClaimBuildJob :one
WITH candidate AS (
    SELECT id
    FROM build_job
    WHERE available_at <= now()
      AND (
        status = 'queued'
        OR (status = 'running' AND leased_until < now())
      )
    ORDER BY available_at, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE build_job AS job
SET status = 'running',
    attempts = job.attempts + 1,
    lease_token = gen_random_uuid(),
    leased_until = now() + interval '60 seconds',
    updated_at = now()
FROM candidate
WHERE job.id = candidate.id
RETURNING job.*;

-- name: CompleteBuildJob :one
UPDATE build_job
SET status = 'completed', leased_until = NULL, updated_at = now()
WHERE id = @id AND lease_token = @lease_token AND status = 'running'
RETURNING *;

-- name: RetryBuildJob :one
UPDATE build_job
SET status = CASE WHEN attempts >= 3 THEN 'failed' ELSE 'queued' END,
    available_at = CASE WHEN attempts >= 3 THEN available_at ELSE @available_at END,
    leased_until = NULL,
    last_error = @last_error,
    updated_at = now()
WHERE id = @id AND lease_token = @lease_token AND status = 'running'
RETURNING *;

-- name: CreateBuildCreation :one
INSERT INTO build_creation (
    workspace_id, creator_user_id, child_profile_id, session_id, title, prompt, archetype,
    recipe, build_plan, validation, ldraw_mpd
) VALUES (
    @workspace_id, @creator_user_id, sqlc.narg(child_profile_id), @session_id, @title, @prompt, @archetype,
    @recipe, @build_plan, @validation, @ldraw_mpd
)
RETURNING *;

-- name: GetBuildCreationInWorkspace :one
SELECT * FROM build_creation
WHERE id = @id
  AND workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id));

-- name: ListBuildCreations :many
SELECT * FROM build_creation
WHERE workspace_id = @workspace_id
  AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id))
ORDER BY created_at DESC
LIMIT @page_size OFFSET @page_offset;
