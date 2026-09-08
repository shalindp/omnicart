-- name: FindProductByBarcode :one
-- Identity lookup for GTIN and PLU. Namespaced, so a PLU can never match a GTIN.
SELECT * FROM product
WHERE barcode_type = sqlc.arg('barcode_type')::barcode_kind
  AND barcode = sqlc.arg('barcode')::text
  AND is_deleted = false;

-- name: FindProductByIdentity :one
-- Identity lookup for products with no cross-retailer identifier. Mirrors
-- idx_product_identity_when_unidentified.
SELECT * FROM product
WHERE barcode IS NULL
  AND lower(coalesce(brand, '')) = lower(coalesce(sqlc.narg('brand')::text, ''))
  AND lower(name) = lower(sqlc.arg('name')::text)
  AND lower(coalesce(pack_size, '')) = lower(coalesce(sqlc.narg('pack_size')::text, ''))
  AND is_deleted = false;

-- name: InsertProduct :one
INSERT INTO product (barcode, barcode_type, name, brand, pack_size)
VALUES (
    sqlc.narg('barcode')::text,
    sqlc.narg('barcode_type')::barcode_kind,
    $1,
    sqlc.narg('brand')::text,
    sqlc.narg('pack_size')::text
)
RETURNING *;

-- name: UpdateProductDetails :one
-- Refreshes the mutable descriptive fields. Identity columns are never touched: changing
-- a barcode would silently re-point every store_product row at a different product.
UPDATE product
SET name             = $2,
    brand            = sqlc.narg('brand')::text,
    pack_size        = sqlc.narg('pack_size')::text,
    last_updated_utc = now()
WHERE product_id = $1
RETURNING *;

-- name: CountProducts :one
SELECT count(*) FROM product WHERE is_deleted = false;
