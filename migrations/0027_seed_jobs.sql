-- +goose Up
-- Seed job definitions for environments using the v1 schema (which doesn't include seed data).
-- These match the archived migration 0019 + 0025 rename (job_ prefix).
INSERT INTO jobs (job_key, display_name, description, associated_features) VALUES
    ('job_blacksmith', 'Blacksmith', 'Masters of crafting, upgrades, and disassembly', ARRAY['upgrade', 'craft', 'disassemble']),
    ('job_explorer', 'Explorer', 'Scouts who find extra rewards', ARRAY['search']),
    ('job_merchant', 'Merchant', 'Traders who get better deals', ARRAY['buy', 'sell']),
    ('job_gambler', 'Gambler', 'High rollers who win bigger prizes', ARRAY['gamble']),
    ('job_farmer', 'Farmer', 'Patient cultivators of valuable crops', ARRAY['farm']),
    ('job_scholar', 'Scholar', 'Contributors to community progress', ARRAY['progression'])
ON CONFLICT (job_key) DO NOTHING;

-- +goose Down
DELETE FROM jobs WHERE job_key IN ('job_blacksmith', 'job_explorer', 'job_merchant', 'job_gambler', 'job_farmer', 'job_scholar');
