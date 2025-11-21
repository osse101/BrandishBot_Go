-- Create platforms table
CREATE TABLE IF NOT EXISTS platforms (
    platform_id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create user_platform_links table
CREATE TABLE IF NOT EXISTS user_platform_links (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    platform_id INTEGER NOT NULL REFERENCES platforms(platform_id),
    platform_user_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (user_id, platform_id),
    UNIQUE (platform_id, platform_user_id)
);

-- Create index on platform_user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_platform_user_id ON user_platform_links(platform_user_id);
