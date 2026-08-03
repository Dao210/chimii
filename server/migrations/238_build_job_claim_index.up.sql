CREATE INDEX CONCURRENTLY IF NOT EXISTS build_job_claim_idx ON build_job (available_at, created_at) WHERE status IN ('queued', 'running');
