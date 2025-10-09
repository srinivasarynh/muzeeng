-- ========================================
-- Post Service Database Initialization
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Posts Table
-- ========================================
CREATE TABLE post_service_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0,
    
    -- Constraints
    CONSTRAINT posts_likes_count_non_negative CHECK (likes_count >= 0),
    CONSTRAINT posts_comments_count_non_negative CHECK (comments_count >= 0)
);

-- ========================================
-- Likes Table
-- ========================================
CREATE TABLE post_service_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES post_service_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Ensure a user can only like a post once
    CONSTRAINT post_likes_unique_user_post UNIQUE (post_id, user_id)
);

-- ========================================
-- Comments Table
-- ========================================
CREATE TABLE post_service_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES post_service_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX idx_posts_user_id ON post_service_posts(user_id);
CREATE INDEX idx_posts_created_at ON post_service_posts(created_at DESC);
CREATE INDEX idx_posts_updated_at ON post_service_posts(updated_at DESC);
CREATE INDEX idx_post_likes_post_id ON post_service_likes(post_id);
CREATE INDEX idx_post_likes_user_id ON post_service_likes(user_id);
CREATE INDEX idx_post_comments_post_id ON post_service_comments(post_id);
CREATE INDEX idx_post_comments_user_id ON post_service_comments(user_id);

-- Composite index for cursor-based pagination
CREATE INDEX idx_posts_created_at_id ON post_service_posts(created_at DESC, id);

-- ========================================
-- Triggers and Functions
-- ========================================

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at on posts
CREATE TRIGGER trigger_update_posts_updated_at
    BEFORE UPDATE ON post_service_posts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to auto-update updated_at on comments
CREATE TRIGGER trigger_update_post_comments_updated_at
    BEFORE UPDATE ON post_service_comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to increment likes_count
CREATE OR REPLACE FUNCTION increment_post_likes_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE post_service_posts SET likes_count = likes_count + 1 WHERE id = NEW.post_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to decrement likes_count
CREATE OR REPLACE FUNCTION decrement_post_likes_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE post_service_posts SET likes_count = likes_count - 1 WHERE id = OLD.post_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic likes_count management
CREATE TRIGGER trigger_increment_likes_count
    AFTER INSERT ON post_service_likes
    FOR EACH ROW
    EXECUTE FUNCTION increment_post_likes_count();

CREATE TRIGGER trigger_decrement_likes_count
    AFTER DELETE ON post_service_likes
    FOR EACH ROW
    EXECUTE FUNCTION decrement_post_likes_count();

-- Function to increment comments_count
CREATE OR REPLACE FUNCTION increment_post_comments_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE post_service_posts SET comments_count = comments_count + 1 WHERE id = NEW.post_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to decrement comments_count
CREATE OR REPLACE FUNCTION decrement_post_comments_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE post_service_posts SET comments_count = comments_count - 1 WHERE id = OLD.post_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic comments_count management
CREATE TRIGGER trigger_increment_comments_count
    AFTER INSERT ON post_service_comments
    FOR EACH ROW
    EXECUTE FUNCTION increment_post_comments_count();

CREATE TRIGGER trigger_decrement_comments_count
    AFTER DELETE ON post_service_comments
    FOR EACH ROW
    EXECUTE FUNCTION decrement_post_comments_count();

-- ========================================
-- Example Queries (commented out)
-- ========================================

-- Get posts with like status for a specific user
-- SELECT 
--     p.*,
--     CASE WHEN pl.user_id IS NOT NULL THEN true ELSE false END as is_liked
-- FROM post_service_posts p
-- LEFT JOIN post_service_likes pl ON p.id = pl.post_id AND pl.user_id = $1
-- ORDER BY p.created_at DESC;

-- Get posts with pagination
-- SELECT * FROM post_service_posts
-- WHERE created_at < $1  -- cursor timestamp
-- ORDER BY created_at DESC, id DESC
-- LIMIT $2;  -- page size