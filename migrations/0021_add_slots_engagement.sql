-- +goose Up
-- +goose StatementBegin
-- Add engagement weights for slots minigame

INSERT INTO engagement_weights (metric_type, weight, description) VALUES
    ('slots_spin', 1.0, 'Player spun the slots'),
    ('slots_win', 1.0, 'Player won on slots'),
    ('slots_big_win', 10.0, 'Player hit a big win (10x+ payout)'),
    ('slots_jackpot', 200.0, 'Player hit jackpot (50x+ payout)');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove slots engagement weights

DELETE FROM engagement_weights WHERE metric_type IN ('slots_spin', 'slots_win', 'slots_big_win', 'slots_jackpot');

-- +goose StatementEnd
