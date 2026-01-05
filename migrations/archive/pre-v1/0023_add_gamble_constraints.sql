-- +goose Up
-- Add unique constraint to prevent multiple active gambles (Bug #1 fix)
CREATE UNIQUE INDEX idx_gambles_single_active 
    ON gambles(state) 
    WHERE state IN ('Joining', 'Opening');

-- +goose Down
DROP INDEX IF EXISTS idx_gambles_single_active;
