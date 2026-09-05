CREATE UNIQUE INDEX CONCURRENTLY circuit_inventory_scope_idx ON circuit_inventory (workspace_id, parent_user_id, kit_id);
