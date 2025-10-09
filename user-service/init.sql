-- ========================================
-- User Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Users Table
-- ========================================
CREATE TABLE IF NOT EXISTS user_service_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    followers_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    posts_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT user_service_username_not_empty CHECK (username <> ''),
    CONSTRAINT user_service_email_not_empty CHECK (email <> ''),
    CONSTRAINT user_service_followers_count_positive CHECK (followers_count >= 0),
    CONSTRAINT user_service_following_count_positive CHECK (following_count >= 0),
    CONSTRAINT user_service_posts_count_positive CHECK (posts_count >= 0)
);

-- ========================================
-- Indexes
-- ========================================
CREATE INDEX IF NOT EXISTS idx_user_service_users_username ON user_service_users(username);
CREATE INDEX IF NOT EXISTS idx_user_service_users_email ON user_service_users(email);
CREATE INDEX IF NOT EXISTS idx_user_service_users_created_at ON user_service_users(created_at DESC);

-- ========================================
-- Trigger to update updated_at timestamp
-- ========================================
CREATE OR REPLACE FUNCTION user_service_update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_user_service_users_updated_at
    BEFORE UPDATE ON user_service_users
    FOR EACH ROW
    EXECUTE FUNCTION user_service_update_updated_at_column();

-- ========================================
-- Follows Table
-- ========================================
CREATE TABLE IF NOT EXISTS user_service_follows (
    follower_id UUID NOT NULL REFERENCES user_service_users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES user_service_users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CONSTRAINT user_service_no_self_follow CHECK (follower_id <> following_id)
);

-- ========================================
-- Indexes for Follows Table
-- ========================================
CREATE INDEX IF NOT EXISTS idx_user_service_follows_follower_id 
ON user_service_follows(follower_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_service_follows_following_id 
ON user_service_follows(following_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_service_follows_relationship 
ON user_service_follows(follower_id, following_id);
