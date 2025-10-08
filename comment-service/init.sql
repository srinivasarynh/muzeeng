-- Comment Service PostgreSQL Schema

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better query performance
CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

-- Composite index for pagination queries
CREATE INDEX idx_comments_post_created ON comments(post_id, created_at DESC);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at on row updates
CREATE TRIGGER update_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Optional: Add foreign key constraints if you have posts and users tables
-- Uncomment these if you want referential integrity

-- ALTER TABLE comments
--     ADD CONSTRAINT fk_comments_post
--     FOREIGN KEY (post_id)
--     REFERENCES posts(id)
--     ON DELETE CASCADE;

-- ALTER TABLE comments
--     ADD CONSTRAINT fk_comments_user
--     FOREIGN KEY (user_id)
--     REFERENCES users(id)
--     ON DELETE CASCADE;

-- Optional: Add check constraint for content length
ALTER TABLE comments
    ADD CONSTRAINT check_content_not_empty
    CHECK (length(trim(content)) > 0);