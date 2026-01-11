-- +goose Up
-- Create progression system tables


-- Progression tree nodes (defines what CAN be unlocked)
CREATE TABLE progression_nodes (
    id SERIAL PRIMARY KEY,
    node_key VARCHAR(100) NOT NULL,
    node_type VARCHAR(50) NOT NULL,             -- 'feature', 'item', 'mechanic', 'upgrade'
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    parent_node_id INTEGER REFERENCES progression_nodes(id) ON DELETE CASCADE,
    max_level INTEGER DEFAULT 1,                -- For incremental upgrades (e.g., 5 levels)
    unlock_cost INTEGER DEFAULT 1000,           -- Engagement score needed per level
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(node_key, parent_node_id)            -- Allow same key under different parents
);

-- Global unlocks (what HAS been unlocked)
CREATE TABLE progression_unlocks (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES progression_nodes(id) ON DELETE CASCADE,
    current_level INTEGER DEFAULT 1,            -- Current unlock level (1 to max_level)
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    unlocked_by VARCHAR(50),                    -- 'vote', 'admin', 'auto', 'instant_override'
    engagement_score INTEGER DEFAULT 0,         -- Score when this level unlocked
    UNIQUE(node_id, current_level)              -- Track each level unlock separately
);

-- Current voting state
CREATE TABLE progression_voting (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES progression_nodes(id) ON DELETE CASCADE,
    target_level INTEGER DEFAULT 1,             -- Which level is being voted for
    vote_count INTEGER DEFAULT 0,
    voting_started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    voting_ends_at TIMESTAMP,                   -- 24 hours from start (or admin override)
    is_active BOOLEAN DEFAULT true,
    UNIQUE(node_id, target_level)
);

-- User votes (prevent double voting per node/level)
CREATE TABLE user_votes (
    user_id VARCHAR(255) NOT NULL,
    node_id INTEGER REFERENCES progression_nodes(id) ON DELETE CASCADE,
    target_level INTEGER DEFAULT 1,
    voted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, node_id, target_level)
);

-- User-level progression (recipes unlocked only - class/XP separate)
CREATE TABLE user_progression (
    user_id VARCHAR(255) NOT NULL,
    progression_type VARCHAR(50) NOT NULL,      -- 'recipe' only for now
    progression_key VARCHAR(100) NOT NULL,      -- e.g., 'recipe_lootbox1'
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,                             -- Flexible data
    PRIMARY KEY (user_id, progression_type, progression_key)
);

-- Engagement tracking per user (integrates with Stats)
CREATE TABLE engagement_metrics (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,              -- Always track per user
    metric_type VARCHAR(50) NOT NULL,           -- 'message', 'command', 'item_crafted', 'item_used'
    metric_value INTEGER DEFAULT 1,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB                              -- Additional context
);

-- Engagement weights configuration
CREATE TABLE engagement_weights (
    metric_type VARCHAR(50) PRIMARY KEY,
    weight DECIMAL(5,2) DEFAULT 1.0,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Progression reset history (for annual resets)
CREATE TABLE progression_resets (
    id SERIAL PRIMARY KEY,
    reset_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reset_by VARCHAR(255),                      -- Admin who triggered reset
    reason TEXT,
    nodes_reset_count INTEGER,
    engagement_score_at_reset INTEGER
);

-- Indexes for performance
CREATE INDEX idx_progression_nodes_parent ON progression_nodes(parent_node_id);
CREATE INDEX idx_progression_unlocks_node ON progression_unlocks(node_id);
CREATE INDEX idx_user_progression ON user_progression(user_id, progression_type);
CREATE INDEX idx_engagement_metrics_user ON engagement_metrics(user_id, metric_type);
CREATE INDEX idx_engagement_metrics_type_time ON engagement_metrics(metric_type, recorded_at);
CREATE INDEX idx_voting_active ON progression_voting(is_active, voting_ends_at);

-- Insert default engagement weights
INSERT INTO engagement_weights (metric_type, weight, description) VALUES
('message', 1.0, 'Messages sent in chat'),
('command', 2.0, 'Commands executed'),
('item_crafted', 3.0, 'Items crafted/upgraded'),
('item_used', 1.5, 'Items consumed/used');

-- +goose Down
DROP TABLE IF EXISTS progression_resets;
DROP TABLE IF EXISTS engagement_weights;
DROP TABLE IF EXISTS engagement_metrics;
DROP TABLE IF EXISTS user_progression;
DROP TABLE IF EXISTS user_votes;
DROP TABLE IF EXISTS progression_voting;
DROP TABLE IF EXISTS progression_unlocks;
DROP TABLE IF EXISTS progression_nodes;
