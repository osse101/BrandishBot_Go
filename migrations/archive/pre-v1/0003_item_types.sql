-- +goose Up
-- Create item_types table
CREATE TABLE IF NOT EXISTS item_types (
    item_type_id SERIAL PRIMARY KEY,
    type_name VARCHAR(100) UNIQUE NOT NULL
);

-- Create item_type_assignments table (many-to-many)
CREATE TABLE IF NOT EXISTS item_type_assignments (
    item_id INTEGER NOT NULL REFERENCES items(item_id) ON DELETE CASCADE,
    item_type_id INTEGER NOT NULL REFERENCES item_types(item_type_id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, item_type_id)
);

-- +goose Down
-- Drop item type tables
DROP TABLE IF EXISTS item_type_assignments;
DROP TABLE IF EXISTS item_types;
