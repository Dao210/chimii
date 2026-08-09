-- Freeze the inventory semantics used by every build session and creation.
-- Existing rows predate configurable inventory and therefore used the
-- unlimited default.
ALTER TABLE build_session
ADD COLUMN inventory_snapshot JSONB NOT NULL DEFAULT '{"configured":false,"catalog_version":"","revision":0,"items":[],"content_hash":""}'::jsonb;

ALTER TABLE build_creation
ADD COLUMN inventory_snapshot JSONB NOT NULL DEFAULT '{"configured":false,"catalog_version":"","revision":0,"items":[],"content_hash":""}'::jsonb;
