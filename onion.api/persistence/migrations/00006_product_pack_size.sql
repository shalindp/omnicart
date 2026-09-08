-- +goose Up

-- Part of a barcodeless product's identity. Retailers give every size of an item the same
-- name and carry the size separately, so (brand, name) alone merged distinct products: a
-- first run collapsed 485 PAK'nSAVE products this way, including two different "100% Pure
-- Orange Juice" SKUs.
ALTER TABLE product ADD COLUMN pack_size text;

DROP INDEX IF EXISTS idx_product_brand_name_when_unidentified;

CREATE UNIQUE INDEX idx_product_identity_when_unidentified
    ON product (lower(coalesce(brand, '')), lower(name), lower(coalesce(pack_size, '')))
    WHERE barcode IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_product_identity_when_unidentified;

CREATE UNIQUE INDEX idx_product_brand_name_when_unidentified
    ON product (lower(coalesce(brand, '')), lower(name))
    WHERE barcode IS NULL;

ALTER TABLE product DROP COLUMN pack_size;
