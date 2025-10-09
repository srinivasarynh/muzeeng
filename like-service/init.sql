-- ========================================
-- Like Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Likes Table
-- ========================================
CREATE TABLE IF NOT EXISTS like_service_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_post_like UNIQUE (post_id, user_id)
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX IF NOT EXISTS idx_like_service_likes_post_id ON like_service_likes(post_id);
CREATE INDEX IF NOT EXISTS idx_like_service_likes_user_id ON like_service_likes(user_id);
CREATE INDEX IF NOT EXISTS idx_like_service_likes_created_at ON like_service_likes(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_like_service_likes_post_user ON like_service_likes(post_id, user_id);

-- ========================================
-- Functions
-- ========================================

-- Get total likes for a post
CREATE OR REPLACE FUNCTION like_service_get_like_count(p_post_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM like_service_likes WHERE post_id = p_post_id);
END;
$$ LANGUAGE plpgsql;

-- Check if a specific user liked a post
CREATE OR REPLACE FUNCTION like_service_has_user_liked_post(p_post_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM like_service_likes 
        WHERE post_id = p_post_id 
        AND user_id = p_user_id
    );
END;
$$ LANGUAGE plpgsql;

-- Get recent liker IDs for a post (limit N)
CREATE OR REPLACE FUNCTION like_service_get_recent_liker_ids(p_post_id UUID, p_limit INTEGER DEFAULT 10)
RETURNS UUID[] AS $$
BEGIN
    RETURN ARRAY(
        SELECT user_id
        FROM like_service_likes
        WHERE post_id = p_post_id
        ORDER BY created_at DESC
        LIMIT p_limit
    );
END;
$$ LANGUAGE plpgsql;
