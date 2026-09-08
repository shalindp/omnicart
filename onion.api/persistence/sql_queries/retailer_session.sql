-- name: UpsertRetailerSession :one
-- Rotation replaces the live token rather than appending a new row.
INSERT INTO retailer_session (store_chain, token, expires_at_utc)
VALUES (sqlc.arg('store_chain')::store_chain, sqlc.arg('token')::text, sqlc.narg('expires_at_utc')::timestamptz)
ON CONFLICT (store_chain) WHERE is_deleted = false DO UPDATE
SET token            = EXCLUDED.token,
    expires_at_utc   = EXCLUDED.expires_at_utc,
    last_updated_utc = now()
RETURNING *;

-- name: FindLiveRetailerSession :one
-- A NULL expiry means the retailer stated none, so it is usable until something rejects it.
SELECT store_chain, token, expires_at_utc
FROM retailer_session
WHERE store_chain = sqlc.arg('store_chain')::store_chain
  AND is_deleted = false
  AND (expires_at_utc IS NULL OR expires_at_utc > sqlc.arg('now')::timestamptz);

-- name: DeleteRetailerSession :exec
UPDATE retailer_session
SET is_deleted = true, last_updated_utc = now()
WHERE store_chain = sqlc.arg('store_chain')::store_chain AND is_deleted = false;
