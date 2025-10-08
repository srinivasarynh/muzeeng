package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"feed-service/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type FeedRepository interface {
	// Feed retrieval
	GetFeed(ctx context.Context, userID uuid.UUID, limit int, after *string) (*models.PostConnection, error)

	// Feed cache management
	GetCachedFeed(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]models.Post, error)
	CacheFeedItems(ctx context.Context, userID uuid.UUID, posts []models.Post) error
	InvalidateUserFeed(ctx context.Context, userID uuid.UUID) error

	// Feed building
	BuildFeedForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Post, error)
	GetFollowingPosts(ctx context.Context, userID uuid.UUID, limit int, since time.Time) ([]models.Post, error)

	// Like status checks
	GetPostsWithLikeStatus(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)

	// Feed item insertion (for fan-out on write)
	InsertFeedItem(ctx context.Context, userID, postID uuid.UUID) error
	BulkInsertFeedItems(ctx context.Context, items []models.FeedCache) error

	// Feed cleanup
	CleanupOldFeedItems(ctx context.Context, olderThan time.Time) error
}

type feedRepository struct {
	db    *sqlx.DB
	redis *redis.Client
}

func NewFeedRepository(db *sqlx.DB, redis *redis.Client) FeedRepository {
	return &feedRepository{
		db:    db,
		redis: redis,
	}
}

// GetFeed retrieves paginated feed for a user
func (r *feedRepository) GetFeed(ctx context.Context, userID uuid.UUID, limit int, after *string) (*models.PostConnection, error) {
	var offset int
	if after != nil && *after != "" {
		decoded, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		offset = decoded
	}

	cachedPosts, err := r.GetCachedFeed(ctx, userID, limit+1, offset)
	if err == nil && len(cachedPosts) > 0 {
		return r.buildPostConnection(cachedPosts, limit, offset), nil
	}

	posts, err := r.BuildFeedForUser(ctx, userID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("failed to build feed: %w", err)
	}

	go func() {
		_ = r.CacheFeedItems(context.Background(), userID, posts)
	}()

	return r.buildPostConnection(posts, limit, offset), nil
}

// BuildFeedForUser creates a personalized feed using a hybrid approach
func (r *feedRepository) BuildFeedForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.Post, error) {
	query := `
		WITH following_users AS (
			SELECT followed_id 
			FROM feed_service_follows 
			WHERE follower_id = $1 AND deleted_at IS NULL
		),
		ranked_posts AS (
			SELECT 
				p.id,
				p.user_id,
				p.content,
				p.created_at,
				p.updated_at,
				p.likes_count,
				p.comments_count,
				-- Ranking algorithm: recency + engagement
				(
					-- Recency score (exponential decay)
					EXP(-EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 86400.0) * 0.5 +
					-- Engagement score
					(LOG(GREATEST(p.likes_count + 1, 1)) * 0.3) +
					(LOG(GREATEST(p.comments_count + 1, 1)) * 0.2)
				) AS feed_score
			FROM feed_service_posts p
			WHERE p.user_id IN (SELECT followed_id FROM following_users)
				AND p.created_at > NOW() - INTERVAL '30 days'
		)
		SELECT 
			id,
			user_id,
			content,
			created_at,
			updated_at,
			likes_count,
			comments_count
		FROM ranked_posts
		ORDER BY feed_score DESC, created_at DESC
		LIMIT $2
	`

	var posts []models.Post
	err := r.db.SelectContext(ctx, &posts, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ranked feed: %w", err)
	}

	return posts, nil
}

// GetFollowingPosts retrieves recent posts from users that the given user follows
func (r *feedRepository) GetFollowingPosts(ctx context.Context, userID uuid.UUID, limit int, since time.Time) ([]models.Post, error) {
	query := `
		SELECT 
			p.id,
			p.user_id,
			p.content,
			p.created_at,
			p.updated_at,
			p.likes_count,
			p.comments_count
		FROM feed_service_posts p
		INNER JOIN feed_service_follows f ON f.followed_id = p.user_id
		WHERE f.follower_id = $1
			AND f.deleted_at IS NULL
			AND p.created_at > $2
		ORDER BY p.created_at DESC
		LIMIT $3
	`

	var posts []models.Post
	err := r.db.SelectContext(ctx, &posts, query, userID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch following posts: %w", err)
	}

	return posts, nil
}

