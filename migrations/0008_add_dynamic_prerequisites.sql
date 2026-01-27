-- +goose Up
-- Add dynamic_prerequisites JSONB column to progression_nodes table
ALTER TABLE progression_nodes
ADD COLUMN IF NOT EXISTS dynamic_prerequisites JSONB DEFAULT '[]'::jsonb;

-- Create index for dynamic prerequisites
CREATE INDEX IF NOT EXISTS idx_progression_nodes_dynamic_prerequisites
ON progression_nodes
USING GIN (dynamic_prerequisites);

COMMENT ON COLUMN progression_nodes.dynamic_prerequisites IS
'Dynamic prerequisites stored as JSON array: [{"type":"nodes_unlocked_below_tier","tier":2,"count":5}]';

-- +goose Down
DROP INDEX IF EXISTS idx_progression_nodes_dynamic_prerequisites;
ALTER TABLE progression_nodes DROP COLUMN IF EXISTS dynamic_prerequisites;
