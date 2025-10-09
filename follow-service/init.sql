-- ========================================
-- Follow Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Follows Table
-- ========================================
CREATE TABLE IF NOT EXISTS follow_service_follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_id UUID NOT NULL,
    following_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_follow UNIQUE (follower_id, following_id),
    CONSTRAINT no_self_follow CHECK (follower_id != following_id)
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX IF NOT EXISTS idx_follow_service_follows_follower_id 
ON follow_service_follows(follower_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_follow_service_follows_following_id 
ON follow_service_follows(following_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_follow_service_follows_relationship 
ON follow_service_follows(follower_id, following_id);

CREATE INDEX IF NOT EXISTS idx_follow_service_follows_created_at 
ON follow_service_follows(created_at DESC, id);

-- ========================================
-- Functions
-- ========================================

-- Get number of followers for a user
CREATE OR REPLACE FUNCTION follow_service_get_followers_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follow_service_follows WHERE following_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

-- Get number of users a user is following
CREATE OR REPLACE FUNCTION follow_service_get_following_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follow_service_follows WHERE follower_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

-- Check if a user is following another user
CREATE OR REPLACE FUNCTION follow_service_is_following(follower UUID, following UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM follow_service_follows
        WHERE follower_id = follower
        AND following_id = following
    );
END;
$$ LANGUAGE plpgsql STABLE;
