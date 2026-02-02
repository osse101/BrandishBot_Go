-- +goose Up
-- +goose StatementBegin
CREATE TABLE duels (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    challenger_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    opponent_id uuid REFERENCES users(user_id) ON DELETE CASCADE,
    state text NOT NULL CHECK (state IN ('pending', 'accepted', 'in_progress', 'completed', 'declined', 'expired')),
    stakes jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    winner_id uuid REFERENCES users(user_id),
    result_data jsonb
);

CREATE INDEX idx_duels_challenger ON duels(challenger_id);
CREATE INDEX idx_duels_opponent ON duels(opponent_id);
CREATE INDEX idx_duels_state ON duels(state);
CREATE INDEX idx_duels_expires ON duels(expires_at) WHERE state = 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS duels;
-- +goose StatementEnd
