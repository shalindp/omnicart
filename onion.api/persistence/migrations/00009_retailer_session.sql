-- +goose Up

-- store_session was keyed by store_id, which does not match what a session actually is.
--
-- PAK'nSAVE's guest token is issued per chain, not per store: the same token serves every
-- PAK'nSAVE store. Worse, it is minted on the very first request -- the one that lists the
-- stores -- so at mint time there is no store row to reference, and the foreign key could not be
-- satisfied. The table was never written to by anything, which is why the mismatch went unnoticed.
DROP INDEX IF EXISTS idx_store_session_live;
DROP TABLE IF EXISTS store_session;

CREATE TABLE retailer_session (
    retailer_session_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_chain         store_chain NOT NULL,
    token               text NOT NULL,
    expires_at_utc      timestamptz,
    date_created_utc    timestamptz NOT NULL DEFAULT now(),
    last_updated_utc    timestamptz NOT NULL DEFAULT now(),
    is_deleted          boolean NOT NULL DEFAULT false
);

-- One live token per chain. Rotation replaces it rather than appending: a superseded token is
-- worthless, and keeping a trail of expired secrets is a liability, not an audit log.
CREATE UNIQUE INDEX idx_retailer_session_current
    ON retailer_session (store_chain) WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_retailer_session_current;
DROP TABLE IF EXISTS retailer_session;

CREATE TABLE store_session (
    store_session_id  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id          uuid NOT NULL REFERENCES store(store_id),
    token             text NOT NULL,
    expires_at_utc    timestamptz,
    date_created_utc  timestamptz NOT NULL DEFAULT now(),
    last_updated_utc  timestamptz NOT NULL DEFAULT now(),
    is_deleted        boolean NOT NULL DEFAULT false
);

CREATE INDEX idx_store_session_live
    ON store_session (store_id, expires_at_utc DESC)
    WHERE is_deleted = false;
