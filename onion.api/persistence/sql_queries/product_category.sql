-- name: BulkUpsertProductCategories :exec
-- One statement per chunk of the catalogue write.
INSERT INTO product_category (product_id, category_id)
SELECT x.product_id, x.category_id
FROM jsonb_to_recordset(sqlc.arg('rows')::jsonb) AS x(
    product_id  uuid,
    category_id uuid
)
ON CONFLICT (product_id) WHERE is_deleted = false DO UPDATE
SET category_id      = EXCLUDED.category_id,
    last_updated_utc = now();

-- name: ListProductCategoryPath :many
-- Where a product sits, and every ancestor above it, root first.
WITH RECURSIVE ancestry AS (
    SELECT c.category_id,
           c.parent_category_id,
           c.name::text AS name,
           c.display_name,
           0::integer   AS depth
    FROM product_category pc
    JOIN category c ON c.category_id = pc.category_id AND c.is_deleted = false
    WHERE pc.product_id = $1 AND pc.is_deleted = false

    UNION ALL

    SELECT parent.category_id,
           parent.parent_category_id,
           parent.name::text,
           parent.display_name,
           ancestry.depth + 1
    FROM ancestry
    JOIN category parent
      ON parent.category_id = ancestry.parent_category_id
     AND parent.is_deleted = false
    WHERE ancestry.depth < 16
)
SELECT category_id, parent_category_id, name, display_name, depth
FROM ancestry
ORDER BY depth DESC;

-- name: CountProductCategories :one
SELECT count(*) FROM product_category WHERE is_deleted = false;
