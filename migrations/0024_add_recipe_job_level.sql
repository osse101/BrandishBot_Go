-- +goose Up
-- skip-destructive-check
ALTER TABLE crafting_recipes ADD COLUMN required_job_level INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE crafting_recipes DROP COLUMN required_job_level;
