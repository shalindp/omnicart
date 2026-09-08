-- +goose Up

CREATE TYPE sync_status AS ENUM ('RUNNING', 'SUCCEEDED', 'FAILED');

-- Tracks catalogue sync runs so a second trigger can be refused while one is in flight,
-- and so a crashed run leaves evidence rather than silently disappearing.
CREATE TABLE sync_run (
    sync_run_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status            sync_status NOT NULL DEFAULT 'RUNNING',
    started_at_utc    timestamptz NOT NULL DEFAULT now(),
    finished_at_utc   timestamptz,
    -- Refreshed periodically while a run works. A running row whose heartbeat has gone
    -- quiet is how we detect a process that died mid-run, rather than blocking forever.
    heartbeat_at_utc  timestamptz NOT NULL DEFAULT now(),
    products_created  integer NOT NULL DEFAULT 0,
    products_matched  integer NOT NULL DEFAULT 0,
    store_products    integer NOT NULL DEFAULT 0,
    delisted          integer NOT NULL DEFAULT 0,
    error_message     text,
    date_created_utc  timestamptz NOT NULL DEFAULT now(),
    last_updated_utc  timestamptz NOT NULL DEFAULT now(),
    is_deleted        boolean NOT NULL DEFAULT false,

    CONSTRAINT sync_run_finished_iff_not_running
        CHECK ((status = 'RUNNING') = (finished_at_utc IS NULL))
);

-- At most one run may be RUNNING at any time, enforced by the database rather than by
-- application logic. Two concurrent triggers cannot both win: the loser gets a unique
-- violation, which is the signal that a run is already in progress.
CREATE UNIQUE INDEX idx_sync_run_single_running
    ON sync_run ((status)) WHERE status = 'RUNNING';

CREATE INDEX idx_sync_run_recent ON sync_run (started_at_utc DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_sync_run_recent;
DROP INDEX IF EXISTS idx_sync_run_single_running;
DROP TABLE IF EXISTS sync_run;
DROP TYPE IF EXISTS sync_status;
