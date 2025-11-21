-- Create user_inventory table with JSONB
CREATE TABLE IF NOT EXISTS user_inventory (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    inventory_data JSONB DEFAULT '{"slots": []}'::jsonb
);

-- Index for fast inventory lookups
CREATE INDEX IF NOT EXISTS idx_inventory_item_id ON user_inventory USING GIN (inventory_data);
