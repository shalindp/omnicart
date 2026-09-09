-- +goose Up

ALTER TABLE store
    DROP CONSTRAINT IF EXISTS store_chain_region_unique;

ALTER TABLE store
    RENAME COLUMN store_name TO retailer;

ALTER TABLE store
    ADD COLUMN external_store_id text,
    ADD COLUMN name              text NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_store_retailer_region
    ON store (retailer, region_id);

CREATE UNIQUE INDEX idx_store_external_id
    ON store (external_store_id)
    WHERE external_store_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_store_external_id;
DROP INDEX IF EXISTS idx_store_retailer_region;

ALTER TABLE store
    DROP COLUMN IF EXISTS external_store_id,
    DROP COLUMN IF EXISTS name;

ALTER TABLE store
    RENAME COLUMN retailer TO store_name;

ALTER TABLE store
    ADD CONSTRAINT store_chain_region_unique UNIQUE (store_name, region_id);