// GetCachedFeed retrieves feed from Redis cache
func (r *feedRepository) GetCachedFeed(ctx context.Context, userID uuid.UUID, limit int, offset int) ([]models.Post, error) {
	cacheKey := fmt.Sprintf("feed:%s", userID.String())

	postIDs, err := r.redis.ZRevRange(ctx, cacheKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cached feed: %w", err)
	}

	if len(postIDs) == 0 {
		return nil, fmt.Errorf("cache miss")
	}

	uuids := make([]uuid.UUID, 0, len(postIDs))
	for _, id := range postIDs {
		parsedID, err := uuid.Parse(id)
		if err != nil {
			continue
		}
		uuids = append(uuids, parsedID)
	}

	if len(uuids) == 0 {
		return nil, fmt.Errorf("no valid post IDs in cache")
	}

	query, args, err := sqlx.In(`
		SELECT id, user_id, content, created_at, updated_at, likes_count, comments_count
		FROM feed_service_posts
		WHERE id IN (?)
		ORDER BY created_at DESC
	`, uuids)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var posts []models.Post
	err = r.db.SelectContext(ctx, &posts, r.db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cached posts: %w", err)
	}

	return posts, nil
}

// CacheFeedItems stores feed items in Redis
func (r *feedRepository) CacheFeedItems(ctx context.Context, userID uuid.UUID, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}

	cacheKey := fmt.Sprintf("feed:%s", userID.String())
	pipe := r.redis.Pipeline()

	for _, post := range posts {
		score := float64(post.CreatedAt.Unix())
		pipe.ZAdd(ctx, cacheKey, redis.Z{
			Score:  score,
			Member: post.ID.String(),
		})
	}

	pipe.Expire(ctx, cacheKey, time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to cache feed items: %w", err)
	}

	return nil
}

// InvalidateUserFeed removes cached feed for a user
func (r *feedRepository) InvalidateUserFeed(ctx context.Context, userID uuid.UUID) error {
	cacheKey := fmt.Sprintf("feed:%s", userID.String())
	err := r.redis.Del(ctx, cacheKey).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate feed cache: %w", err)
	}
	return nil
}

// GetPostsWithLikeStatus retrieves like status for posts
func (r *feedRepository) GetPostsWithLikeStatus(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
	}

	query, args, err := sqlx.In(`
		SELECT post_id
		FROM feed_service_likes
		WHERE user_id = ? AND post_id IN (?) AND deleted_at IS NULL
	`, userID, postIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var likedPostIDs []uuid.UUID
	err = r.db.SelectContext(ctx, &likedPostIDs, r.db.Rebind(query), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch like status: %w", err)
	}

	result := make(map[uuid.UUID]bool, len(postIDs))
	for _, postID := range postIDs {
		result[postID] = false
	}
	for _, likedID := range likedPostIDs {
		result[likedID] = true
	}

	return result, nil
}

// InsertFeedItem inserts a single feed item (for fan-out on write)
func (r *feedRepository) InsertFeedItem(ctx context.Context, userID, postID uuid.UUID) error {
	query := `
		INSERT INTO feed_service_cache (id, user_id, post_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, post_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), userID, postID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert feed item: %w", err)
	}

	return nil
}

// BulkInsertFeedItems inserts multiple feed items efficiently
func (r *feedRepository) BulkInsertFeedItems(ctx context.Context, items []models.FeedCache) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO feed_service_cache (id, user_id, post_id, created_at)
		VALUES (:id, :user_id, :post_id, :created_at)
		ON CONFLICT (user_id, post_id) DO NOTHING
	`

	_, err := r.db.NamedExecContext(ctx, query, items)
	if err != nil {
		return fmt.Errorf("failed to bulk insert feed items: %w", err)
	}

	return nil
}

// CleanupOldFeedItems removes old feed cache entries
func (r *feedRepository) CleanupOldFeedItems(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM feed_service_cache WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to cleanup old feed items: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d old feed items\n", rowsAffected)

	return nil
}

// Helper functions

func (r *feedRepository) buildPostConnection(posts []models.Post, limit int, offset int) *models.PostConnection {
	hasNextPage := len(posts) > limit
	if hasNextPage {
		posts = posts[:limit]
	}

	edges := make([]models.PostEdge, len(posts))
	for i, post := range posts {
		cursor := encodeCursor(offset + i)
		edges[i] = models.PostEdge{
			Cursor: cursor,
			Node:   post,
		}
	}

	var endCursor, startCursor *string
	if len(edges) > 0 {
		ec := edges[len(edges)-1].Cursor
		sc := edges[0].Cursor
		endCursor = &ec
		startCursor = &sc
	}

	return &models.PostConnection{
		Edges: edges,
		PageInfo: models.PageInfo{
			EndCursor:       endCursor,
			HasNextPage:     hasNextPage,
			StartCursor:     startCursor,
			HasPreviousPage: offset > 0,
		},
		TotalCount: int32(len(posts)),
	}
}

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset)))
}

func decodeCursor(cursor string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	var offset int
	_, err = fmt.Sscanf(string(decoded), "%d", &offset)
	return offset, err
}
