-- +goose Up

-- Woolworths' departments, mapped onto PAK'nSAVE's vocabulary.
--
-- Only level 0 is seeded, and only Woolworths' side of it. Woolworths publishes exactly one level,
-- so the root is the only depth at which the two chains describe the same shelf in different
-- words; below it there is one vocabulary and the resolver adopts PAK'nSAVE's own. A row is still
-- how a subcategory gets renamed later, it is simply not required to have one.
--
-- PAK'nSAVE's names are the canonical side because it is the deeper source: it supplies all three
-- levels for every product it carries, so its wording is what the rest of the tree is already
-- written in. Naming a Woolworths word here is what stops the same shelf existing twice.
--
-- Taken from a real discovery run against an unseeded normaliser: 686 nodes across 22 roots, of
-- which these were the pairs describing the same thing. Counts in the comments are that run's
-- products, as a record of what each row is worth.
--
-- Spelled exactly as CategoryName.Normalise leaves a name -- trimmed, single-spaced. The lookup is
-- by normalised name and no constraint can check that for you.
--
-- No ON CONFLICT: goose runs a migration once, and this is the first thing to write these rows.

INSERT INTO category_normaliser (woolworths_category, canonical_name) VALUES
    -- Names both chains already spell identically. Listed rather than left to coincidence, so the
    -- full Woolworths mapping reads in one place instead of nine rows and four silent matches.
    ('Pantry',         'Pantry'),                    -- 3324 + 3223
    ('Health & Body',  'Health & Body'),             -- 2760 + 765
    ('Frozen',         'Frozen'),                    --  595 + 448
    ('Bakery',         'Bakery'),                    --  320 + 277

    -- The same shelf under different words.
    ('Household',      'Household & Cleaning'),      -- 1585 -> 731
    ('Fridge & Deli',  'Fridge, Deli & Eggs'),       -- 1169 -> 769
    ('Beer & Wine',    'Beer, Wine & Cider'),        --  902 -> 745
    ('Drinks',         'Hot & Cold Drinks'),         --  798 -> 783
    ('Baby & Child',   'Baby & Toddler'),            --  441 -> 175
    ('Pet',            'Pets'),                      --  387 -> 274
    ('Fruit & Veg',    'Fruit & Vegetables'),        --  321 -> 156

    -- Two Woolworths departments, one PAK'nSAVE root. Many-to-one is the normal case for a
    -- mapping table and needs nothing special: both rows simply name the same canonical.
    ('Meat & Poultry', 'Meat, Poultry & Seafood'),   --  223 -> 423
    ('Fish & Seafood', 'Meat, Poultry & Seafood');   --   26 -> 423

-- Not seeded, on purpose:
--
--   PAK'nSAVE's own roots and every level below them. They pass through as written, which is the
--   whole point of the normaliser being an override rather than a requirement -- 681 names were
--   adopted on the discovery run and every product was categorised regardless.
--
--   "Snacks, Treats & Easy Meals". A PAK'nSAVE root with no Woolworths counterpart, so there is
--   nothing to map it to.
--
-- Woolworths keeps Frozen and Fridge & Deli apart and so does PAK'nSAVE, so neither collapses and
-- no detail is lost in the translation.

-- +goose Down
DELETE FROM category_normaliser
WHERE woolworths_category IN (
    'Pantry', 'Health & Body', 'Frozen', 'Bakery', 'Household', 'Fridge & Deli', 'Beer & Wine',
    'Drinks', 'Baby & Child', 'Pet', 'Fruit & Veg', 'Meat & Poultry', 'Fish & Seafood'
);
