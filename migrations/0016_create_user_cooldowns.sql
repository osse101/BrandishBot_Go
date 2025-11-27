-- +goose Up
-- Create user_cooldowns table to track per-user, per-action cooldowns
CREATE TABLE IF NOT EXISTS user_cooldowns (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    action_name VARCHAR(50) NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, action_name)
);

-- Create index for efficient cooldown lookups
CREATE INDEX IF NOT EXISTS idx_user_cooldowns_user_action ON user_cooldowns(user_id, action_name);

-- +goose Down
-- Drop the user_cooldowns table
DROP TABLE IF EXISTS user_cooldowns;
