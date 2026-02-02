-- +goose Up
-- +goose StatementBegin
CREATE TABLE expeditions (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    initiator_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    expedition_type varchar(50) NOT NULL,
    state text NOT NULL CHECK (state IN ('Created', 'Recruiting', 'InProgress', 'Completed')),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    join_deadline timestamp with time zone NOT NULL,
    completion_deadline timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    metadata jsonb
);

CREATE INDEX idx_expeditions_initiator ON expeditions(initiator_id);
CREATE INDEX idx_expeditions_state ON expeditions(state);
CREATE INDEX idx_expeditions_completion ON expeditions(completion_deadline) WHERE state = 'InProgress';

CREATE TABLE expedition_participants (
    expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    joined_at timestamp with time zone DEFAULT now() NOT NULL,
    rewards jsonb,
    PRIMARY KEY (expedition_id, user_id)
);

CREATE INDEX idx_expedition_participants_user ON expedition_participants(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS expedition_participants;
DROP TABLE IF EXISTS expeditions;
-- +goose StatementEnd
