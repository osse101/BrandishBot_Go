-- +goose Up
DROP TABLE IF EXISTS job_xp_events;

-- +goose Down
CREATE TABLE job_xp_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    job_id INTEGER NOT NULL REFERENCES jobs(id),
    xp_amount INTEGER NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    source_metadata JSONB,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_xp_user_id ON job_xp_events(user_id);
CREATE INDEX idx_job_xp_recorded_at ON job_xp_events(recorded_at);
