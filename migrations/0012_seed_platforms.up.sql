INSERT INTO platforms (name) VALUES ('twitch'), ('youtube'), ('discord') ON CONFLICT (name) DO NOTHING;
