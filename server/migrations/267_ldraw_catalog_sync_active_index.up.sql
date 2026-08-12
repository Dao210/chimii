CREATE UNIQUE INDEX CONCURRENTLY ldraw_catalog_sync_active_idx ON ldraw_catalog_sync_job ((true)) WHERE status IN ('queued', 'running');
