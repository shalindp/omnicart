-- +goose Up

-- The retailer's own ETag for the image we stored.
--
-- content_sha256 is a hash of the bytes *we* hold, which lets us skip rewriting an unchanged file
-- but only after downloading it. This is the retailer's validator, so a refresh can send
-- If-None-Match and be told "304, nothing changed" without transferring the image at all. Product
-- photos rarely change, so on a refresh run most of the traffic disappears.
ALTER TABLE product_image ADD COLUMN source_etag text;

-- +goose Down
ALTER TABLE product_image DROP COLUMN source_etag;
