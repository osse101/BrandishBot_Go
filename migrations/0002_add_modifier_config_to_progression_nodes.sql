-- +goose Up
-- Add modifier_config JSONB column to progression_nodes table
ALTER TABLE progression_nodes
ADD COLUMN IF NOT EXISTS modifier_config JSONB;

-- Create index for fast feature_key lookups
CREATE INDEX IF NOT EXISTS idx_progression_nodes_modifier_feature
ON progression_nodes ((modifier_config->>'feature_key'))
WHERE modifier_config IS NOT NULL;

COMMENT ON COLUMN progression_nodes.modifier_config IS 'JSON configuration for value modification (feature_key, modifier_type, per_level_value, base_value, max_value, min_value)';

-- +goose Down
DROP INDEX IF EXISTS idx_progression_nodes_modifier_feature;
ALTER TABLE progression_nodes DROP COLUMN IF EXISTS modifier_config;
