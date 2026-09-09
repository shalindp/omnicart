-- +goose Up

-- ============================================================
-- Enums
-- ============================================================

CREATE TYPE barcode_kind AS ENUM ('GTIN', 'PLU', 'INTERNAL');
CREATE TYPE store_chain AS ENUM ('WOOLWORTHS', 'PAKNSAVE', 'NEWWORLD');
CREATE TYPE sync_status AS ENUM ('RUNNING', 'SUCCEEDED', 'FAILED');

-- ============================================================
-- product
-- ============================================================

CREATE TABLE product (
    product_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    barcode      text,
    barcode_type barcode_kind,
    name         text NOT NULL,
    brand        text,
    pack_size    text,

    CONSTRAINT product_barcode_unique UNIQUE (barcode_type, barcode),
    CONSTRAINT product_barcode_type_set_iff_barcode_set
        CHECK ((barcode IS NULL) = (barcode_type IS NULL)),
    CONSTRAINT product_barcode_type_is_identity
        CHECK (barcode_type IS NULL OR barcode_type IN ('GTIN', 'PLU')),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_product_gtin ON product (barcode) WHERE barcode_type = 'GTIN';

CREATE UNIQUE INDEX idx_product_identity_when_unidentified
    ON product (lower(coalesce(brand, '')), lower(name), lower(coalesce(pack_size, '')))
    WHERE barcode IS NULL;

-- ============================================================
-- store
-- ============================================================

CREATE TABLE store (
    store_id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    retailer          store_chain NOT NULL,
    region_id         text NOT NULL,
    external_store_id text,
    name              text NOT NULL DEFAULT '',
    latitude          double precision,
    longitude         double precision,

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_store_external_id
    ON store (external_store_id)
    WHERE external_store_id IS NOT NULL;

-- ============================================================
-- retailer_session
-- ============================================================

CREATE TABLE retailer_session (
    retailer_session_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    store_chain         store_chain NOT NULL,
    token               text NOT NULL,
    expires_at_utc      timestamptz,

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_retailer_session_current
    ON retailer_session (store_chain) WHERE is_deleted = false;

-- ============================================================
-- store_product
-- ============================================================

CREATE TABLE store_product (
    store_product_id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id          uuid NOT NULL REFERENCES product(product_id),
    store_id            uuid NOT NULL REFERENCES store(store_id),
    external_product_id text NOT NULL,
    unit_of_measure     text NOT NULL DEFAULT 'EACH',
    is_in_stock         boolean,

    CONSTRAINT store_product_external_unique UNIQUE (store_id, external_product_id),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_store_product_product_store ON store_product (product_id, store_id);

-- ============================================================
-- store_product_price
-- ============================================================

CREATE TABLE store_product_price (
    store_product_price_id bigserial PRIMARY KEY,
    store_product_id       uuid NOT NULL REFERENCES store_product(store_product_id),
    store_id               uuid NOT NULL REFERENCES store(store_id),
    price_cents            integer NOT NULL CHECK (price_cents >= 0),
    sale_price_cents       integer CHECK (sale_price_cents >= 0),
    is_sale                boolean NOT NULL
        GENERATED ALWAYS AS (sale_price_cents IS NOT NULL) STORED,
    observed_at_utc        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT store_product_price_sale_is_lower
        CHECK (sale_price_cents IS NULL OR sale_price_cents < price_cents),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_spp_latest
    ON store_product_price (store_product_id, store_id, observed_at_utc DESC)
    WHERE is_deleted = false;

-- ============================================================
-- sync_run
-- ============================================================

CREATE TABLE sync_run (
    sync_run_id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status           sync_status NOT NULL DEFAULT 'RUNNING',
    started_at_utc   timestamptz NOT NULL DEFAULT now(),
    finished_at_utc  timestamptz,
    heartbeat_at_utc timestamptz NOT NULL DEFAULT now(),
    products_created integer NOT NULL DEFAULT 0,
    products_matched integer NOT NULL DEFAULT 0,
    store_products   integer NOT NULL DEFAULT 0,
    delisted         bigint NOT NULL DEFAULT 0,
    error_message    text,

    CONSTRAINT sync_run_finished_iff_not_running
        CHECK ((status = 'RUNNING') = (finished_at_utc IS NULL)),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_sync_run_single_running
    ON sync_run ((status)) WHERE status = 'RUNNING';

CREATE INDEX idx_sync_run_recent ON sync_run (started_at_utc DESC);

-- ============================================================
-- product_image
-- ============================================================

CREATE TABLE product_image (
    product_image_id  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        uuid NOT NULL REFERENCES product(product_id),
    store_chain       store_chain NOT NULL,
    position          integer NOT NULL,
    source_url        text NOT NULL,
    source_etag       text,
    file_name         text,
    content_sha256    text,
    bytes             integer,
    width             integer,
    height            integer,
    downloaded_at_utc timestamptz,
    placeholder_at_utc timestamptz,

    CONSTRAINT product_image_position_positive CHECK (position >= 1),
    CONSTRAINT product_image_file_complete CHECK (
        (file_name IS NULL AND content_sha256 IS NULL AND bytes IS NULL
             AND width IS NULL AND height IS NULL AND downloaded_at_utc IS NULL)
        OR (file_name IS NOT NULL AND content_sha256 IS NOT NULL AND bytes IS NOT NULL
             AND width IS NOT NULL AND height IS NOT NULL AND downloaded_at_utc IS NOT NULL)
    ),
    CONSTRAINT product_image_placeholder_has_no_file
        CHECK (placeholder_at_utc IS NULL OR file_name IS NULL),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_product_image_slot
    ON product_image (product_id, store_chain, position) WHERE is_deleted = false;

CREATE INDEX idx_product_image_current
    ON product_image (product_id, position) WHERE is_deleted = false AND file_name IS NOT NULL;

CREATE INDEX idx_product_image_age ON product_image (downloaded_at_utc);

-- ============================================================
-- category
-- ============================================================

CREATE TABLE category (
    category_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_category_id uuid REFERENCES category(category_id),
    name               text NOT NULL,
    display_name       text,

    CONSTRAINT category_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT category_not_its_own_parent
        CHECK (parent_category_id IS DISTINCT FROM category_id),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_category_child_name
    ON category (parent_category_id, lower(name))
    WHERE is_deleted = false AND parent_category_id IS NOT NULL;

CREATE UNIQUE INDEX idx_category_root_name
    ON category (lower(name))
    WHERE is_deleted = false AND parent_category_id IS NULL;

CREATE INDEX idx_category_children
    ON category (parent_category_id) WHERE is_deleted = false;

-- ============================================================
-- category_normaliser
-- ============================================================

CREATE TABLE category_normaliser (
    category_normaliser_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    paknsave_category      text,
    woolworths_category    text,
    canonical_name         text NOT NULL,

    CONSTRAINT category_normaliser_exactly_one_source CHECK (
        (paknsave_category IS NOT NULL)::int
        + (woolworths_category IS NOT NULL)::int = 1
    ),
    CONSTRAINT category_normaliser_canonical_not_blank
        CHECK (length(btrim(canonical_name)) > 0),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_category_normaliser_paknsave
    ON category_normaliser (lower(paknsave_category))
    WHERE is_deleted = false AND paknsave_category IS NOT NULL;

CREATE UNIQUE INDEX idx_category_normaliser_woolworths
    ON category_normaliser (lower(woolworths_category))
    WHERE is_deleted = false AND woolworths_category IS NOT NULL;

-- Seed Woolworths-to-canonical mappings
INSERT INTO category_normaliser (woolworths_category, canonical_name) VALUES
    ('Pantry',         'Pantry'),
    ('Health & Body',  'Health & Body'),
    ('Frozen',         'Frozen'),
    ('Bakery',         'Bakery'),
    ('Household',      'Household & Cleaning'),
    ('Fridge & Deli',  'Fridge, Deli & Eggs'),
    ('Beer & Wine',    'Beer, Wine & Cider'),
    ('Drinks',         'Hot & Cold Drinks'),
    ('Baby & Child',   'Baby & Toddler'),
    ('Pet',            'Pets'),
    ('Fruit & Veg',    'Fruit & Vegetables'),
    ('Meat & Poultry', 'Meat, Poultry & Seafood'),
    ('Fish & Seafood', 'Meat, Poultry & Seafood');

-- ============================================================
-- product_category
-- ============================================================

CREATE TABLE product_category (
    product_category_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id          uuid NOT NULL REFERENCES product(product_id),
    category_id         uuid NOT NULL REFERENCES category(category_id),

    is_deleted       boolean NOT NULL DEFAULT false,
    date_created_utc timestamptz NOT NULL DEFAULT now(),
    last_updated_utc timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_product_category_product
    ON product_category (product_id) WHERE is_deleted = false;

CREATE INDEX idx_product_category_members
    ON product_category (category_id, product_id) WHERE is_deleted = false;

-- +goose Down

DROP TABLE IF EXISTS product_category;
DROP TABLE IF EXISTS category_normaliser;
DROP TABLE IF EXISTS category;
DROP TABLE IF EXISTS product_image;
DROP TABLE IF EXISTS sync_run;
DROP TABLE IF EXISTS store_product_price;
DROP TABLE IF EXISTS store_product;
DROP TABLE IF EXISTS retailer_session;
DROP TABLE IF EXISTS store;
DROP TABLE IF EXISTS product;

DROP TYPE IF EXISTS sync_status;
DROP TYPE IF EXISTS store_chain;
DROP TYPE IF EXISTS barcode_kind;
