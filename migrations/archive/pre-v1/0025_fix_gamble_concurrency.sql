-- +goose Up
-- Add partial unique index to ensure only one active gamble exists
CREATE UNIQUE INDEX IF NOT EXISTS idx_gambles_single_active
ON gambles ((1))
WHERE state IN ('Joining', 'Opening');

-- Add unique constraint to prevent users from joining the same gamble multiple times
CREATE UNIQUE INDEX IF NOT EXISTS idx_gamble_participants_unique_user
ON gamble_participants (gamble_id, user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_gambles_single_active;
DROP INDEX IF EXISTS idx_gamble_participants_unique_user;
