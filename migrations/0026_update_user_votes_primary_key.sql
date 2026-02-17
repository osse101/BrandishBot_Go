-- +goose Up
-- Allow voting for the same node in different sessions
-- First, ensure session_id is not null. For existing records with null session_id,
-- we'll assume they belong to a 'legacy' session 0.
UPDATE public.user_votes SET session_id = 0 WHERE session_id IS NULL;
ALTER TABLE public.user_votes ALTER COLUMN session_id SET NOT NULL;

-- Now update the primary key to be (userID, sessionID)
-- This ensures a user can only vote once per session, but can vote for the same node in different sessions.
-- We DROP the old PK (user_id, node_id, target_level) which was causing the bug.
ALTER TABLE public.user_votes DROP CONSTRAINT IF EXISTS user_votes_pkey;
ALTER TABLE public.user_votes ADD CONSTRAINT user_votes_pkey PRIMARY KEY (user_id, session_id);

-- Also remove the too-restrictive unique constraint on the legacy progression_voting table
-- to prevent similar issues if it is ever used.
ALTER TABLE public.progression_voting DROP CONSTRAINT IF EXISTS progression_voting_node_id_target_level_key;

-- +goose Down
ALTER TABLE public.user_votes DROP CONSTRAINT IF EXISTS user_votes_pkey;
-- Note: Rolling back might fail if there are now duplicate (user_id, node_id, target_level) across different sessions.
-- This is intentional as the new data model is more permissive.
ALTER TABLE public.user_votes ADD CONSTRAINT user_votes_pkey PRIMARY KEY (user_id, node_id, target_level);
ALTER TABLE public.user_votes ALTER COLUMN session_id DROP NOT NULL;

ALTER TABLE public.progression_voting ADD CONSTRAINT progression_voting_node_id_target_level_key UNIQUE (node_id, target_level);
