-- Create items table
CREATE TABLE IF NOT EXISTS items (
    item_id SERIAL PRIMARY KEY,
    item_name VARCHAR(255) UNIQUE NOT NULL,
    item_description TEXT,
    base_value INTEGER DEFAULT 0
);
