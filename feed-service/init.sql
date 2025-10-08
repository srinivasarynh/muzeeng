-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0
);

-- Feed cache table
CREATE TABLE feed_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Feed stats table
CREATE TABLE feed_stats (
    user_id UUID PRIMARY KEY,
    total_posts INTEGER NOT NULL DEFAULT 0,
    unread_posts INTEGER NOT NULL DEFAULT 0,
    last_refreshed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    following_count INTEGER NOT NULL DEFAULT 0
);

-- Indexes for better query performance
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_feed_cache_user_id ON feed_cache(user_id);
CREATE INDEX idx_feed_cache_post_id ON feed_cache(post_id);
CREATE INDEX idx_feed_cache_user_post ON feed_cache(user_id, post_id);
CREATE INDEX idx_feed_cache_created_at ON feed_cache(created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at on posts
CREATE TRIGGER update_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Optional: Add comments for documentation
COMMENT ON TABLE posts IS 'Stores user posts in the feed';
COMMENT ON TABLE feed_cache IS 'Cached feed items for faster feed retrieval';
COMMENT ON TABLE feed_stats IS 'Statistics about user feeds';
COMMENT ON COLUMN posts.likes_count IS 'Denormalized count of likes for performance';
COMMENT ON COLUMN posts.comments_count IS 'Denormalized count of comments for performance';