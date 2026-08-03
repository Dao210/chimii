-- name: CreateChildProfile :one
INSERT INTO child_profile (workspace_id, parent_user_id, display_name, avatar_seed, pin_hash)
VALUES (@workspace_id, @parent_user_id, @display_name, @avatar_seed, @pin_hash)
RETURNING *;

-- name: ListChildProfiles :many
SELECT * FROM child_profile
WHERE workspace_id = @workspace_id AND parent_user_id = @parent_user_id AND enabled
ORDER BY created_at;

-- name: GetChildProfileForParent :one
SELECT * FROM child_profile
WHERE id = @id AND workspace_id = @workspace_id AND parent_user_id = @parent_user_id AND enabled;

-- name: CreateChildSession :one
INSERT INTO child_session (token_hash, profile_id, workspace_id, parent_user_id, expires_at)
VALUES (@token_hash, @profile_id, @workspace_id, @parent_user_id, @expires_at)
RETURNING *;

-- name: GetActiveChildSessionByTokenHash :one
SELECT cs.*, cp.display_name, cp.avatar_seed, cp.pin_hash
FROM child_session cs
JOIN child_profile cp ON cp.id = cs.profile_id
WHERE cs.token_hash = @token_hash
  AND cs.revoked_at IS NULL
  AND cs.expires_at > now()
  AND cp.enabled;

-- name: GetActiveChildSessionByID :one
SELECT cs.*, cp.display_name, cp.avatar_seed, cp.pin_hash
FROM child_session cs
JOIN child_profile cp ON cp.id = cs.profile_id
WHERE cs.id = @id
  AND cs.revoked_at IS NULL
  AND cs.expires_at > now()
  AND cp.enabled;

-- name: TouchChildSession :exec
UPDATE child_session SET last_used_at = now()
WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeChildSession :exec
UPDATE child_session SET revoked_at = now()
WHERE id = @id AND revoked_at IS NULL;

-- name: RecordChildUnlockFailure :one
UPDATE child_session
SET unlock_failures = unlock_failures + 1,
    unlock_locked_until = CASE WHEN unlock_failures + 1 >= 5 THEN now() + interval '5 minutes' ELSE unlock_locked_until END
WHERE id = @id AND revoked_at IS NULL
RETURNING *;

-- name: ResetChildUnlockFailures :exec
UPDATE child_session SET unlock_failures = 0, unlock_locked_until = NULL
WHERE id = @id AND revoked_at IS NULL;

-- name: DeleteBuildAndChildDataForParentInWorkspace :exec
-- Member removal must explicitly erase all parent-owned FK-free state in this
-- workspace. The caller first takes LockWorkspaceMemberForRevocation so none
-- of these same-member rows can commit behind this statement's snapshot.
WITH target_build_sessions AS (
    SELECT bs.id FROM build_session AS bs
    WHERE bs.workspace_id = @workspace_id AND bs.creator_user_id = @parent_user_id
),
deleted_build_jobs AS (
    DELETE FROM build_job AS bj
    WHERE bj.workspace_id = @workspace_id
      AND bj.session_id IN (SELECT id FROM target_build_sessions)
),
deleted_build_creations AS (
    DELETE FROM build_creation AS bc
    WHERE bc.workspace_id = @workspace_id AND bc.creator_user_id = @parent_user_id
),
deleted_build_sessions AS (
    DELETE FROM build_session AS bs
    WHERE bs.workspace_id = @workspace_id AND bs.creator_user_id = @parent_user_id
),
deleted_child_sessions AS (
    DELETE FROM child_session AS cs
    WHERE cs.workspace_id = @workspace_id AND cs.parent_user_id = @parent_user_id
)
DELETE FROM child_profile AS cp
WHERE cp.workspace_id = @workspace_id AND cp.parent_user_id = @parent_user_id;
