-- +goose Up

-- Catalogue link only, no price: "this store carries this product". Populated by the cron
-- scrape against one reference store per chain, and includes out-of-stock products.
CREATE TABLE store_product (
    store_product_id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id           uuid NOT NULL REFERENCES product(product_id),
    store_id             uuid NOT NULL REFERENCES store(store_id),
    -- The retailer's own product id, required for on-demand price fetching: those calls
    -- address the retailer's API by their identifier, not ours.
    external_product_id  text NOT NULL,
    unit_of_measure      text NOT NULL DEFAULT 'EACH',
    -- Separate from is_deleted: out of stock is not the same as delisted.
    is_in_stock          boolean,
    date_created_utc     timestamptz NOT NULL DEFAULT now(),
    last_updated_utc     timestamptz NOT NULL DEFAULT now(),
    is_deleted           boolean NOT NULL DEFAULT false,

    CONSTRAINT store_product_product_store_unique UNIQUE (product_id, store_id),
    -- The scraper's upsert key: it knows the retailer's id, not our product_id.
    CONSTRAINT store_product_external_unique UNIQUE (store_id, external_product_id)
);

-- Append-only price observations, written whenever a price is fetched for a user, so the
-- newest row is the current price and older rows are history.
CREATE TABLE store_product_price (
    -- bigserial, not uuid: the only unbounded table here, and a monotonic key keeps
    -- inserts and time-ordered reads cheap.
    store_product_price_id  bigserial PRIMARY KEY,
    store_product_id        uuid NOT NULL REFERENCES store_product(store_product_id),
    -- The store the price was observed at, which is NOT necessarily
    -- store_product.store_id (the reference store the catalogue came from). This column is
    -- what makes per-branch pricing representable off a single scraped catalogue.
    store_id                uuid NOT NULL REFERENCES store(store_id),
    -- Whole cents. PAK'nSAVE returns cents natively; Woolworths returns decimal dollars
    -- and is multiplied by 100 on ingest.
    price_cents             integer NOT NULL CHECK (price_cents >= 0),
    observed_at_utc         timestamptz NOT NULL DEFAULT now(),
    date_created_utc        timestamptz NOT NULL DEFAULT now(),
    last_updated_utc        timestamptz NOT NULL DEFAULT now(),
    is_deleted              boolean NOT NULL DEFAULT false
);

-- No UNIQUE constraint: repeated observations over time are the point of this table.
CREATE INDEX idx_spp_latest
    ON store_product_price (store_product_id, store_id, observed_at_utc DESC)
    WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_spp_latest;
DROP TABLE IF EXISTS store_product_price;
DROP TABLE IF EXISTS store_product;
