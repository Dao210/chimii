-- name: LockBrickInventoryForWorkspace :exec
SELECT pg_advisory_xact_lock(hashtextextended('brick-inventory:' || @workspace_id::text, 0));

-- name: GetBrickInventoryByWorkspace :one
SELECT * FROM brick_inventory WHERE workspace_id = @workspace_id;

-- name: ListBrickInventoryItems :many
SELECT * FROM brick_inventory_item
WHERE inventory_id = @inventory_id
ORDER BY part_key, color_code;

-- name: SaveBrickInventory :one
INSERT INTO brick_inventory (workspace_id, catalog_version, configured, revision, updated_by)
VALUES (@workspace_id, @catalog_version, TRUE, 1, @updated_by)
ON CONFLICT (workspace_id) DO UPDATE
SET catalog_version = EXCLUDED.catalog_version,
    configured = TRUE,
    revision = brick_inventory.revision + 1,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING *;

-- name: ResetBrickInventory :one
INSERT INTO brick_inventory (workspace_id, catalog_version, configured, revision, updated_by)
VALUES (@workspace_id, @catalog_version, FALSE, 1, @updated_by)
ON CONFLICT (workspace_id) DO UPDATE
SET catalog_version = EXCLUDED.catalog_version,
    configured = FALSE,
    revision = brick_inventory.revision + 1,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING *;

-- name: DeleteBrickInventoryItems :exec
DELETE FROM brick_inventory_item WHERE inventory_id = @inventory_id;

-- name: CreateBrickInventoryItem :exec
INSERT INTO brick_inventory_item (inventory_id, part_key, color_code, quantity)
VALUES (@inventory_id, @part_key, @color_code, @quantity);

-- name: DeleteBrickInventoryByWorkspace :exec
WITH deleted_items AS (
    DELETE FROM brick_inventory_item
    WHERE inventory_id IN (
        SELECT id FROM brick_inventory AS inventory
        WHERE inventory.workspace_id = sqlc.arg(workspace_id)
    )
)
DELETE FROM brick_inventory AS inventory
WHERE inventory.workspace_id = sqlc.arg(workspace_id);
