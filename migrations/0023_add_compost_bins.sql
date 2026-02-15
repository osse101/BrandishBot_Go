-- +goose Up
-- +goose StatementBegin
-- skip-destructive-check
DROP TABLE IF EXISTS compost_deposits;

CREATE TABLE compost_bins (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status varchar(20) NOT NULL DEFAULT 'idle'
        CHECK (status IN ('idle', 'composting', 'ready', 'sludge')),
    capacity integer NOT NULL DEFAULT 5,
    items jsonb NOT NULL DEFAULT '[]'::jsonb,
    item_count integer NOT NULL DEFAULT 0,
    started_at timestamptz,
    ready_at timestamptz,
    sludge_at timestamptz,
    input_value integer NOT NULL DEFAULT 0,
    dominant_type varchar(100) NOT NULL DEFAULT '',
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT one_bin_per_user UNIQUE (user_id)
);

CREATE INDEX idx_compost_bins_user ON compost_bins(user_id);
CREATE INDEX idx_compost_bins_status ON compost_bins(status) WHERE status != 'idle';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS compost_bins;
-- +goose StatementEnd
