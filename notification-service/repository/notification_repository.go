package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"notification-service/model"
)

const (
	// Cache TTLs
	unreadCountTTL       = 5 * time.Minute
	notificationTTL      = 15 * time.Minute
	userNotificationsTTL = 2 * time.Minute

	// Cache key prefixes
	unreadCountPrefix  = "notif:unread:"
	notificationPrefix = "notif:id:"
	userNotifsPrefix   = "notif:user:"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, first int, after *string) (*models.NotificationConnection, error)
	MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int32, error)
}

type notificationRepository struct {
	db    *sqlx.DB
	redis *redis.Client
}

func NewNotificationRepository(db *sqlx.DB, redisClient *redis.Client) NotificationRepository {
	return &notificationRepository{
		db:    db,
		redis: redisClient,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notification_service_notifications (id, user_id, type, message, actor_id, related_id, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Message,
		notification.ActorID,
		notification.RelatedID,
		notification.IsRead,
		notification.CreatedAt,
	)

	if err != nil {
		return err
	}

	r.invalidateUserCaches(ctx, notification.UserID)

	r.cacheNotification(ctx, notification)

	return nil
}

func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	cacheKey := notificationPrefix + id.String()
	cached, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var notification models.Notification
		if err := json.Unmarshal([]byte(cached), &notification); err == nil {
			return &notification, nil
		}
	}

	query := `
		SELECT id, user_id, type, message, actor_id, related_id, is_read, created_at
		FROM notification_service_notifications
		WHERE id = $1
	`

	var notification models.Notification
	err = r.db.GetContext(ctx, &notification, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, err
	}

	r.cacheNotification(ctx, &notification)

	return &notification, nil
}

func (r *notificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID, first int, after *string) (*models.NotificationConnection, error) {
	cacheKey := fmt.Sprintf("%s%s:first:%d", userNotifsPrefix, userID.String(), first)
	if after != nil && *after != "" {
		cacheKey += ":after:" + *after
	}

	if after == nil || *after == "" {
		cached, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var connection models.NotificationConnection
			if err := json.Unmarshal([]byte(cached), &connection); err == nil {
				return &connection, nil
			}
		}
	}

	var notifications []models.Notification
	var totalCount int32
	var args []interface{}
	argIndex := 1

	countQuery := `SELECT COUNT(*) FROM notification_service_notifications WHERE user_id = $1`
	err := r.db.GetContext(ctx, &totalCount, countQuery, userID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, type, message, actor_id, related_id, is_read, created_at
		FROM notification_service_notifications
		WHERE user_id = $` + fmt.Sprintf("%d", argIndex)
	args = append(args, userID)
	argIndex++

	if after != nil && *after != "" {
		cursorTime, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND created_at < $%d", argIndex)
		args = append(args, cursorTime)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIndex)
	args = append(args, first+1)

	err = r.db.SelectContext(ctx, &notifications, query, args...)
	if err != nil {
		return nil, err
	}

	unreadCount, err := r.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	hasNextPage := len(notifications) > first
	if hasNextPage {
		notifications = notifications[:first]
	}

	edges := make([]models.NotificationEdge, len(notifications))
	for i, notification := range notifications {
		cursor := encodeCursor(notification.CreatedAt)
		edges[i] = models.NotificationEdge{
			Cursor: cursor,
			Node:   notification,
		}
	}

	var endCursor, startCursor *string
	if len(edges) > 0 {
		endCursor = &edges[len(edges)-1].Cursor
		startCursor = &edges[0].Cursor
	}

	pageInfo := models.PageInfo{
		EndCursor:       endCursor,
		HasNextPage:     hasNextPage,
		StartCursor:     startCursor,
		HasPreviousPage: after != nil && *after != "",
	}

	connection := &models.NotificationConnection{
		Edges:       edges,
		PageInfo:    pageInfo,
		TotalCount:  totalCount,
		UnreadCount: unreadCount,
	}

	if after == nil || *after == "" {
		if data, err := json.Marshal(connection); err == nil {
			r.redis.Set(ctx, cacheKey, data, userNotificationsTTL)
		}
	}

	return connection, nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	query := `
		UPDATE notification_service_notifications
		SET is_read = true
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, notificationID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("notification not found or unauthorized")
	}

	r.invalidateUserCaches(ctx, userID)
	r.redis.Del(ctx, notificationPrefix+notificationID.String())

	return nil
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE notification_service_notifications
		SET is_read = true
		WHERE user_id = $1 AND is_read = false
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	r.invalidateUserCaches(ctx, userID)

	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	notification, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM notification_service_notifications WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("notification not found")
	}

	r.invalidateUserCaches(ctx, notification.UserID)
	r.redis.Del(ctx, notificationPrefix+id.String())

	return nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int32, error) {
	cacheKey := unreadCountPrefix + userID.String()
	cached, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var count int32
		if _, err := fmt.Sscanf(cached, "%d", &count); err == nil {
			return count, nil
		}
	}

	query := `SELECT COUNT(*) FROM notification_service_notifications WHERE user_id = $1 AND is_read = false`

	var count int32
	err = r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}

	r.redis.Set(ctx, cacheKey, fmt.Sprintf("%d", count), unreadCountTTL)

	return count, nil
}

// Helper functions for caching

func (r *notificationRepository) cacheNotification(ctx context.Context, notification *models.Notification) {
	cacheKey := notificationPrefix + notification.ID.String()
	if data, err := json.Marshal(notification); err == nil {
		r.redis.Set(ctx, cacheKey, data, notificationTTL)
	}
}

func (r *notificationRepository) invalidateUserCaches(ctx context.Context, userID uuid.UUID) {
	r.redis.Del(ctx, unreadCountPrefix+userID.String())

	pattern := userNotifsPrefix + userID.String() + ":*"
	iter := r.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		r.redis.Del(ctx, iter.Val())
	}
}

// Helper functions for cursor encoding/decoding
func encodeCursor(t time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(t.Format(time.RFC3339Nano)))
}

func decodeCursor(cursor string) (time.Time, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(decoded))
}
