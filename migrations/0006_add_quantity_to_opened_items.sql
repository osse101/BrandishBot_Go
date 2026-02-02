-- +goose Up
ALTER TABLE gamble_opened_items ADD COLUMN quantity INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE gamble_opened_items DROP COLUMN quantity;
