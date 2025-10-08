-- Create extension for UUID support if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Follows table - stores the follower/following relationships
CREATE TABLE follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_id UUID NOT NULL,  -- User who is following
    following_id UUID NOT NULL, -- User being followed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Ensure a user cannot follow the same person twice
    CONSTRAINT unique_follow UNIQUE (follower_id, following_id),
    
    -- Ensure a user cannot follow themselves
    CONSTRAINT no_self_follow CHECK (follower_id != following_id)
);

-- Index for finding all users that a specific user is following
CREATE INDEX idx_follows_follower_id ON follows(follower_id, created_at DESC);

-- Index for finding all followers of a specific user
CREATE INDEX idx_follows_following_id ON follows(following_id, created_at DESC);

-- Composite index for quick lookup of follow relationships
CREATE INDEX idx_follows_relationship ON follows(follower_id, following_id);

-- Index for cursor-based pagination queries
CREATE INDEX idx_follows_created_at ON follows(created_at DESC, id);

-- Optional: Add foreign key constraints if you have a users table
-- ALTER TABLE follows 
--     ADD CONSTRAINT fk_follows_follower 
--     FOREIGN KEY (follower_id) 
--     REFERENCES users(id) 
--     ON DELETE CASCADE;

-- ALTER TABLE follows 
--     ADD CONSTRAINT fk_follows_following 
--     FOREIGN KEY (following_id) 
--     REFERENCES users(id) 
--     ON DELETE CASCADE;

-- Function to get follower count for a user
CREATE OR REPLACE FUNCTION get_followers_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follows WHERE following_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get following count for a user
CREATE OR REPLACE FUNCTION get_following_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follows WHERE follower_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if user A follows user B
CREATE OR REPLACE FUNCTION is_following(follower UUID, following UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM follows 
        WHERE follower_id = follower 
        AND following_id = following
    );
END;
$$ LANGUAGE plpgsql STABLE;