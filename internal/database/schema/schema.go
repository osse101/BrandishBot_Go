package schema

// SchemaSQL contains the full database schema initialization script
const SchemaSQL = `
-- Users & Auth Schema

-- 1. Platforms Registry
CREATE TABLE IF NOT EXISTS platforms (
    platform_id SERIAL PRIMARY KEY,
    platform_name VARCHAR(50) UNIQUE NOT NULL
);

-- Seed Platforms
INSERT INTO platforms (platform_name) VALUES ('twitch'), ('youtube'), ('discord') ON CONFLICT DO NOTHING;

-- 2. Core User Information
-- Note: We are dropping the old table to enforce the new schema.
DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. User Platform Links
CREATE TABLE IF NOT EXISTS user_platform_links (
    user_platform_link_id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform_id INTEGER NOT NULL REFERENCES platforms(platform_id) ON DELETE RESTRICT,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    UNIQUE (user_id, platform_id)
);

-- Items Table
CREATE TABLE IF NOT EXISTS items (
    item_id SERIAL PRIMARY KEY,
    item_name VARCHAR(100) NOT NULL,
    item_description TEXT,
    base_value INTEGER DEFAULT 0,
    item_properties JSONB
);

-- Item Types Table
CREATE TABLE IF NOT EXISTS item_types (
    item_type_id SERIAL PRIMARY KEY,
    type_name VARCHAR(50) UNIQUE NOT NULL,
    type_properties JSONB
);

-- Item Type Assignments Table (Many-to-Many)
CREATE TABLE IF NOT EXISTS item_type_assignments (
    item_id INTEGER REFERENCES items(item_id) ON DELETE CASCADE,
    item_type_id INTEGER REFERENCES item_types(item_type_id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, item_type_id)
);

-- User Inventory Table (JSONB)
CREATE TABLE IF NOT EXISTS user_inventory (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    inventory_data JSONB NOT NULL DEFAULT '{"slots": []}'
);

-- Index for fast inventory lookups
CREATE INDEX IF NOT EXISTS idx_inventory_item_id ON user_inventory USING GIN (inventory_data);

-- Seed Item Types
INSERT INTO item_types (type_name) VALUES ('consumable'), ('upgradeable') ON CONFLICT DO NOTHING;

-- Seed Items (Lootboxes)
INSERT INTO items (item_name, item_description, base_value) VALUES 
('lootbox0', 'Junk Lootbox', 10),
('lootbox1', 'Basic Lootbox', 50),
('lootbox2', 'Good Lootbox', 100)
ON CONFLICT DO NOTHING;

-- Assign Types to Items
-- Note: This assumes sequential IDs for simplicity in this seed script. 
-- In a real migration, we'd look up IDs.
-- Assuming lootbox0 is ID 1, lootbox1 is ID 2, lootbox2 is ID 3
-- Assuming consumable is ID 1, upgradeable is ID 2
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, t.item_type_id
FROM items i, item_types t
WHERE i.item_name IN ('lootbox0', 'lootbox1', 'lootbox2')
  AND t.type_name IN ('consumable', 'upgradeable')
ON CONFLICT DO NOTHING;


`
