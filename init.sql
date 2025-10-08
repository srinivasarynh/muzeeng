-- ========================================
-- Create Databases for Each Service
-- ========================================
CREATE DATABASE auth_service_db;
CREATE DATABASE user_service_db;
CREATE DATABASE post_service_db;
CREATE DATABASE comment_service_db;
CREATE DATABASE like_service_db;
CREATE DATABASE follow_service_db;
CREATE DATABASE feed_service_db;
CREATE DATABASE notification_service_db;

-- ========================================
-- Connect to auth_service_db
-- ========================================
\c auth_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'role_type') THEN
        CREATE TYPE role_type AS ENUM ('USER', 'ADMIN');
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS auth_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    followers_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    posts_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT check_followers_count CHECK (followers_count >= 0),
    CONSTRAINT check_following_count CHECK (following_count >= 0),
    CONSTRAINT check_posts_count CHECK (posts_count >= 0)
);

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    token VARCHAR(512) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS auth_token_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(512) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auth_user_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
    role role_type NOT NULL DEFAULT 'USER',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role)
);

-- ========================================
-- Connect to user_service_db
-- ========================================
\c user_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS user_service_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    followers_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    posts_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT username_not_empty CHECK (username <> ''),
    CONSTRAINT email_not_empty CHECK (email <> ''),
    CONSTRAINT followers_count_positive CHECK (followers_count >= 0),
    CONSTRAINT following_count_positive CHECK (following_count >= 0),
    CONSTRAINT posts_count_positive CHECK (posts_count >= 0)
);

CREATE TABLE IF NOT EXISTS user_service_follows (
    follower_id UUID NOT NULL REFERENCES user_service_users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES user_service_users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CONSTRAINT no_self_follow CHECK (follower_id <> following_id)
);

-- ========================================
-- Connect to post_service_db
-- ========================================
\c post_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS post_service_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT posts_likes_count_non_negative CHECK (likes_count >= 0),
    CONSTRAINT posts_comments_count_non_negative CHECK (comments_count >= 0)
);

CREATE TABLE IF NOT EXISTS post_service_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES post_service_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT post_likes_unique_user_post UNIQUE (post_id, user_id)
);

CREATE TABLE IF NOT EXISTS post_service_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES post_service_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- ========================================
-- Connect to comment_service_db
-- ========================================
\c comment_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
-- Connect to like_service_db
-- ========================================
\c like_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS like_service_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_post_like UNIQUE (post_id, user_id)
);

CREATE OR REPLACE FUNCTION like_service_get_like_count(p_post_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM like_service_likes WHERE post_id = p_post_id);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION like_service_has_user_liked_post(p_post_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(SELECT 1 FROM like_service_likes WHERE post_id = p_post_id AND user_id = p_user_id);
END;
$$ LANGUAGE plpgsql;

-- ========================================
-- Connect to follow_service_db
-- ========================================
\c follow_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS follow_service_follows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    follower_id UUID NOT NULL,
    following_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_follow UNIQUE (follower_id, following_id),
    CONSTRAINT no_self_follow CHECK (follower_id != following_id)
);

CREATE OR REPLACE FUNCTION follow_service_get_followers_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follow_service_follows WHERE following_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION follow_service_get_following_count(user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM follow_service_follows WHERE follower_id = user_id);
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION follow_service_is_following(follower UUID, following UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(
        SELECT 1 FROM follow_service_follows 
        WHERE follower_id = follower 
        AND following_id = following
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- ========================================
-- Connect to feed_service_db
-- ========================================
\c feed_service_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS feed_service_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS feed_service_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES feed_service_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS feed_service_stats (
    user_id UUID PRIMARY KEY,
    total_posts INTEGER NOT NULL DEFAULT 0,
    unread_posts INTEGER NOT NULL DEFAULT 0,
    last_refreshed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    following_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS feed_service_likes (
    user_id UUID NOT NULL,
    post_id UUID NOT NULL REFERENCES feed_service_posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (user_id, post_id)
);


CREATE TABLE IF NOT EXISTS feed_service_follows (
    follower_id UUID NOT NULL,
    followed_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (follower_id, followed_id)
);

CREATE UNIQUE INDEX idx_feed_cache_user_post 
ON feed_service_cache(user_id, post_id);


-- ========================================
-- Connect to notification_service_db
-- ========================================
\c notification_service_db

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notification_type') THEN
        CREATE TYPE notification_type AS ENUM ('LIKE','COMMENT','FOLLOW');
    END IF;
END
$$;
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

CREATE OR REPLACE FUNCTION notification_service_get_unread_notification_count(p_user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*)::INTEGER FROM notification_service_notifications WHERE user_id = p_user_id AND is_read = FALSE);
END;
$$ LANGUAGE plpgsql STABLE;

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

-- ========================================
-- Update Triggers (for all databases)
-- ========================================
\c auth_service_db
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_auth_users_updated_at
BEFORE UPDATE ON auth_users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\c user_service_db
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_users_updated_at
BEFORE UPDATE ON user_service_users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\c post_service_db
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_posts_updated_at
BEFORE UPDATE ON post_service_posts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_post_comments_updated_at
BEFORE UPDATE ON post_service_comments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\c comment_service_db
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_comments_updated_at
BEFORE UPDATE ON comment_service_comments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\c feed_service_db
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_feed_posts_updated_at
BEFORE UPDATE ON feed_service_posts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();