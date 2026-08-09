CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS brick_inventory_item_idx ON brick_inventory_item (inventory_id, part_key, color_code);
