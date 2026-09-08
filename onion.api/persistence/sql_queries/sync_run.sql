-- name: StartSyncRun :one
-- Fails with a unique violation if a run is already RUNNING, which is how the caller
-- learns a sync is in progress. Reclaim stale runs first.
INSERT INTO sync_run (status) VALUES ('RUNNING')
RETURNING *;

-- name: FindRunningSyncRun :one
SELECT * FROM sync_run WHERE status = 'RUNNING';

-- name: HeartbeatSyncRun :exec
UPDATE sync_run
SET heartbeat_at_utc = now(), last_updated_utc = now()
WHERE sync_run_id = $1 AND status = 'RUNNING';

-- name: ReclaimStaleSyncRuns :execrows
-- Marks runs whose heartbeat has gone quiet as FAILED, so a process that died mid-run
-- cannot block every future sync.
UPDATE sync_run
SET status           = 'FAILED',
    finished_at_utc  = now(),
    error_message    = 'run abandoned: heartbeat stopped',
    last_updated_utc = now()
WHERE status = 'RUNNING' AND heartbeat_at_utc < $1;

-- name: CompleteSyncRun :one
UPDATE sync_run
SET status           = 'SUCCEEDED',
    finished_at_utc  = now(),
    products_created = $2,
    products_matched = $3,
    store_products   = $4,
    delisted         = $5,
    last_updated_utc = now()
WHERE sync_run_id = $1
RETURNING *;

-- name: FailSyncRun :one
UPDATE sync_run
SET status           = 'FAILED',
    finished_at_utc  = now(),
    error_message    = $2,
    last_updated_utc = now()
WHERE sync_run_id = $1
RETURNING *;

-- name: LatestSyncRun :one
SELECT * FROM sync_run ORDER BY started_at_utc DESC LIMIT 1;
