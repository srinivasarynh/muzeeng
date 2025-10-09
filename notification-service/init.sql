-- ========================================
-- Notification Service Schema (Standalone)
-- ========================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Notification Type ENUM
-- ========================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_type') THEN
        CREATE TYPE notification_type AS ENUM ('LIKE', 'COMMENT', 'FOLLOW');
    END IF;
END
$$;

-- ========================================
-- Notifications Table
-- ========================================
CREATE TABLE IF NOT EXISTS notification_service_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type notification_type NOT NULL,
    message TEXT NOT NULL,
    actor_id UUID,
    related_id UUID,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- Indexes for Performance
-- ========================================
CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_user_id 
ON notification_service_notifications(user_id);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_user_created_at 
ON notification_service_notifications(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_user_is_read 
ON notification_service_notifications(user_id, is_read);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_actor_id 
ON notification_service_notifications(actor_id);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_related_id 
ON notification_service_notifications(related_id);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_created_at 
ON notification_service_notifications(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_service_notifications_user_pagination 
ON notification_service_notifications(user_id, is_read, created_at DESC, id);

-- ========================================
-- Comments for Documentation
-- ========================================
COMMENT ON TABLE notification_service_notifications IS 'Stores user notifications for likes, comments, and follows';
COMMENT ON COLUMN notification_service_notifications.user_id IS 'The user who receives the notification';
COMMENT ON COLUMN notification_service_notifications.actor_id IS 'The user who triggered the notification (optional)';
COMMENT ON COLUMN notification_service_notifications.related_id IS 'Reference to related entity (post, comment, etc.)';
COMMENT ON COLUMN notification_service_notifications.is_read IS 'Whether the notification has been read by the user';

-- ========================================
-- Functions
-- ========================================

-- Get unread notification count for a user
CREATE OR REPLACE FUNCTION notification_service_get_unread_notification_count(p_user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM notification_service_notifications WHERE user_id = p_user_id AND is_read = FALSE);
END;
$$ LANGUAGE plpgsql STABLE;

-- Mark notifications as read for a user
CREATE OR REPLACE FUNCTION notification_service_mark_notifications_as_read(p_user_id UUID, p_notification_ids UUID[])
RETURNS INTEGER AS $$
DECLARE
    updated_count INTEGER;
BEGIN
    UPDATE notification_service_notifications
    SET is_read = TRUE
    WHERE user_id = p_user_id
    AND id = ANY(p_notification_ids)
    AND is_read = FALSE;

    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;
