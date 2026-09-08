-- name: ListCategoryNormalisers :many
-- Every retailer word and what it canonicalises to, read once per write alongside the tree.
SELECT paknsave_category, woolworths_category, canonical_name
FROM category_normaliser
WHERE is_deleted = false;
