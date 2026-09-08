-- name: UpsertProductImageSource :one
-- The catalogue sync's only write to this table.
INSERT INTO product_image (product_id, store_chain, position, source_url)
VALUES ($1, sqlc.arg('store_chain')::store_chain, 1, sqlc.arg('source_url')::text)
ON CONFLICT (product_id, store_chain, position) WHERE is_deleted = false DO UPDATE
SET source_url       = EXCLUDED.source_url,
    last_updated_utc = now()
RETURNING *;

-- name: ListProductImageWork :many
-- Every candidate for every product that needs any image work.
SELECT pi.product_image_id,
       pi.product_id,
       pi.store_chain,
       pi.source_url,
       pi.position,
       pi.file_name,
       pi.content_sha256,
       pi.source_etag,
       pi.downloaded_at_utc,
       pi.placeholder_at_utc
FROM product_image pi
JOIN product p USING (product_id)
WHERE pi.is_deleted = false
  AND p.is_deleted = false
  AND pi.product_id IN (
      SELECT due.product_id
      FROM product_image due
      JOIN product dp ON dp.product_id = due.product_id AND dp.is_deleted = false
      WHERE due.is_deleted = false
      GROUP BY due.product_id
      HAVING (
                 max(due.downloaded_at_utc) IS NOT NULL
                 AND max(due.downloaded_at_utc) < sqlc.arg('stale_before')::timestamptz
             )
             OR (
                 max(due.downloaded_at_utc) IS NULL
                 AND (max(due.placeholder_at_utc) IS NULL
                      OR max(due.placeholder_at_utc)
                         < sqlc.arg('retry_placeholders_before')::timestamptz)
             )
      ORDER BY due.product_id
      LIMIT sqlc.arg('max_products')::integer
  )
ORDER BY pi.product_id, pi.store_chain, pi.position;

-- name: UpsertHarvestedImage :one
-- The harvest's write.
INSERT INTO product_image (
    product_id, store_chain, position, source_url,
    file_name, content_sha256, bytes, width, height, downloaded_at_utc, placeholder_at_utc
)
VALUES (
    $1, sqlc.arg('store_chain')::store_chain, sqlc.arg('position')::integer,
    sqlc.arg('source_url')::text,
    sqlc.narg('file_name')::text, sqlc.narg('content_sha256')::text, sqlc.narg('bytes')::integer,
    sqlc.narg('width')::integer, sqlc.narg('height')::integer,
    sqlc.narg('downloaded_at_utc')::timestamptz, sqlc.narg('placeholder_at_utc')::timestamptz
)
ON CONFLICT (product_id, store_chain, position) WHERE is_deleted = false DO UPDATE
SET source_url         = EXCLUDED.source_url,
    file_name          = EXCLUDED.file_name,
    content_sha256     = EXCLUDED.content_sha256,
    source_etag        = EXCLUDED.source_etag,
    bytes              = EXCLUDED.bytes,
    width              = EXCLUDED.width,
    height             = EXCLUDED.height,
    downloaded_at_utc  = EXCLUDED.downloaded_at_utc,
    placeholder_at_utc = EXCLUDED.placeholder_at_utc,
    last_updated_utc   = now()
RETURNING *;

-- name: ListProductImages :many
-- Serving. Only slots with a stored file, in display order.
SELECT position, file_name, content_sha256, bytes, width, height, downloaded_at_utc
FROM product_image
WHERE product_id = $1 AND is_deleted = false AND file_name IS NOT NULL
ORDER BY position;

-- name: FindProductImage :one
SELECT position, file_name, content_sha256, bytes, width, height, downloaded_at_utc
FROM product_image
WHERE product_id = $1
  AND position = sqlc.arg('position')::integer
  AND is_deleted = false
  AND file_name IS NOT NULL;

-- name: ListStoredImagesForProduct :many
-- Every stored file for a product, whichever chain and position it came from.
SELECT product_image_id, store_chain, position, file_name, content_sha256
FROM product_image
WHERE product_id = $1 AND is_deleted = false AND file_name IS NOT NULL
ORDER BY store_chain, position;

-- name: ListOrphanedProductImages :many
-- Stored files whose product has gone or been soft-deleted.
SELECT pi.product_image_id, pi.product_id, pi.file_name
FROM product_image pi
LEFT JOIN product p ON p.product_id = pi.product_id AND p.is_deleted = false
WHERE pi.is_deleted = false
  AND pi.file_name IS NOT NULL
  AND p.product_id IS NULL;

-- name: MarkProductImageDeleted :exec
UPDATE product_image
SET is_deleted = true, last_updated_utc = now()
WHERE product_image_id = $1;

-- name: CountProductImages :one
SELECT count(*) FROM product_image WHERE is_deleted = false AND file_name IS NOT NULL;

-- name: BulkUpsertProductImages :exec
-- One statement for a whole chunk of the harvest.
INSERT INTO product_image (
    product_id, store_chain, position, source_url,
    file_name, content_sha256, source_etag, bytes, width, height, downloaded_at_utc, placeholder_at_utc
)
SELECT x.product_id,
       x.store_chain::store_chain,
       x.position,
       x.source_url,
       x.file_name,
       x.content_sha256,
       x.source_etag,
       x.bytes,
       x.width,
       x.height,
       x.downloaded_at_utc,
       x.placeholder_at_utc
FROM jsonb_to_recordset(sqlc.arg('rows')::jsonb) AS x(
    product_id uuid,
    store_chain text,
    position integer,
    source_url text,
    file_name text,
    content_sha256 text,
    source_etag text,
    bytes integer,
    width integer,
    height integer,
    downloaded_at_utc timestamptz,
    placeholder_at_utc timestamptz
)
ON CONFLICT (product_id, store_chain, position) WHERE is_deleted = false DO UPDATE
SET source_url         = EXCLUDED.source_url,
    file_name          = EXCLUDED.file_name,
    content_sha256     = EXCLUDED.content_sha256,
    source_etag        = EXCLUDED.source_etag,
    bytes              = EXCLUDED.bytes,
    width              = EXCLUDED.width,
    height             = EXCLUDED.height,
    downloaded_at_utc  = EXCLUDED.downloaded_at_utc,
    placeholder_at_utc = EXCLUDED.placeholder_at_utc,
    last_updated_utc   = now();

-- name: BulkMarkProductImagesDeleted :exec
UPDATE product_image
SET is_deleted = true, last_updated_utc = now()
WHERE product_image_id = ANY(sqlc.arg('ids')::uuid[]);
