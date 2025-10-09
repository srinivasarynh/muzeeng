-- ========================================
-- Feed Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Posts Table
-- ========================================
CREATE TABLE IF NOT EXISTS feed_service_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0
);

-- ========================================
-- Feed Cache Table
-- ========================================
CREATE TABLE IF NOT EXISTS feed_service_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES feed_service_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================
-- Feed Stats Table
-- ========================================
CREATE TABLE IF NOT EXISTS feed_service_stats (
    user_id UUID PRIMARY KEY,
    total_posts INTEGER NOT NULL DEFAULT 0,
    unread_posts INTEGER NOT NULL DEFAULT 0,
    last_refreshed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    following_count INTEGER NOT NULL DEFAULT 0
);

-- ========================================
-- Feed Likes Table
-- ========================================
CREATE TABLE IF NOT EXISTS feed_service_likes (
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES feed_service_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (user_id, post_id)
);

-- ========================================
-- Feed Follows Table
-- ========================================
CREATE TABLE IF NOT EXISTS feed_service_follows (
    follower_id UUID NOT NULL,
    followed_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (follower_id, followed_id)
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX IF NOT EXISTS idx_feed_service_posts_user_id ON feed_service_posts(user_id);
CREATE INDEX IF NOT EXISTS idx_feed_service_posts_created_at ON feed_service_posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feed_service_cache_user_id ON feed_service_cache(user_id);
CREATE INDEX IF NOT EXISTS idx_feed_service_cache_post_id ON feed_service_cache(post_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_feed_service_cache_user_post ON feed_service_cache(user_id, post_id);
CREATE INDEX IF NOT EXISTS idx_feed_service_cache_created_at ON feed_service_cache(created_at DESC);

-- ========================================
-- Function: Update 'updated_at' Column
-- ========================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- Trigger: Automatically Update 'updated_at'
-- ========================================
CREATE TRIGGER trigger_update_feed_posts_updated_at
BEFORE UPDATE ON feed_service_posts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- ========================================
-- Table Comments (Optional Documentation)
-- ========================================
COMMENT ON TABLE feed_service_posts IS 'Stores user posts in the feed';
COMMENT ON TABLE feed_service_cache IS 'Cached feed items for faster feed retrieval';
COMMENT ON TABLE feed_service_stats IS 'Statistics about user feeds';
COMMENT ON COLUMN feed_service_posts.likes_count IS 'Denormalized count of likes for performance';
COMMENT ON COLUMN feed_service_posts.comments_count IS 'Denormalized count of comments for performance';
