-- +goose Up
-- Remove deprecated parent_node_id column from progression_nodes
-- This field is no longer used as of v2.0 (replaced by progression_prerequisites junction table)

ALTER TABLE progression_nodes DROP COLUMN IF EXISTS parent_node_id;

-- +goose Down
-- Restore parent_node_id column for rollback
-- Note: Data cannot be recovered, this is for schema compatibility only

ALTER TABLE progression_nodes ADD COLUMN parent_node_id INT REFERENCES progression_nodes(id) ON DELETE SET NULL;
