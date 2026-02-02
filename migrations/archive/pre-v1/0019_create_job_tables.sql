-- +goose Up

-- Job definitions table
CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    job_key TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    associated_features TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed initial jobs
INSERT INTO jobs (job_key, display_name, description, associated_features) VALUES
    ('blacksmith', 'Blacksmith', 'Masters of crafting, upgrades, and disassembly', ARRAY['upgrade', 'craft', 'disassemble']),
    ('explorer', 'Explorer', 'Scouts who find extra rewards', ARRAY['search']),
    ('merchant', 'Merchant', 'Traders who get better deals', ARRAY['buy', 'sell']),
    ('gambler', 'Gambler', 'High rollers who win bigger prizes', ARRAY['gamble']),
    ('farmer', 'Farmer', 'Patient cultivators of valuable crops', ARRAY['farm']),
    ('scholar', 'Scholar', 'Contributors to community progress', ARRAY['progression']);

-- User job progress tracking
CREATE TABLE user_jobs (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    job_id INT NOT NULL REFERENCES jobs(id),
    current_xp BIGINT NOT NULL DEFAULT 0,
    current_level INT NOT NULL DEFAULT 0,
    xp_gained_today BIGINT DEFAULT 0,
    last_xp_gain TIMESTAMPTZ,
    PRIMARY KEY (user_id, job_id)
);

CREATE INDEX idx_user_jobs_user ON user_jobs(user_id);
CREATE INDEX idx_user_jobs_level ON user_jobs(current_level DESC);

-- XP event audit log
CREATE TABLE job_xp_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id),
    job_id INT NOT NULL REFERENCES jobs(id),
    xp_amount INT NOT NULL,
    source_type TEXT NOT NULL,
    source_metadata JSONB,
    recorded_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_job_xp_events_user ON job_xp_events(user_id, recorded_at DESC);
CREATE INDEX idx_job_xp_events_job ON job_xp_events(job_id);

-- Configurable bonuses per job level
CREATE TABLE job_level_bonuses (
    id SERIAL PRIMARY KEY,
    job_id INT NOT NULL REFERENCES jobs(id),
    min_level INT NOT NULL,
    bonus_type TEXT NOT NULL,
    bonus_value DECIMAL(10,4) NOT NULL,
    description TEXT,
    UNIQUE (job_id, min_level, bonus_type)
);

-- Seed explorer bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 1, 'bonus_money_chance', 0.25, '25% base chance'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 10, 'bonus_money_chance', 0.35, '35% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 20, 'bonus_money_chance', 0.45, '45% at level 20');

-- Seed gambler bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 1, 'prize_increase', 0.01, '1% prize increase per level'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 10, 'prize_increase', 0.10, '10% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 25, 'prize_increase', 0.25, '25% max at level 25');

-- +goose Down
DROP TABLE IF EXISTS job_level_bonuses;
DROP TABLE IF EXISTS job_xp_events;
DROP TABLE IF EXISTS user_jobs;
DROP TABLE IF EXISTS jobs;
