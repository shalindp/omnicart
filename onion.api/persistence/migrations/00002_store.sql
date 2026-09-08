-- +goose Up

CREATE TYPE store_chain AS ENUM ('WOOLWORTHS', 'PAKNSAVE', 'NEWWORLD');

CREATE TABLE store (
    store_id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_name        store_chain NOT NULL,
    -- The retailer's own location id. text because formats differ: PAK'nSAVE uses a GUID,
    -- Woolworths a numeric pickup address id.
    region_id         text NOT NULL,
    date_created_utc  timestamptz NOT NULL DEFAULT now(),
    last_updated_utc  timestamptz NOT NULL DEFAULT now(),
    is_deleted        boolean NOT NULL DEFAULT false,

    CONSTRAINT store_chain_region_unique UNIQUE (store_name, region_id)
);

-- Guest API tokens. PAK'nSAVE requires one and it expires after ~30 minutes; Woolworths
-- needs none, so it simply has no rows.
CREATE TABLE store_session (
    store_session_id  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id          uuid NOT NULL REFERENCES store(store_id),
    token             text NOT NULL,
    expires_at_utc    timestamptz,
    date_created_utc  timestamptz NOT NULL DEFAULT now(),
    last_updated_utc  timestamptz NOT NULL DEFAULT now(),
    is_deleted        boolean NOT NULL DEFAULT false
);

-- No UNIQUE(store_id): rotation is an append, keeping a short trail of which token was
-- used when. This partial index keeps "current live token" cheap.
CREATE INDEX idx_store_session_live
    ON store_session (store_id, expires_at_utc DESC)
    WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_store_session_live;
DROP TABLE IF EXISTS store_session;
DROP TABLE IF EXISTS store;
DROP TYPE IF EXISTS store_chain;
