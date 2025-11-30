-- +goose Up
-- Create events table for audit logging
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(100),
    payload JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_user ON events(user_id);
CREATE INDEX idx_events_created ON events(created_at DESC);
CREATE INDEX idx_events_payload ON events USING GIN(payload);

COMMENT ON TABLE events IS 'Event audit log with 10-day retention policy';
COMMENT ON COLUMN events.event_type IS 'Type of event (e.g., item.sold, item.bought)';
COMMENT ON COLUMN events.user_id IS 'User who triggered the event';
COMMENT ON COLUMN events.payload IS 'Event payload as JSON';
COMMENT ON COLUMN events.created_at IS 'Timestamp when event was logged';

-- +goose Down
-- Drop events table
DROP TABLE IF EXISTS events;
