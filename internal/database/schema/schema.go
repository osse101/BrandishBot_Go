package schema

// SchemaSQL contains the full database schema initialization script
const SchemaSQL = `
-- Users Table
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    twitch_id VARCHAR(255),
    youtube_id VARCHAR(255),
    discord_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
`
