-- name: ListMembers :many
SELECT * FROM member
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: GetMember :one
SELECT * FROM member
WHERE id = $1;

-- name: GetMemberByUserAndWorkspace :one
SELECT * FROM member
WHERE user_id = $1 AND workspace_id = $2;

-- name: LockWorkspaceMemberForScopedWrite :one
-- Build Studio and child mode are owned by a workspace member but intentionally
-- carry no database foreign keys. Their writers take this shared row lock in
-- the same transaction as the write. Member revocation takes FOR UPDATE before
-- its cleanup sweep, closing the create/remove race explicitly in the app layer.
SELECT id FROM member
WHERE workspace_id = @workspace_id AND user_id = @user_id
FOR KEY SHARE;

-- name: LockWorkspaceMemberForRevocation :one
-- Acquired before deleting any member-owned FK-free state. It conflicts with
-- LockWorkspaceMemberForScopedWrite, so the cleanup snapshot cannot miss a
-- Build/child-mode write that is still committing.
SELECT id FROM member
WHERE id = @id AND workspace_id = @workspace_id AND user_id = @user_id
FOR UPDATE;

-- name: CreateMember :one
INSERT INTO member (workspace_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateMemberRole :one
UPDATE member SET role = $2
WHERE id = $1
RETURNING *;

-- name: DeleteMember :exec
DELETE FROM member WHERE id = $1;

-- name: ListMembersWithUser :many
SELECT m.id, m.workspace_id, m.user_id, m.role, m.created_at,
       u.name as user_name, u.email as user_email, u.avatar_url as user_avatar_url
FROM member m
JOIN "user" u ON u.id = m.user_id
WHERE m.workspace_id = $1
ORDER BY m.created_at ASC;
