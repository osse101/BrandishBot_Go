-- +goose Up
-- +goose StatementBegin

-- Subscription tier reference table
CREATE TABLE subscription_tiers (
    tier_id SERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    tier_name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    tier_level INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_platform_tier UNIQUE (platform, tier_name)
);

CREATE INDEX idx_subscription_tiers_platform ON subscription_tiers(platform);

-- Seed subscription tiers
INSERT INTO subscription_tiers (platform, tier_name, display_name, tier_level) VALUES
    ('twitch', 'tier1', 'Tier 1 Subscriber', 1),
    ('twitch', 'tier2', 'Tier 2 Subscriber', 2),
    ('twitch', 'tier3', 'Tier 3 Subscriber', 3),
    ('youtube', 'member', 'YouTube Member', 1);

-- User subscription status table
CREATE TABLE user_subscriptions (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    tier_id INT NOT NULL REFERENCES subscription_tiers(tier_id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'expired', 'cancelled')),
    subscribed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, platform)
);

-- Index for expiration checks (used by background worker)
CREATE INDEX idx_user_subscriptions_expiring ON user_subscriptions(expires_at)
    WHERE status = 'active';

-- Indices for common queries
CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX idx_user_subscriptions_platform ON user_subscriptions(platform);

-- Subscription audit trail
CREATE TABLE subscription_history (
    history_id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    tier_id INT NOT NULL REFERENCES subscription_tiers(tier_id),
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('subscribed', 'renewed', 'upgraded', 'downgraded', 'cancelled', 'expired')),
    subscribed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indices for history queries
CREATE INDEX idx_subscription_history_user ON subscription_history(user_id, created_at DESC);
CREATE INDEX idx_subscription_history_platform ON subscription_history(platform);
CREATE INDEX idx_subscription_history_event_type ON subscription_history(event_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS subscription_history;
DROP TABLE IF EXISTS user_subscriptions;
DROP TABLE IF EXISTS subscription_tiers;
-- +goose StatementEnd
