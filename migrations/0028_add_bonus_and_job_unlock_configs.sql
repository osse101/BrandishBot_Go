-- +goose Up
-- Unified modifiers table
CREATE TABLE bonus_config (
    id SERIAL PRIMARY KEY,
    node_key VARCHAR(100) NOT NULL, -- The source: e.g. "upgrade_progression_basic" or "job_farmer"
    source_type VARCHAR(50) NOT NULL, -- 'progression' (global) or 'job' (per-user)
    feature_key VARCHAR(100) NOT NULL, -- The bonus being applied: e.g. "progression_rate", "harvest_yield"
    modifier_type VARCHAR(20) NOT NULL, -- "multiplicative", "linear", etc.
    base_value NUMERIC(10, 4) NOT NULL DEFAULT 0,
    per_level_value NUMERIC(10, 4) NOT NULL DEFAULT 0,
    max_value NUMERIC(10, 4),
    min_value NUMERIC(10, 4),
    UNIQUE(node_key, feature_key)
);

CREATE INDEX idx_bonus_config_feature ON bonus_config(feature_key);

-- SubFeature locking specifically for jobs
CREATE TABLE job_unlock_config (
    id SERIAL PRIMARY KEY,
    job_key VARCHAR(50) NOT NULL,
    feature_key VARCHAR(100) NOT NULL, -- e.g., "compost", "disassemble"
    required_level INT NOT NULL,
    UNIQUE(job_key, feature_key)
);

-- Extract existing progression modifiers to unified table
INSERT INTO bonus_config (
    node_key, source_type, feature_key, modifier_type, base_value, per_level_value, max_value, min_value
)
SELECT 
    node_key,
    'progression',
    modifier_config->>'feature_key',
    modifier_config->>'modifier_type',
    COALESCE(CAST(modifier_config->>'base_value' AS NUMERIC), 0),
    COALESCE(CAST(modifier_config->>'per_level_value' AS NUMERIC), 0),
    CAST(modifier_config->>'max_value' AS NUMERIC),
    CAST(modifier_config->>'min_value' AS NUMERIC)
FROM progression_nodes
WHERE modifier_config IS NOT NULL;

-- Remove old JSONB column
ALTER TABLE progression_nodes DROP COLUMN IF EXISTS modifier_config;
-- Remove old job level bonuses table
DROP TABLE IF EXISTS job_level_bonuses;

-- +goose Down
-- Restore progression_nodes modifier_config column
ALTER TABLE progression_nodes ADD COLUMN IF NOT EXISTS modifier_config JSONB;

-- Re-populate progression_nodes modifier_config
UPDATE progression_nodes
SET modifier_config = json_build_object(
    'feature_key', bc.feature_key,
    'modifier_type', bc.modifier_type,
    'base_value', bc.base_value,
    'per_level_value', bc.per_level_value,
    'max_value', bc.max_value,
    'min_value', bc.min_value
)
FROM bonus_config bc
WHERE progression_nodes.node_key = bc.node_key
  AND bc.source_type = 'progression';

-- Drop the unified tables
DROP TABLE IF EXISTS job_unlock_config;
DROP TABLE IF EXISTS bonus_config;

-- Restore job_level_bonuses table
CREATE TABLE public.job_level_bonuses (
    id SERIAL PRIMARY KEY,
    job_id integer NOT NULL,
    min_level integer NOT NULL,
    bonus_type text NOT NULL,
    bonus_value numeric(10,4) NOT NULL,
    description text
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
