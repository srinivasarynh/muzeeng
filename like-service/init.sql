-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Likes table
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Ensure a user can only like a post once
    CONSTRAINT unique_user_post_like UNIQUE (post_id, user_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_likes_post_id ON likes(post_id);
CREATE INDEX idx_likes_user_id ON likes(user_id);
CREATE INDEX idx_likes_created_at ON likes(created_at DESC);

-- Composite index for checking if a specific user liked a specific post
CREATE INDEX idx_likes_post_user ON likes(post_id, user_id);

-- Optional: Add foreign key constraints if you have posts and users tables
-- Uncomment and modify these if needed:
-- ALTER TABLE likes 
--     ADD CONSTRAINT fk_likes_post 
--     FOREIGN KEY (post_id) 
--     REFERENCES posts(id) 
--     ON DELETE CASCADE;
--
-- ALTER TABLE likes 
--     ADD CONSTRAINT fk_likes_user 
--     FOREIGN KEY (user_id) 
--     REFERENCES users(id) 
--     ON DELETE CASCADE;

-- Function to get like count for a post
CREATE OR REPLACE FUNCTION get_like_count(p_post_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM likes WHERE post_id = p_post_id);
END;
$$ LANGUAGE plpgsql;

-- Function to check if user liked a post
CREATE OR REPLACE FUNCTION has_user_liked_post(p_post_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(SELECT 1 FROM likes WHERE post_id = p_post_id AND user_id = p_user_id);
END;
$$ LANGUAGE plpgsql;

-- Function to get recent liker IDs (last N users who liked a post)
CREATE OR REPLACE FUNCTION get_recent_liker_ids(p_post_id UUID, p_limit INTEGER DEFAULT 10)
RETURNS UUID[] AS $$
BEGIN
    RETURN ARRAY(
        SELECT user_id 
        FROM likes 
        WHERE post_id = p_post_id 
        ORDER BY created_at DESC 
        LIMIT p_limit
    );
END;
$$ LANGUAGE plpgsql;