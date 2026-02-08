-- +goose Up
-- +goose StatementBegin

-- Quest definitions table (stores active weekly quests)
CREATE TABLE quests (
    quest_id SERIAL PRIMARY KEY,
    quest_key VARCHAR(100) NOT NULL UNIQUE,
    quest_type VARCHAR(50) NOT NULL, -- 'buy_items', 'sell_items', 'earn_money', 'craft_recipe', 'perform_searches'
    description TEXT NOT NULL,
    target_category VARCHAR(100), -- For: buy_items, sell_items (item category)
    target_recipe_key VARCHAR(100), -- For: craft_recipe (recipe identifier)
    base_requirement INT NOT NULL,
    base_reward_money INT NOT NULL,
    base_reward_xp INT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    week_number INT NOT NULL, -- ISO week number
    year INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quests_active_week ON quests(active, year, week_number);

-- User quest progress tracking
CREATE TABLE quest_progress (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    quest_id INT NOT NULL REFERENCES quests(quest_id) ON DELETE CASCADE,
    progress_current INT NOT NULL DEFAULT 0,
    progress_required INT NOT NULL,
    reward_money INT NOT NULL,
    reward_xp INT NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    claimed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, quest_id)
);

CREATE INDEX idx_quest_progress_user ON quest_progress(user_id);
CREATE INDEX idx_quest_progress_unclaimed ON quest_progress(user_id, completed_at, claimed_at)
    WHERE completed_at IS NOT NULL AND claimed_at IS NULL;

-- Weekly reset state tracking
CREATE TABLE weekly_quest_reset_state (
    id INT PRIMARY KEY DEFAULT 1,
    last_reset_time TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    week_number INT NOT NULL DEFAULT 0,
    year INT NOT NULL DEFAULT 1970,
    quests_generated INT NOT NULL DEFAULT 0,
    progress_reset INT NOT NULL DEFAULT 0,
    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO weekly_quest_reset_state (id, last_reset_time, week_number, year, quests_generated, progress_reset)
VALUES (1, '1970-01-01 00:00:00+00', 0, 1970, 0, 0)
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quest_progress;
DROP TABLE IF EXISTS quests;
DROP TABLE IF EXISTS weekly_quest_reset_state;
-- +goose StatementEnd
