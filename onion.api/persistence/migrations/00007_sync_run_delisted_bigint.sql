-- +goose Up

-- Delisting counts rows affected by an UPDATE, which Npgsql reports as a long. Holding it
-- in an integer column forced a narrowing cast at the call site, so a count past 2^31 would
-- have been recorded as a wrong number rather than failing -- the one outcome a sync
-- statistic must never have. The other counters stay integer: they are incremented by
-- in-process int counters and cannot exceed one.
ALTER TABLE sync_run ALTER COLUMN delisted TYPE bigint;

-- +goose Down
ALTER TABLE sync_run ALTER COLUMN delisted TYPE integer;
