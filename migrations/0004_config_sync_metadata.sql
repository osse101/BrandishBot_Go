-- +goose Up
-- Create config_sync_metadata table for tracking JSON config file sync state
-- Used by both progression tree and items config loaders
CREATE TABLE IF NOT EXISTS config_sync_metadata (
    config_name VARCHAR(100) PRIMARY KEY,
    last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    file_mod_time TIMESTAMP WITH TIME ZONE NOT NULL,
    sync_details JSONB DEFAULT '{}'::jsonb
);

-- Create index for lookups
CREATE INDEX idx_config_sync_metadata_config_name ON config_sync_metadata(config_name);

-- +goose Down
DROP INDEX IF EXISTS idx_config_sync_metadata_config_name;
DROP TABLE IF EXISTS config_sync_metadata;
