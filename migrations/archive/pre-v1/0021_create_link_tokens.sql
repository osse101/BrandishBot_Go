-- +goose Up
-- Create link_tokens table for cross-platform account linking

CREATE TABLE link_tokens (
    token VARCHAR(8) PRIMARY KEY,
    source_platform VARCHAR(20) NOT NULL,
    source_platform_id VARCHAR(100) NOT NULL,
    target_platform VARCHAR(20),
    target_platform_id VARCHAR(100),
    state VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Index for finding tokens by source
CREATE INDEX idx_link_tokens_source ON link_tokens(source_platform, source_platform_id);

-- Index for cleanup of expired tokens
CREATE INDEX idx_link_tokens_expires ON link_tokens(expires_at);

-- Index for state queries
CREATE INDEX idx_link_tokens_state ON link_tokens(state);

-- +goose Down
DROP TABLE IF EXISTS link_tokens;
