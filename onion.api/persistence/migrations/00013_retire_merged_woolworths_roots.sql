-- +goose Up

-- Retires the nine roots 00012 emptied.
--
-- Before the seed, each chain's words made their own root, so the tree carried "Household" and
-- "Household & Cleaning" as separate top-level categories describing one shelf. 00012 pointed
-- Woolworths' words at PAK'nSAVE's, every product moved across on the next sync, and these nine
-- were left holding nothing:
--
--   Household -> Household & Cleaning          Beer & Wine    -> Beer, Wine & Cider
--   Fridge & Deli -> Fridge, Deli & Eggs       Drinks         -> Hot & Cold Drinks
--   Baby & Child -> Baby & Toddler             Pet            -> Pets
--   Fruit & Veg -> Fruit & Vegetables          Meat & Poultry -> Meat, Poultry & Seafood
--                                              Fish & Seafood -> Meat, Poultry & Seafood
--
-- Soft-deleted rather than dropped, matching every other table here, and named individually
-- rather than swept by "any root with no products". A generic sweep would also retire a category
-- that happens to be empty this week, and the next sync would rebuild it under a new id -- which
-- is exactly the churn that makes a category_id not worth referencing.
--
-- Safe to re-run against a database that never had them: it matches nothing.
UPDATE category
SET is_deleted = true, last_updated_utc = now()
WHERE parent_category_id IS NULL
  AND is_deleted = false
  AND name IN (
      'Household', 'Fridge & Deli', 'Beer & Wine', 'Drinks', 'Baby & Child',
      'Pet', 'Fruit & Veg', 'Meat & Poultry', 'Fish & Seafood'
  )
  -- Only if it is genuinely empty. If a product still sits here the merge has not run yet, and
  -- retiring the node would orphan it.
  AND NOT EXISTS (
      SELECT 1 FROM product_category pc
      WHERE pc.category_id = category.category_id AND pc.is_deleted = false
  )
  AND NOT EXISTS (
      SELECT 1 FROM category child
      WHERE child.parent_category_id = category.category_id AND child.is_deleted = false
  );

-- +goose Down
UPDATE category
SET is_deleted = false, last_updated_utc = now()
WHERE parent_category_id IS NULL
  AND is_deleted = true
  AND name IN (
      'Household', 'Fridge & Deli', 'Beer & Wine', 'Drinks', 'Baby & Child',
      'Pet', 'Fruit & Veg', 'Meat & Poultry', 'Fish & Seafood'
  );
