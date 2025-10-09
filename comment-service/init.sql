-- ========================================
-- Comment Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Comments Table
-- ========================================
CREATE TABLE IF NOT EXISTS comment_service_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_content_not_empty CHECK (length(trim(content)) > 0)
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX IF NOT EXISTS idx_comment_service_comments_post_id ON comment_service_comments(post_id);
CREATE INDEX IF NOT EXISTS idx_comment_service_comments_user_id ON comment_service_comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comment_service_comments_created_at ON comment_service_comments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comment_service_comments_post_created ON comment_service_comments(post_id, created_at DESC);

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
CREATE TRIGGER trigger_update_comments_updated_at
BEFORE UPDATE ON comment_service_comments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
