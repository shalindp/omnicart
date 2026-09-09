-- name: FindStoreByChainAndRegion :one
SELECT * FROM store
WHERE retailer = sqlc.arg('retailer')::store_chain
  AND region_id = sqlc.arg('region_id')::text
  AND is_deleted = false;

-- name: UpsertStore :one
INSERT INTO store (retailer, region_id, external_store_id, name, latitude, longitude)
VALUES (sqlc.arg('retailer')::store_chain, sqlc.arg('region_id')::text, sqlc.arg('external_store_id')::text, sqlc.arg('name')::text, sqlc.arg('latitude')::double precision, sqlc.arg('longitude')::double precision)
ON CONFLICT (external_store_id) WHERE external_store_id IS NOT NULL DO UPDATE
SET name            = EXCLUDED.name,
    retailer        = EXCLUDED.retailer,
    latitude        = EXCLUDED.latitude,
    longitude       = EXCLUDED.longitude,
    last_updated_utc = now(),
    is_deleted       = false
RETURNING *;

-- name: ListStoresByChain :many
SELECT * FROM store
WHERE retailer = sqlc.arg('retailer')::store_chain AND is_deleted = false
ORDER BY region_id;
