-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM type for notification types
CREATE TYPE notification_type AS ENUM ('LIKE', 'COMMENT', 'FOLLOW');

-- Create notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type notification_type NOT NULL,
    message TEXT NOT NULL,
    actor_id UUID,
    related_id UUID,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for common query patterns
    CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT notifications_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for efficient querying
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_id_created_at ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_user_id_is_read ON notifications(user_id, is_read);
CREATE INDEX idx_notifications_actor_id ON notifications(actor_id);
CREATE INDEX idx_notifications_related_id ON notifications(related_id);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Composite index for cursor-based pagination with unread filtering
CREATE INDEX idx_notifications_user_pagination ON notifications(user_id, is_read, created_at DESC, id);

-- Add table comments for documentation
COMMENT ON TABLE notifications IS 'Stores user notifications for likes, comments, and follows';
COMMENT ON COLUMN notifications.user_id IS 'The user who receives the notification';
COMMENT ON COLUMN notifications.actor_id IS 'The user who triggered the notification (optional)';
COMMENT ON COLUMN notifications.related_id IS 'Reference to related entity (post, comment, etc.)';
COMMENT ON COLUMN notifications.is_read IS 'Whether the notification has been read by the user';

-- Optional: Create a function to get unread count
CREATE OR REPLACE FUNCTION get_unread_notification_count(p_user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM notifications WHERE user_id = p_user_id AND is_read = FALSE);
END;
$$ LANGUAGE plpgsql STABLE;

-- Optional: Create a function to mark notifications as read
CREATE OR REPLACE FUNCTION mark_notifications_as_read(p_user_id UUID, p_notification_ids UUID[])
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE notifications
    SET is_read = TRUE
    WHERE user_id = p_user_id
    AND id = ANY(p_notification_ids)
    AND is_read = FALSE;
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Optional: Create a trigger to automatically clean up old notifications
-- (keeps only last 90 days or last 1000 notifications per user)
CREATE OR REPLACE FUNCTION cleanup_old_notifications()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM notifications
    WHERE id IN (
        SELECT id FROM notifications
        WHERE user_id = NEW.user_id
        AND (
            created_at < CURRENT_TIMESTAMP - INTERVAL '90 days'
            OR id NOT IN (
                SELECT id FROM notifications
                WHERE user_id = NEW.user_id
                ORDER BY created_at DESC
                LIMIT 1000
            )
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Optional: Attach cleanup trigger (commented out by default)
-- CREATE TRIGGER trigger_cleanup_old_notifications
-- AFTER INSERT ON notifications
-- FOR EACH ROW
-- EXECUTE FUNCTION cleanup_old_notifications();