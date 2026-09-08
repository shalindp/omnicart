-- +goose Up

-- A product's category moves out of the product row and becomes a tree.
--
-- product.category held one text value from whichever chain resolved the product first: a
-- Woolworths department label for most, a PAK'nSAVE level-1 name for the rest, and a URL slug
-- wherever the Woolworths fallback fired. Three vocabularies in one column, overwritten on every
-- sync, with no way to ask "everything under Butchery" and no way to say that Ham sits under Pork.
--
-- The existing values are not migrated. They are the mess being replaced, nothing reads them, and
-- one scrape rebuilds the lot.
ALTER TABLE product DROP COLUMN category;

-- The canonical tree.
--
-- Deliberately carries no store or chain reference: a category is a fact about groceries, not
-- about a retailer. Which retailer's words produced a node is recorded in category_normaliser,
-- which is where that question belongs.
CREATE TABLE category (
    category_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Null for a root. An adjacency list rather than a path column or ltree: a few hundred nodes
    -- at most four deep, and the only query needing the whole chain is one recursive walk.
    parent_category_id uuid REFERENCES category(category_id),
    -- Decides identity, so the sync owns it: two chains whose words canonicalise to the same name
    -- land on the same node, which is the whole job of category_normaliser.
    name               text NOT NULL,
    -- An operator's column, and the safe way to rename. Nothing in SqlQueries names it and there
    -- is no UPDATE on this table, so the nightly run cannot reach it -- the rule product.img_url
    -- did not have. Renaming through canonical_name instead would resolve to a different node and
    -- duplicate everything beneath it; this renames in place.
    display_name       text,

    date_created_utc   timestamptz NOT NULL DEFAULT now(),
    last_updated_utc   timestamptz NOT NULL DEFAULT now(),
    is_deleted         boolean NOT NULL DEFAULT false,

    CONSTRAINT category_name_not_blank
        CHECK (length(btrim(name)) > 0),

    CONSTRAINT category_not_its_own_parent
        CHECK (parent_category_id IS DISTINCT FROM category_id)
);

-- Node identity is (parent, name), case-insensitively: "Sauces" under Pantry is not "Sauces"
-- under Fridge & Deli, and that is the point.
--
-- Two indexes rather than one because NULL is not equal to NULL in Postgres. A single unique
-- index on (parent_category_id, lower(name)) constrains children and lets an unlimited number of
-- identically named roots through, which is the exact duplicate this table exists to prevent.
CREATE UNIQUE INDEX idx_category_child_name
    ON category (parent_category_id, lower(name))
    WHERE is_deleted = false AND parent_category_id IS NOT NULL;

CREATE UNIQUE INDEX idx_category_root_name
    ON category (lower(name))
    WHERE is_deleted = false AND parent_category_id IS NULL;

-- Walking down the tree, and the self-FK's own lookups.
CREATE INDEX idx_category_children
    ON category (parent_category_id) WHERE is_deleted = false;

-- Each retailer's word for one level of its own hierarchy, and what this project calls it.
--
-- A column per chain rather than a (store_chain, source_name) pair: the set of chains is fixed and
-- small, the check below makes "exactly one source per row" a database rule instead of a
-- convention, and "everything PAK'nSAVE calls X" reads as one column. The cost is that adding a
-- chain is a migration, which is the honest price of a fixed vocabulary.
--
-- Mapping is per level name, never per path. PAK'nSAVE's "Butchery" and Woolworths'
-- "Meat & Poultry" both canonicalise to "Butchery"; the tree is composed from the mapped names
-- afterwards.
--
-- A row is an override, not a requirement. A name with no row passes through as the retailer
-- wrote it, so every product is categorised from the first run and rows are added only to rename
-- or to merge two chains' words onto one node.
CREATE TABLE category_normaliser (
    category_normaliser_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    paknsave_category      text,
    woolworths_category    text,
    -- What the category table will call it. Matched to a node by name, so it must be spelled
    -- exactly as the canonical vocabulary spells it.
    canonical_name         text NOT NULL,

    date_created_utc       timestamptz NOT NULL DEFAULT now(),
    last_updated_utc       timestamptz NOT NULL DEFAULT now(),
    is_deleted             boolean NOT NULL DEFAULT false,

    -- One row says one thing: "this chain's word for this level is that". A row naming two chains
    -- would be two facts that could not be edited or retired independently.
    CONSTRAINT category_normaliser_exactly_one_source CHECK (
        (paknsave_category IS NOT NULL)::int
        + (woolworths_category IS NOT NULL)::int = 1
    ),

    CONSTRAINT category_normaliser_canonical_not_blank
        CHECK (length(btrim(canonical_name)) > 0)
);

-- One canonical answer per source name, per chain. Case-insensitive for the same reason the
-- category indexes are: the retailers are not consistent about it.
CREATE UNIQUE INDEX idx_category_normaliser_paknsave
    ON category_normaliser (lower(paknsave_category))
    WHERE is_deleted = false AND paknsave_category IS NOT NULL;

CREATE UNIQUE INDEX idx_category_normaliser_woolworths
    ON category_normaliser (lower(woolworths_category))
    WHERE is_deleted = false AND woolworths_category IS NOT NULL;

-- Where a product sits: exactly one node, the innermost the retailers placed it in.
--
-- Not a set of memberships. A product is in one place in one tree and every ancestor is implied
-- by walking up from here, so a second row could only ever contradict the first.
CREATE TABLE product_category (
    product_category_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id          uuid NOT NULL REFERENCES product(product_id),
    category_id         uuid NOT NULL REFERENCES category(category_id),

    date_created_utc    timestamptz NOT NULL DEFAULT now(),
    last_updated_utc    timestamptz NOT NULL DEFAULT now(),
    is_deleted          boolean NOT NULL DEFAULT false
);

-- Exactly one, and what BulkUpsertProductCategories conflicts on.
CREATE UNIQUE INDEX idx_product_category_product
    ON product_category (product_id) WHERE is_deleted = false;

-- Serving the other direction: which products are in a node.
--
-- category_id here holds a mix of depths -- a product carried by both chains gets PAK'nSAVE's
-- leaf, a Woolworths-only product gets a root -- so "every product under Butchery" must walk the
-- descendants. An equality test on this column silently answers a different question.
CREATE INDEX idx_product_category_members
    ON product_category (category_id, product_id) WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_product_category_members;
DROP INDEX IF EXISTS idx_product_category_product;
DROP TABLE IF EXISTS product_category;

DROP INDEX IF EXISTS idx_category_normaliser_woolworths;
DROP INDEX IF EXISTS idx_category_normaliser_paknsave;
DROP TABLE IF EXISTS category_normaliser;

DROP INDEX IF EXISTS idx_category_children;
DROP INDEX IF EXISTS idx_category_root_name;
DROP INDEX IF EXISTS idx_category_child_name;
DROP TABLE IF EXISTS category;

-- Recreated empty, as 00008's down recreates img_url. The values are not recoverable and were
-- not worth keeping.
ALTER TABLE product ADD COLUMN category text;
