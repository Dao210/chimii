-- name: LockBuildActorCapacity :exec
-- Serialize the active-build count and insert for one parent/child actor. This
-- keeps the two-active-build limit exact under concurrent/retried requests
-- without blocking a sibling profile or another family.
SELECT pg_advisory_xact_lock(hashtextextended(@actor_key::text, 0));

-- name: CreateBuildSession :one
WITH new_session AS (SELECT gen_random_uuid() AS id)
INSERT INTO build_session (
    id, conversation_id, kind, request_hash,
    workspace_id, creator_user_id, child_profile_id, client_request_id, prompt, status, question, answers,
    inventory_snapshot, recipe, phase
) SELECT id, COALESCE(sqlc.narg(conversation_id)::uuid, id) AS conversation_id,
    COALESCE(NULLIF(@kind::text, ''), 'brick') AS kind, @request_hash::text AS request_hash,
    @workspace_id::uuid AS workspace_id, @creator_user_id::uuid AS creator_user_id,
    sqlc.narg(child_profile_id)::uuid AS child_profile_id, @client_request_id::uuid AS client_request_id,
    @prompt::text AS prompt, @status::text AS status, sqlc.narg(question)::jsonb AS question,
    COALESCE(sqlc.narg(answers)::jsonb, '{}'::jsonb) AS answers, @inventory_snapshot::jsonb AS inventory_snapshot,
    sqlc.narg(recipe)::jsonb AS recipe, CASE WHEN @direct_compile::boolean THEN 'compiling' ELSE 'planning' END AS phase
FROM new_session
ON CONFLICT (
    workspace_id,
    creator_user_id,
    client_request_id,
    (COALESCE(child_profile_id, '00000000-0000-0000-0000-000000000000'::uuid))
)
DO UPDATE SET updated_at = build_session.updated_at
RETURNING build_session.*;

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
SET answers = answers || @answers::jsonb,
    revision = revision + 1,
    phase = 'planning',
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
  AND revision = @revision
RETURNING *;

-- name: MarkBuildSessionGenerating :one
UPDATE build_session s
SET status = 'generating', updated_at = now()
WHERE s.id = @id AND s.revision = @revision AND s.status IN ('queued', 'generating')
  AND EXISTS (SELECT 1 FROM build_job j WHERE j.session_id = s.id AND j.status = 'running' AND j.lease_token = @lease_token AND j.leased_until > clock_timestamp())
RETURNING s.*;

-- name: SaveBuildSessionRecipe :one
UPDATE build_session s
SET recipe = @recipe, phase = 'compiling', updated_at = now()
WHERE s.id = @id AND s.revision = @revision AND s.status = 'generating'
  AND EXISTS (SELECT 1 FROM build_job j WHERE j.session_id = s.id AND j.status = 'running' AND j.lease_token = @lease_token AND j.leased_until > clock_timestamp())
RETURNING s.*;

-- name: PauseBuildSession :one
UPDATE build_session
SET status = 'clarifying', question = @question, recipe = @recipe, phase = 'planning', updated_at = now()
WHERE id = @id AND revision = @revision AND status = 'generating'
RETURNING *;

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
ON CONFLICT (session_id) DO UPDATE
SET status = 'queued', attempts = 0, available_at = now(), leased_until = NULL, lease_token = NULL, last_error = NULL, updated_at = now()
WHERE build_job.status IN ('completed', 'failed')
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
WHERE id = @id AND lease_token = @lease_token AND status = 'running' AND leased_until > clock_timestamp()
RETURNING *;

-- name: FailBuildJob :one
UPDATE build_job
SET status = 'failed', leased_until = NULL, last_error = @last_error, updated_at = now()
WHERE id = @id AND lease_token = @lease_token AND status = 'running' AND leased_until > clock_timestamp()
RETURNING *;

-- name: RetryBuildJob :one
UPDATE build_job
SET status = CASE WHEN attempts >= 3 THEN 'failed' ELSE 'queued' END,
    available_at = CASE WHEN attempts >= 3 THEN available_at ELSE @available_at END,
    leased_until = NULL,
    last_error = @last_error,
    updated_at = now()
WHERE id = @id AND lease_token = @lease_token AND status = 'running' AND leased_until > clock_timestamp()
RETURNING *;

-- name: CreateBuildCreation :one
INSERT INTO build_creation (
    workspace_id, creator_user_id, child_profile_id, session_id, title, prompt, archetype,
    recipe, build_plan, validation, ldraw_mpd, inventory_snapshot
) VALUES (
    @workspace_id, @creator_user_id, sqlc.narg(child_profile_id), @session_id, @title, @prompt, @archetype,
    @recipe, @build_plan, @validation, @ldraw_mpd, @inventory_snapshot
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

-- name: LockBuildSessionForAnswer :one
SELECT * FROM build_session
WHERE id = @id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id
 AND ((sqlc.narg(child_profile_id)::uuid IS NULL AND child_profile_id IS NULL) OR child_profile_id = sqlc.narg(child_profile_id))
FOR UPDATE;

-- name: SaveBuildSessionMessage :one
UPDATE build_session SET recipe = @recipe, updated_at = now()
WHERE id = @id AND revision = @revision AND status = 'generating'
RETURNING *;

-- name: LockBuildSessionForWorker :one
SELECT * FROM build_session WHERE id = @id FOR UPDATE;

-- name: CancelBuildJob :exec
UPDATE build_job SET status = 'completed', lease_token = NULL, leased_until = NULL, updated_at = now()
WHERE session_id = @session_id AND status IN ('queued', 'running');

-- name: LockBuildJobLease :one
SELECT id FROM build_job
WHERE id = @id AND lease_token = @lease_token AND status = 'running' AND leased_until > clock_timestamp()
FOR UPDATE;

-- name: ListBuildCreationSummaries :many
SELECT id, title, prompt, archetype, validation, current_step, completed_at, progress_revision, created_at
FROM build_creation
WHERE workspace_id = @workspace_id AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id))
ORDER BY created_at DESC, id DESC
LIMIT @page_size;

-- name: GetBuildProgress :one
SELECT id, current_step, completed_at, progress_revision, validation
FROM build_creation
WHERE id = @id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id));

-- name: UpdateBuildProgress :one
UPDATE build_creation
SET current_step = @current_step,
    completed_at = CASE WHEN @completed::boolean THEN COALESCE(completed_at, now()) ELSE completed_at END,
    progress_revision = progress_revision + 1, updated_at = now()
WHERE id = @id AND workspace_id = @workspace_id AND creator_user_id = @creator_user_id
  AND (sqlc.narg(child_profile_id)::uuid IS NULL OR child_profile_id = sqlc.narg(child_profile_id))
  AND progress_revision = @expected_revision
  AND @current_step::integer BETWEEN 1 AND (validation->>'step_count')::integer
  AND (NOT @completed::boolean OR @current_step::integer = (validation->>'step_count')::integer)
RETURNING id, current_step, completed_at, progress_revision, validation;
