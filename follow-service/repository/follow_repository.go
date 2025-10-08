package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"follow-service/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type FollowRepository interface {
	FollowUser(ctx context.Context, followerID, followingID uuid.UUID) error
	UnfollowUser(ctx context.Context, followerID, followingID uuid.UUID) error
	GetFollowers(ctx context.Context, userID uuid.UUID, first int32, after *string) (*models.FollowConnection, error)
	GetFollowing(ctx context.Context, userID uuid.UUID, first int32, after *string) (*models.FollowConnection, error)
	IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
	GetFollowStatus(ctx context.Context, userID uuid.UUID, targetUserIDs []uuid.UUID) ([]models.FollowStatus, error)
	GetFollowersCounts(ctx context.Context, userIDs []uuid.UUID) ([]models.UserFollowCounts, error)
}

type followRepository struct {
	db *sqlx.DB
}

func NewFollowRepository(db *sqlx.DB) FollowRepository {
	return &followRepository{db: db}
}

// FollowUser creates a new follow relationship
func (r *followRepository) FollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	if followerID == followingID {
		return fmt.Errorf("users cannot follow themselves")
	}

	query := `
		INSERT INTO follow_service_follows (id, follower_id, following_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (follower_id, following_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), followerID, followingID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to follow user: %w", err)
	}

	return nil
}

// UnfollowUser removes a follow relationship
func (r *followRepository) UnfollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	query := `
		DELETE FROM follow_service_follows
		WHERE follower_id = $1 AND following_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, followerID, followingID)
	if err != nil {
		return fmt.Errorf("failed to unfollow user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("follow relationship not found")
	}

	return nil
}

// GetFollowers returns paginated list of followers
func (r *followRepository) GetFollowers(ctx context.Context, userID uuid.UUID, first int32, after *string) (*models.FollowConnection, error) {
	var startTime time.Time
	var startID uuid.UUID

	if after != nil && *after != "" {
		decoded, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		startTime = decoded.Timestamp
		startID = decoded.ID
	}

	query := `
		SELECT f.follower_id, f.created_at, f.id
		FROM follow_service_follows f
		WHERE f.following_id = $1
	`

	args := []interface{}{userID}
	argCount := 1

	if after != nil && *after != "" {
		argCount++
		query += fmt.Sprintf(" AND (f.created_at, f.id) < ($%d, $%d)", argCount, argCount+1)
		args = append(args, startTime, startID)
		argCount++
	}

	query += " ORDER BY f.created_at DESC, f.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argCount+1)
	args = append(args, first+1)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query followers: %w", err)
	}
	defer rows.Close()

	var edges []models.FollowEdge
	for rows.Next() {
		var followerID uuid.UUID
		var createdAt time.Time
		var id uuid.UUID

		if err := rows.Scan(&followerID, &createdAt, &id); err != nil {
			return nil, fmt.Errorf("failed to scan follower: %w", err)
		}

		cursor := encodeCursor(createdAt, id)
		edges = append(edges, models.FollowEdge{
			Cursor:     cursor,
			UserID:     followerID,
			FollowedAt: createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating followers: %w", err)
	}

	hasNextPage := len(edges) > int(first)
	if hasNextPage {
		edges = edges[:first]
	}

	var pageInfo models.PageInfo
	pageInfo.HasNextPage = hasNextPage

	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor
	}

	totalCount, err := r.getFollowersCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	return &models.FollowConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}

// GetFollowing returns paginated list of users being followed
func (r *followRepository) GetFollowing(ctx context.Context, userID uuid.UUID, first int32, after *string) (*models.FollowConnection, error) {
	var startTime time.Time
	var startID uuid.UUID

	if after != nil && *after != "" {
		decoded, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		startTime = decoded.Timestamp
		startID = decoded.ID
	}

	query := `
		SELECT f.following_id, f.created_at, f.id
		FROM follow_service_follows f
		WHERE f.follower_id = $1
	`

	args := []interface{}{userID}
	argCount := 1

	if after != nil && *after != "" {
		argCount++
		query += fmt.Sprintf(" AND (f.created_at, f.id) < ($%d, $%d)", argCount, argCount+1)
		args = append(args, startTime, startID)
		argCount++
	}

	query += " ORDER BY f.created_at DESC, f.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argCount+1)
	args = append(args, first+1)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query following: %w", err)
	}
	defer rows.Close()

	var edges []models.FollowEdge
	for rows.Next() {
		var followingID uuid.UUID
		var createdAt time.Time
		var id uuid.UUID

		if err := rows.Scan(&followingID, &createdAt, &id); err != nil {
			return nil, fmt.Errorf("failed to scan following: %w", err)
		}

		cursor := encodeCursor(createdAt, id)
		edges = append(edges, models.FollowEdge{
			Cursor:     cursor,
			UserID:     followingID,
			FollowedAt: createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating following: %w", err)
	}

	hasNextPage := len(edges) > int(first)
	if hasNextPage {
		edges = edges[:first]
	}

	var pageInfo models.PageInfo
	pageInfo.HasNextPage = hasNextPage

	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor
	}

	totalCount, err := r.getFollowingCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	return &models.FollowConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}

