-- +goose Up
INSERT INTO platforms (name) VALUES ('twitch'), ('youtube'), ('discord') ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM platforms WHERE name IN ('twitch', 'youtube', 'discord');
