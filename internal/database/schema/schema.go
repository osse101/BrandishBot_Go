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

`
