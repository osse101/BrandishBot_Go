-- +goose Up
-- Rename job keys to match the progression system (adding 'job_' prefix)
UPDATE jobs SET job_key = 'job_' || job_key WHERE job_key NOT LIKE 'job_%';

-- +goose Down
-- Remove the 'job_' prefix if it exists
UPDATE jobs SET job_key = REPLACE(job_key, 'job_', '') WHERE job_key LIKE 'job_%';
