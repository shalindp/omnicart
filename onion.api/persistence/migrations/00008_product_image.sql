-- +goose Up

-- A product's images move out of the product row entirely.
--
-- product.img_url could hold exactly one URL from whichever chain resolved the product last,
-- which cannot express either of the things now required: that a product has several images,
-- and that more than one chain has a photo of it so a priority order can pick between them.
-- It also meant every sync overwrote the column, so a run where the scraper had no URL blanked
-- a good one.
ALTER TABLE product DROP COLUMN img_url;

CREATE TABLE product_image (
    product_image_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id         uuid NOT NULL REFERENCES product(product_id),
    store_chain        store_chain NOT NULL,
    -- 1-based. The stored file is img_{product_id}_{position}.webp.
    position           integer NOT NULL,
    -- Written by the sync for position 1, and by image discovery for the rest.
    source_url         text NOT NULL,

    -- Everything below belongs to the harvest and is never written by the sync. See the
    -- comment on UpsertProductImageSource in SqlQueries/product_image.sql.
    file_name          text,
    -- Serves as the ETag, and lets a refresh skip rewriting bytes that have not changed.
    content_sha256     text,
    bytes              integer,
    width              integer,
    height             integer,
    downloaded_at_utc  timestamptz,
    -- Set when this chain served a known "Image coming soon" placeholder. No file is written;
    -- the row exists so the next run backs off instead of re-fetching a product the retailer
    -- has no photo of.
    placeholder_at_utc timestamptz,

    date_created_utc   timestamptz NOT NULL DEFAULT now(),
    last_updated_utc   timestamptz NOT NULL DEFAULT now(),
    is_deleted         boolean NOT NULL DEFAULT false,

    CONSTRAINT product_image_position_positive CHECK (position >= 1),

    -- A stored file needs all of its metadata or none of it, so a half-written harvest cannot
    -- masquerade as a usable image.
    CONSTRAINT product_image_file_complete CHECK (
        (file_name IS NULL AND content_sha256 IS NULL AND bytes IS NULL
             AND width IS NULL AND height IS NULL AND downloaded_at_utc IS NULL)
        OR (file_name IS NOT NULL AND content_sha256 IS NOT NULL AND bytes IS NOT NULL
             AND width IS NOT NULL AND height IS NOT NULL AND downloaded_at_utc IS NOT NULL)
    ),

    -- A placeholder and a stored file are mutually exclusive: the whole point is that a
    -- placeholder is never saved.
    CONSTRAINT product_image_placeholder_has_no_file
        CHECK (placeholder_at_utc IS NULL OR file_name IS NULL)
);

-- One row per slot. This is what UpsertProductImageSource and UpsertHarvestedImage conflict on.
CREATE UNIQUE INDEX idx_product_image_slot
    ON product_image (product_id, store_chain, position) WHERE is_deleted = false;

-- Serving: the ordered images a product actually has.
CREATE INDEX idx_product_image_current
    ON product_image (product_id, position) WHERE is_deleted = false AND file_name IS NOT NULL;

-- Freshness sweeps and cleanup.
CREATE INDEX idx_product_image_age ON product_image (downloaded_at_utc);

-- +goose Down
DROP INDEX IF EXISTS idx_product_image_age;
DROP INDEX IF EXISTS idx_product_image_current;
DROP INDEX IF EXISTS idx_product_image_slot;
DROP TABLE IF EXISTS product_image;

ALTER TABLE product ADD COLUMN img_url text;
