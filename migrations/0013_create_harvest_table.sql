-- +goose Up
CREATE TABLE harvest_state (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    last_harvested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS harvest_state;
