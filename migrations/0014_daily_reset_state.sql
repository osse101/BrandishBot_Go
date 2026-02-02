-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS daily_reset_state (
    id INT PRIMARY KEY DEFAULT 1,
    last_reset_time TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    records_affected INT NOT NULL DEFAULT 0,
    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO daily_reset_state (id, last_reset_time, records_affected)
VALUES (1, '1970-01-01 00:00:00+00', 0)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS daily_reset_state;
-- +goose StatementEnd
