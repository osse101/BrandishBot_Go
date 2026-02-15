-- +goose Up
-- +goose StatementBegin
-- skip-destructive-check
ALTER TABLE items ADD COLUMN IF NOT EXISTS content_type text[] NOT NULL DEFAULT '{}';
ALTER TABLE items DROP COLUMN IF EXISTS tier;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE items DROP COLUMN IF EXISTS content_type;
ALTER TABLE items ADD COLUMN IF NOT EXISTS tier integer NOT NULL DEFAULT 0;
-- +goose StatementEnd
