-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_traps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setter_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    shine_level TEXT NOT NULL DEFAULT 'COMMON',
    timeout_seconds INT NOT NULL DEFAULT 60,
    placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at TIMESTAMPTZ,
    CONSTRAINT one_active_trap_per_target UNIQUE NULLS NOT DISTINCT (target_id, triggered_at)
);

CREATE INDEX idx_user_traps_active_target ON user_traps(target_id)
    WHERE triggered_at IS NULL;
CREATE INDEX idx_user_traps_setter ON user_traps(setter_id);

COMMENT ON TABLE user_traps IS 'Stores active and historical trap placements';
COMMENT ON CONSTRAINT one_active_trap_per_target ON user_traps IS
    'Ensures only one active (untriggered) trap per target user';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_traps;
-- +goose StatementEnd
