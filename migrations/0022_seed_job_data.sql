-- +goose Up
-- Seed initial jobs
INSERT INTO jobs (job_key, display_name, description, associated_features) VALUES
    ('blacksmith', 'Blacksmith', 'Masters of crafting, upgrades, and disassembly', ARRAY['upgrade', 'craft', 'disassemble']),
    ('explorer', 'Explorer', 'Scouts who find extra rewards', ARRAY['search']),
    ('merchant', 'Merchant', 'Traders who get better deals', ARRAY['buy', 'sell']),
    ('gambler', 'Gambler', 'High rollers who win bigger prizes', ARRAY['gamble']),
    ('farmer', 'Farmer', 'Patient cultivators of valuable crops', ARRAY['farm']),
    ('scholar', 'Scholar', 'Contributors to community progress', ARRAY['progression'])
ON CONFLICT (job_key) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    associated_features = EXCLUDED.associated_features;

-- Seed explorer bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 1, 'bonus_money_chance', 0.25, '25% base chance'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 10, 'bonus_money_chance', 0.35, '35% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'explorer'), 20, 'bonus_money_chance', 0.45, '45% at level 20')
ON CONFLICT (job_id, min_level, bonus_type) DO UPDATE SET
    bonus_value = EXCLUDED.bonus_value,
    description = EXCLUDED.description;

-- Seed gambler bonuses
INSERT INTO job_level_bonuses (job_id, min_level, bonus_type, bonus_value, description) VALUES
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 1, 'prize_increase', 0.01, '1% prize increase per level'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 10, 'prize_increase', 0.10, '10% at level 10'),
    ((SELECT id FROM jobs WHERE job_key = 'gambler'), 25, 'prize_increase', 0.25, '25% max at level 25')
ON CONFLICT (job_id, min_level, bonus_type) DO UPDATE SET
    bonus_value = EXCLUDED.bonus_value,
    description = EXCLUDED.description;

-- +goose Down
-- Remove seeded data
DELETE FROM job_level_bonuses WHERE job_id IN (SELECT id FROM jobs WHERE job_key IN ('explorer', 'gambler'));
DELETE FROM jobs WHERE job_key IN ('blacksmith', 'explorer', 'merchant', 'gambler', 'farmer', 'scholar');
