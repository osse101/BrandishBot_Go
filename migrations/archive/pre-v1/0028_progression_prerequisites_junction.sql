-- +goose Up
-- Add new columns for dynamic cost calculation
ALTER TABLE progression_nodes 
ADD COLUMN tier INT NOT NULL DEFAULT 1,
ADD COLUMN size VARCHAR(20) NOT NULL DEFAULT 'medium',
ADD COLUMN category VARCHAR(50) NOT NULL DEFAULT 'uncategorized';

-- Create junction table for multiple prerequisites
CREATE TABLE progression_prerequisites (
    node_id INT NOT NULL REFERENCES progression_nodes(id) ON DELETE CASCADE,
    prerequisite_node_id INT NOT NULL REFERENCES progression_nodes(id) ON DELETE CASCADE,
    PRIMARY KEY (node_id, prerequisite_node_id),
    -- Prevent self-references
    CHECK (node_id != prerequisite_node_id)
);

-- Create index for reverse lookups (what depends on this node?)
CREATE INDEX idx_progression_prerequisites_prerequisite ON progression_prerequisites(prerequisite_node_id);

-- Migrate existing parent relationships to junction table
INSERT INTO progression_prerequisites (node_id, prerequisite_node_id)
SELECT id, parent_node_id
FROM progression_nodes
WHERE parent_node_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON TABLE progression_prerequisites IS 'Junction table for progression node prerequisites - supports multiple parents (AND logic)';
COMMENT ON COLUMN progression_nodes.tier IS 'Difficulty tier (0=foundation, 1=basic, 2=intermediate, 3=advanced, 4=endgame)';
COMMENT ON COLUMN progression_nodes.size IS 'Node size/scope: small, medium, or large (affects cost multiplier 1:2:4)';
COMMENT ON COLUMN progression_nodes.category IS 'Node category for grouping (e.g., economy, combat, progression)';

-- +goose Down
-- Drop junction table
DROP TABLE IF EXISTS progression_prerequisites;

-- Remove new columns
ALTER TABLE progression_nodes
DROP COLUMN IF EXISTS tier,
DROP COLUMN IF EXISTS size,
DROP COLUMN IF EXISTS category;
