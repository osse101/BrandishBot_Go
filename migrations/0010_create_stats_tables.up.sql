-- Create stats_events table for tracking all user activities
CREATE TABLE IF NOT EXISTS stats_events (
    event_id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_stats_events_user_id ON stats_events(user_id);
CREATE INDEX IF NOT EXISTS idx_stats_events_event_type ON stats_events(event_type);
CREATE INDEX IF NOT EXISTS idx_stats_events_created_at ON stats_events(created_at);
CREATE INDEX IF NOT EXISTS idx_stats_events_user_type ON stats_events(user_id, event_type);

-- Create stats_aggregates table for pre-calculated statistics
CREATE TABLE IF NOT EXISTS stats_aggregates (
    aggregate_id SERIAL PRIMARY KEY,
    period VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'monthly'
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    metrics JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(period, period_start)
);

-- Create index for efficient aggregate lookups
CREATE INDEX IF NOT EXISTS idx_stats_aggregates_period ON stats_aggregates(period, period_start);