// IsFollowing checks if followerID follows followingID
func (r *followRepository) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM follow_service_follows
			WHERE follower_id = $1 AND following_id = $2
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, followerID, followingID)
	if err != nil {
		return false, fmt.Errorf("failed to check following status: %w", err)
	}

	return exists, nil
}

// GetFollowStatus returns follow status for multiple users
func (r *followRepository) GetFollowStatus(ctx context.Context, userID uuid.UUID, targetUserIDs []uuid.UUID) ([]models.FollowStatus, error) {
	if len(targetUserIDs) == 0 {
		return []models.FollowStatus{}, nil
	}

	query := `
		SELECT following_id, TRUE as is_following
		FROM follow_service_follows
		WHERE follower_id = $1 AND following_id = ANY($2)
	`

	rows, err := r.db.QueryxContext(ctx, query, userID, targetUserIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query follow status: %w", err)
	}
	defer rows.Close()

	followingMap := make(map[uuid.UUID]bool)
	for rows.Next() {
		var targetID uuid.UUID
		var isFollowing bool
		if err := rows.Scan(&targetID, &isFollowing); err != nil {
			return nil, fmt.Errorf("failed to scan follow status: %w", err)
		}
		followingMap[targetID] = isFollowing
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating follow status: %w", err)
	}

	statuses := make([]models.FollowStatus, len(targetUserIDs))
	for i, targetID := range targetUserIDs {
		statuses[i] = models.FollowStatus{
			UserID:      targetID,
			IsFollowing: followingMap[targetID],
		}
	}

	return statuses, nil
}

// GetFollowersCounts returns follower and following counts for multiple users
func (r *followRepository) GetFollowersCounts(ctx context.Context, userIDs []uuid.UUID) ([]models.UserFollowCounts, error) {
	if len(userIDs) == 0 {
		return []models.UserFollowCounts{}, nil
	}

	query := `
		WITH followers AS (
			SELECT following_id as user_id, COUNT(*) as followers_count
			FROM follow_service_follows
			WHERE following_id = ANY($1)
			GROUP BY following_id
		),
		following AS (
			SELECT follower_id as user_id, COUNT(*) as following_count
			FROM follow_service_follows
			WHERE follower_id = ANY($1)
			GROUP BY follower_id
		)
		SELECT 
			u.user_id,
			COALESCE(followers.followers_count, 0) as followers_count,
			COALESCE(following.following_count, 0) as following_count
		FROM (SELECT UNNEST($1::uuid[]) as user_id) u
		LEFT JOIN followers ON u.user_id = followers.user_id
		LEFT JOIN following ON u.user_id = following.user_id
	`

	rows, err := r.db.QueryxContext(ctx, query, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query follow counts: %w", err)
	}
	defer rows.Close()

	var counts []models.UserFollowCounts
	for rows.Next() {
		var count models.UserFollowCounts
		if err := rows.Scan(&count.UserID, &count.FollowersCount, &count.FollowingCount); err != nil {
			return nil, fmt.Errorf("failed to scan follow counts: %w", err)
		}
		counts = append(counts, count)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating follow counts: %w", err)
	}

	return counts, nil
}

// Helper functions

func (r *followRepository) getFollowersCount(ctx context.Context, userID uuid.UUID) (int32, error) {
	query := `SELECT COUNT(*) FROM follow_service_follows WHERE following_id = $1`
	var count int32
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *followRepository) getFollowingCount(ctx context.Context, userID uuid.UUID) (int32, error) {
	query := `SELECT COUNT(*) FROM follow_service_follows WHERE follower_id = $1`
	var count int32
	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Cursor encoding/decoding

type CursorData struct {
	Timestamp time.Time
	ID        uuid.UUID
}

func encodeCursor(timestamp time.Time, id uuid.UUID) string {
	cursor := fmt.Sprintf("%d:%s", timestamp.Unix(), id.String())
	return base64.StdEncoding.EncodeToString([]byte(cursor))
}

func decodeCursor(cursor string) (*CursorData, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	var timestamp int64
	var idStr string
	_, err = fmt.Sscanf(string(decoded), "%d:%s", &timestamp, &idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cursor: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UUID from cursor: %w", err)
	}

	return &CursorData{
		Timestamp: time.Unix(timestamp, 0),
		ID:        id,
	}, nil
}
