-- +goose Up
CREATE TABLE gambles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    initiator_id UUID NOT NULL REFERENCES users(user_id),
    state TEXT NOT NULL CHECK (state IN ('Created', 'Joining', 'Opening', 'Completed', 'Refunded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    join_deadline TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_gambles_state ON gambles(state);

CREATE TABLE gamble_participants (
    gamble_id UUID REFERENCES gambles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id),
    lootbox_bets JSONB NOT NULL, -- Stores array of {item_id, quantity}
    PRIMARY KEY (gamble_id, user_id)
);

CREATE INDEX idx_gp_gamble_id ON gamble_participants(gamble_id);

CREATE TABLE gamble_opened_items (
    gamble_id UUID REFERENCES gambles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id),
    item_id INT REFERENCES items(item_id),
    value BIGINT NOT NULL,
    UNIQUE (gamble_id, user_id, item_id)
);

CREATE INDEX idx_goi_gamble_id ON gamble_opened_items(gamble_id);

-- +goose Down
DROP TABLE IF EXISTS gamble_opened_items;
DROP TABLE IF EXISTS gamble_participants;
DROP TABLE IF EXISTS gambles;
