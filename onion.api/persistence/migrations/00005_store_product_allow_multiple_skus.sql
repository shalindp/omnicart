-- +goose Up

-- A store can carry one product under more than one SKU: PAK'nSAVE lists per-each and
-- per-kilogram variants of the same item, which share a barcode but differ in
-- unit_of_measure and price. UNIQUE (product_id, store_id) rejected the second variant, so
-- the grain is wrong.
--
-- UNIQUE (store_id, external_product_id) remains and is the correct grain: one row per
-- retailer SKU per store. Readers must therefore expect several store_product rows for a
-- product at one store and choose between them, usually on unit_of_measure.
ALTER TABLE store_product DROP CONSTRAINT store_product_product_store_unique;

-- Still the common lookup, just no longer unique.
CREATE INDEX idx_store_product_product_store ON store_product (product_id, store_id);

-- +goose Down
DROP INDEX IF EXISTS idx_store_product_product_store;
ALTER TABLE store_product
    ADD CONSTRAINT store_product_product_store_unique UNIQUE (product_id, store_id);
