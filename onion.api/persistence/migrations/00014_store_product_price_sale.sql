-- +goose Up

-- Records a discount alongside the shelf price.
--
-- Woolworths publishes originalPrice, salePrice and isSpecial on every search hit; until now the
-- ingest kept salePrice alone, so price_cents silently meant "whatever the product costs today"
-- and the fact that it was discounted -- and from what -- was thrown away. Both numbers are free,
-- and a price history that cannot distinguish a promotion from a price rise is worth much less.
--
-- price_cents becomes the shelf price and sale_price_cents the discounted one, populated only
-- when the discount is real (strictly less than the shelf price). A "sale" at the same price is
-- not a sale, and storing it as one would make every trend line lie.
--
-- Nothing has ever written this table, so redefining price_cents costs nothing.
ALTER TABLE store_product_price
    ADD COLUMN sale_price_cents integer CHECK (sale_price_cents >= 0);

-- Derived, not stored by the writer: is_sale is exactly "there is a sale price", and a column the
-- application had to remember to set is a column that will eventually disagree with the one it
-- describes. Generated means the two cannot drift, and no INSERT anywhere may name it.
ALTER TABLE store_product_price
    ADD COLUMN is_sale boolean NOT NULL
        GENERATED ALWAYS AS (sale_price_cents IS NOT NULL) STORED;

-- A sale above or equal to the shelf price is a contradiction, not a discount.
ALTER TABLE store_product_price
    ADD CONSTRAINT store_product_price_sale_is_lower
        CHECK (sale_price_cents IS NULL OR sale_price_cents < price_cents);

-- +goose Down
ALTER TABLE store_product_price DROP CONSTRAINT IF EXISTS store_product_price_sale_is_lower;
ALTER TABLE store_product_price DROP COLUMN IF EXISTS is_sale;
ALTER TABLE store_product_price DROP COLUMN IF EXISTS sale_price_cents;
