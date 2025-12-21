-- +goose Up
-- Migration: Progression unlock system with voting sessions and contribution tracking
-- Focused on NODE UNLOCKING through community contributions

-- Voting sessions (just handles the vote, ends when voting closes)
CREATE TABLE IF NOT EXISTS progression_voting_sessions (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,                           -- When voting closed
    voting_deadline TIMESTAMP NOT NULL DEFAULT NOW() + INTERVAL '24 hours', -- Auto-close after 24h
    winning_option_id INTEGER,                    -- Set when voting ends
    status VARCHAR(20) NOT NULL DEFAULT 'voting', -- 'voting', 'completed'
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Voting options (4 options per session)
CREATE TABLE IF NOT EXISTS progression_voting_options (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES progression_voting_sessions(id) ON DELETE CASCADE,
    node_id INTEGER NOT NULL REFERENCES progression_nodes(id),
    target_level INTEGER NOT NULL DEFAULT 1,
    vote_count INTEGER NOT NULL DEFAULT 0,
    last_highest_vote_at TIMESTAMP,               -- When this option first reached current highest
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, node_id, target_level)
);

-- Unlock progress tracking (accumulates contribution points toward next unlock)
-- Starts immediately after previous unlock, before vote ends
CREATE TABLE IF NOT EXISTS progression_unlock_progress (
    id SERIAL PRIMARY KEY,
    node_id INTEGER REFERENCES progression_nodes(id),     -- NULL until vote ends
    target_level INTEGER,                                  -- NULL until vote ends
    contributions_accumulated INTEGER NOT NULL DEFAULT 0, -- Points accumulated
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    unlocked_at TIMESTAMP,                                 -- When unlock completed
    voting_session_id INTEGER REFERENCES progression_voting_sessions(id)
);

-- Update user_votes to support sessions
ALTER TABLE user_votes 
    ADD COLUMN IF NOT EXISTS session_id INTEGER REFERENCES progression_voting_sessions(id),
    ADD COLUMN IF NOT EXISTS option_id INTEGER REFERENCES progression_voting_options(id);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_voting_sessions_status ON progression_voting_sessions(status);
CREATE INDEX IF NOT EXISTS idx_voting_sessions_active ON progression_voting_sessions(status) WHERE status = 'voting';
CREATE INDEX IF NOT EXISTS idx_voting_options_session ON progression_voting_options(session_id);
CREATE INDEX IF NOT EXISTS idx_unlock_progress_active ON progression_unlock_progress(unlocked_at) WHERE unlocked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_votes_session ON user_votes(session_id) WHERE session_id IS NOT NULL;

-- Add foreign key constraint for winning_option_id
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_winning_option'
    ) THEN
        ALTER TABLE progression_voting_sessions
            ADD CONSTRAINT fk_winning_option 
            FOREIGN KEY (winning_option_id) 
            REFERENCES progression_voting_options(id);
    END IF;
END $$;

-- +goose Down
-- Note: Down migration not implemented as this would require data migration
-- To rollback, manually drop tables in reverse order
