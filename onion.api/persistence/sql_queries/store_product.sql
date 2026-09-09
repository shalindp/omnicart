-- name: UpsertStoreProduct :one
-- Keyed on the retailer's own id, which is what the scraper knows. Deliberately no price:
-- prices live in store_product_price and are fetched on demand.
INSERT INTO store_product (
    product_id, store_id, external_product_id, unit_of_measure, is_in_stock
)
VALUES ($1, $2, $3, $4, sqlc.narg('is_in_stock')::boolean)
ON CONFLICT (store_id, external_product_id) DO UPDATE
SET product_id       = EXCLUDED.product_id,
    unit_of_measure  = EXCLUDED.unit_of_measure,
    is_in_stock      = EXCLUDED.is_in_stock,
    is_deleted       = false,
    last_updated_utc = now()
RETURNING *;

-- name: MarkStoreProductsUnseenAsDeleted :execrows
-- Soft-deletes anything the latest scrape did not touch, so a delisted product stops
-- appearing without the scraper needing to diff the catalogue itself.
UPDATE store_product
SET is_deleted = true, last_updated_utc = now()
WHERE store_id = $1
  AND is_deleted = false
  AND last_updated_utc < $2;

-- name: CountStoreProducts :one
SELECT count(*) FROM store_product
WHERE store_id = $1 AND is_deleted = false;

-- name: FindStoreProductByExternalID :one
SELECT * FROM store_product
WHERE store_id = $1 AND external_product_id = $2 AND is_deleted = false;

-- name: ListKnownBarcodesByChain :many
-- Feeds the barcode cache. Keyed by the retailer's own product id rather than by store,
-- because a barcode is store-independent: what one store resolved is valid for all of that
-- chain's stores.
SELECT DISTINCT ON (sp.external_product_id)
       sp.external_product_id,
       p.barcode,
       p.barcode_type,
       pi.source_url AS img_url
FROM store_product sp
JOIN product p USING (product_id)
JOIN store s USING (store_id)
LEFT JOIN product_image pi
       ON pi.product_id = p.product_id
      AND pi.store_chain = s.retailer
      AND pi.position = 1
      AND pi.is_deleted = false
WHERE s.retailer = sqlc.arg('retailer')::store_chain
  AND p.barcode IS NOT NULL
  AND sp.is_deleted = false
  AND p.is_deleted = false
ORDER BY sp.external_product_id, sp.last_updated_utc DESC;

-- name: BulkRecordStoreProductPrices :execrows
-- Records today's price for a chunk of the catalogue, at most once per store product per day.
INSERT INTO store_product_price (store_product_id, store_id, price_cents, sale_price_cents, observed_at_utc)
SELECT x.store_product_id, x.store_id, x.price_cents, x.sale_price_cents,
       sqlc.arg('observed_at')::timestamptz
FROM jsonb_to_recordset(sqlc.arg('rows')::jsonb) AS x(
    store_product_id uuid,
    store_id         uuid,
    price_cents      integer,
    sale_price_cents integer
)
WHERE NOT EXISTS (
    SELECT 1
    FROM store_product_price recent
    WHERE recent.store_product_id = x.store_product_id
      AND recent.store_id = x.store_id
      AND recent.is_deleted = false
      AND recent.observed_at_utc > sqlc.arg('observed_after')::timestamptz
);

-- name: ListRecentStoreProductPrices :many
-- The price history for one store product at one store, newest first.
SELECT store_product_price_id, store_product_id, store_id, price_cents, sale_price_cents,
       is_sale, observed_at_utc
FROM store_product_price
WHERE store_product_id = $1 AND store_id = $2 AND is_deleted = false
ORDER BY observed_at_utc DESC;
