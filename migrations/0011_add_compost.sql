-- +goose Up
-- +goose StatementBegin
CREATE TABLE compost_deposits (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    item_key varchar(255) NOT NULL,
    quantity integer NOT NULL CHECK (quantity > 0),
    deposited_at timestamp with time zone DEFAULT now() NOT NULL,
    ready_at timestamp with time zone NOT NULL,
    harvested_at timestamp with time zone,
    gems_awarded integer,
    metadata jsonb
);

CREATE INDEX idx_compost_user ON compost_deposits(user_id);
CREATE INDEX idx_compost_ready ON compost_deposits(ready_at) WHERE harvested_at IS NULL;
CREATE INDEX idx_compost_active ON compost_deposits(user_id, harvested_at) WHERE harvested_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS compost_deposits;
-- +goose StatementEnd
