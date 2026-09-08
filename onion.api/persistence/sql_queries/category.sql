-- name: ListCategories :many
-- The whole tree, read once at the start of a catalogue write.
SELECT category_id, parent_category_id, name, display_name
FROM category
WHERE is_deleted = false
ORDER BY category_id;

-- name: InsertCategory :one
-- Creates one node. The sync's only write to this table.
INSERT INTO category (parent_category_id, name)
VALUES (sqlc.narg('parent_category_id')::uuid, sqlc.arg('name')::text)
RETURNING category_id, parent_category_id, name, display_name;

-- name: CountCategories :one
SELECT count(*) FROM category WHERE is_deleted = false;
