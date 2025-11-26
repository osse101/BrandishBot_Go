-- +goose Up
-- Create moderators table
CREATE TABLE IF NOT EXISTS moderators (
    moderator_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
-- Drop moderators table
DROP TABLE IF EXISTS moderators;
