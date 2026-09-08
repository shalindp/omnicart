-- +goose Up

-- Only GTINs are safely comparable across retailers: Woolworths issues a private GTIN-13
-- for loose produce where PAK'nSAVE uses a 4-digit PLU.
CREATE TYPE barcode_kind AS ENUM ('GTIN', 'PLU', 'INTERNAL');

CREATE TABLE product (
    product_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Holds only cross-retailer identifiers (GTIN or PLU). A retailer's own internal code
    -- is not an identity, and is already recorded as store_product.external_product_id.
    barcode           text,
    barcode_type      barcode_kind,
    name              text NOT NULL,
    -- Needed to identify barcodeless products: PAK'nSAVE strips the brand out of name
    -- ("Breakfast Cereal", brand "Pams"), so name alone would merge different brands.
    brand             text,
    img_url           text,
    category          text,
    date_created_utc  timestamptz NOT NULL DEFAULT now(),
    last_updated_utc  timestamptz NOT NULL DEFAULT now(),
    is_deleted        boolean NOT NULL DEFAULT false,

    -- Unique per namespace, so ('PLU','4011') and ('GTIN','4011') cannot collide into one
    -- row. Repeated NULLs are permitted, which is what allows barcodeless products.
    CONSTRAINT product_barcode_unique UNIQUE (barcode_type, barcode),

    CONSTRAINT product_barcode_type_set_iff_barcode_set
        CHECK ((barcode IS NULL) = (barcode_type IS NULL)),

    -- Only identity-bearing namespaces belong here.
    CONSTRAINT product_barcode_type_is_identity
        CHECK (barcode_type IS NULL OR barcode_type IN ('GTIN', 'PLU'))
);

CREATE INDEX idx_product_gtin ON product (barcode) WHERE barcode_type = 'GTIN';

-- Barcodeless products are identified by (brand, name), so enforce that in the database
-- rather than trusting the ingest code to look before it inserts.
CREATE UNIQUE INDEX idx_product_brand_name_when_unidentified
    ON product (lower(coalesce(brand, '')), lower(name))
    WHERE barcode IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_product_brand_name_when_unidentified;
DROP INDEX IF EXISTS idx_product_gtin;
DROP TABLE IF EXISTS product;
DROP TYPE IF EXISTS barcode_kind;
