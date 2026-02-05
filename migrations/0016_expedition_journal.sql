-- +goose Up
-- +goose StatementBegin
CREATE TABLE expedition_journal_entries (
    id SERIAL PRIMARY KEY,
    expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
    turn_number INT NOT NULL,
    encounter_type VARCHAR(30) NOT NULL,
    outcome VARCHAR(10) NOT NULL,
    skill_checked VARCHAR(20),
    skill_passed BOOLEAN,
    primary_member VARCHAR(100),
    narrative TEXT NOT NULL,
    fatigue INT NOT NULL,
    purse INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE INDEX idx_expedition_journal_expedition ON expedition_journal_entries(expedition_id, turn_number);

-- Add columns to expedition_participants for expedition engine results
ALTER TABLE expedition_participants
    ADD COLUMN username VARCHAR(100),
    ADD COLUMN is_leader BOOLEAN DEFAULT false,
    ADD COLUMN job_levels JSONB,
    ADD COLUMN final_money INT DEFAULT 0,
    ADD COLUMN final_xp INT DEFAULT 0,
    ADD COLUMN final_items JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE expedition_participants
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS is_leader,
    DROP COLUMN IF EXISTS job_levels,
    DROP COLUMN IF EXISTS final_money,
    DROP COLUMN IF EXISTS final_xp,
    DROP COLUMN IF EXISTS final_items;

DROP TABLE IF EXISTS expedition_journal_entries;
-- +goose StatementEnd
