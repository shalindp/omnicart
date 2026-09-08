-- name: FindStoreByChainAndRegion :one
SELECT * FROM store
WHERE store_name = sqlc.arg('store_name')::store_chain
  AND region_id = sqlc.arg('region_id')::text
  AND is_deleted = false;

-- name: UpsertStore :one
INSERT INTO store (store_name, region_id)
VALUES (sqlc.arg('store_name')::store_chain, sqlc.arg('region_id')::text)
ON CONFLICT (store_name, region_id) DO UPDATE
SET last_updated_utc = now(),
    is_deleted       = false
RETURNING *;

-- name: ListStoresByChain :many
SELECT * FROM store
WHERE store_name = sqlc.arg('store_name')::store_chain AND is_deleted = false
ORDER BY region_id;
