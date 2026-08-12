-- name: GetLatestLDrawCatalogSyncJob :one
SELECT *
FROM ldraw_catalog_sync_job
ORDER BY created_at DESC
LIMIT 1;

-- name: GetActiveLDrawCatalogSyncJob :one
SELECT *
FROM ldraw_catalog_sync_job
WHERE status IN ('queued', 'running')
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateLDrawCatalogSyncJob :one
INSERT INTO ldraw_catalog_sync_job (
    workspace_id,
    requested_by,
    catalog_version,
    kit_id,
    target_part_count
) VALUES (
    @workspace_id,
    @requested_by,
    @catalog_version,
    @kit_id,
    @target_part_count
)
RETURNING *;

-- name: ClaimLDrawCatalogSyncJob :one
WITH candidate AS (
    SELECT id
    FROM ldraw_catalog_sync_job
    WHERE available_at <= now()
      AND (
        status = 'queued'
        OR (status = 'running' AND leased_until < now())
      )
    ORDER BY available_at, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE ldraw_catalog_sync_job AS job
SET status = 'running',
    attempts = job.attempts + 1,
    lease_token = gen_random_uuid(),
    leased_until = now() + interval '20 minutes',
    started_at = COALESCE(job.started_at, now()),
    error = NULL,
    updated_at = now()
FROM candidate
WHERE job.id = candidate.id
RETURNING job.*;

-- name: UpdateLDrawCatalogSyncProgress :one
UPDATE ldraw_catalog_sync_job
SET progress_part_count = @progress_part_count,
    leased_until = now() + interval '20 minutes',
    updated_at = now()
WHERE id = @id
  AND lease_token = @lease_token
  AND status = 'running'
RETURNING *;

-- name: CompleteLDrawCatalogSyncJob :one
UPDATE ldraw_catalog_sync_job
SET status = 'completed',
    progress_part_count = target_part_count,
    lease_token = NULL,
    leased_until = NULL,
    error = NULL,
    completed_at = now(),
    updated_at = now()
WHERE id = @id
  AND lease_token = @lease_token
  AND status = 'running'
RETURNING *;

-- name: RetryLDrawCatalogSyncJob :one
UPDATE ldraw_catalog_sync_job
SET status = CASE WHEN attempts >= 3 THEN 'failed' ELSE 'queued' END,
    available_at = CASE WHEN attempts >= 3 THEN available_at ELSE @available_at END,
    lease_token = NULL,
    leased_until = NULL,
    error = @error,
    completed_at = CASE WHEN attempts >= 3 THEN now() ELSE completed_at END,
    updated_at = now()
WHERE id = @id
  AND lease_token = @lease_token
  AND status = 'running'
RETURNING *;
