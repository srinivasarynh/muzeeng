package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"like-service/model"
)

type LikeRepository interface {
	CreateLike(ctx context.Context, postID, userID uuid.UUID) error
	DeleteLike(ctx context.Context, postID, userID uuid.UUID) error
	GetLikeByPostAndUser(ctx context.Context, postID, userID uuid.UUID) (*models.Like, error)
	GetLikeCountByPost(ctx context.Context, postID uuid.UUID) (int32, error)
	GetRecentLikersByPost(ctx context.Context, postID uuid.UUID, limit int32) ([]uuid.UUID, error)
	IsPostLikedByUser(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	GetPostLikesByUsers(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) ([]models.PostLikeStatus, error)
	GetLikesByPost(ctx context.Context, postID uuid.UUID) ([]*models.Like, error)
}

type likeRepository struct {
	db *sqlx.DB
}

func NewLikeRepository(db *sqlx.DB) LikeRepository {
	return &likeRepository{db: db}
}

// CreateLike adds a new like for a post by a user
func (r *likeRepository) CreateLike(ctx context.Context, postID, userID uuid.UUID) error {
	query := `
		INSERT INTO like_service_likes (id, post_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (post_id, user_id) DO NOTHING
	`

	likeID := uuid.New()
	now := time.Now()

	result, err := r.db.ExecContext(ctx, query, likeID, postID, userID, now)
	if err != nil {
		return fmt.Errorf("failed to create like: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("like already exists")
	}

	return nil
}

// DeleteLike removes a like for a post by a user
func (r *likeRepository) DeleteLike(ctx context.Context, postID, userID uuid.UUID) error {
	query := `
		DELETE FROM like_service_likes
		WHERE post_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, postID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete like: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("like not found")
	}

	return nil
}

// GetLikeByPostAndUser retrieves a specific like
func (r *likeRepository) GetLikeByPostAndUser(ctx context.Context, postID, userID uuid.UUID) (*models.Like, error) {
	query := `
		SELECT id, post_id, user_id, created_at
		FROM like_service_likes
		WHERE post_id = $1 AND user_id = $2
	`

	var like models.Like
	err := r.db.GetContext(ctx, &like, query, postID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get like: %w", err)
	}

	return &like, nil
}

// GetLikeCountByPost returns the total number of likes for a post
func (r *likeRepository) GetLikeCountByPost(ctx context.Context, postID uuid.UUID) (int32, error) {
	query := `
		SELECT COUNT(*)
		FROM like_service_likes
		WHERE post_id = $1
	`

	var count int32
	err := r.db.GetContext(ctx, &count, query, postID)
	if err != nil {
		return 0, fmt.Errorf("failed to get like count: %w", err)
	}

	return count, nil
}

// GetRecentLikersByPost returns the most recent users who liked a post
func (r *likeRepository) GetRecentLikersByPost(ctx context.Context, postID uuid.UUID, limit int32) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT user_id
		FROM like_service_likes
		WHERE post_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var userIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &userIDs, query, postID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent likers: %w", err)
	}

	return userIDs, nil
}

// IsPostLikedByUser checks if a user has liked a specific post
func (r *likeRepository) IsPostLikedByUser(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM like_service_likes
			WHERE post_id = $1 AND user_id = $2
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, postID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check if post is liked: %w", err)
	}

	return exists, nil
}

// GetPostLikesByUsers checks like status for multiple posts by a specific user
func (r *likeRepository) GetPostLikesByUsers(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) ([]models.PostLikeStatus, error) {
	if len(postIDs) == 0 {
		return []models.PostLikeStatus{}, nil
	}

	query := `
		SELECT post_id, true as is_liked
		FROM like_service_likes
		WHERE post_id = ANY($1) AND user_id = $2
	`

	var likedPosts []models.PostLikeStatus
	err := r.db.SelectContext(ctx, &likedPosts, query, postIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post likes by user: %w", err)
	}

	likedMap := make(map[uuid.UUID]bool)
	for _, liked := range likedPosts {
		likedMap[liked.PostID] = true
	}

	result := make([]models.PostLikeStatus, 0, len(postIDs))
	for _, postID := range postIDs {
		result = append(result, models.PostLikeStatus{
			PostID:  postID,
			IsLiked: likedMap[postID],
		})
	}

	return result, nil
}

// GetLikesByPost retrieves all likes for a specific post
func (r *likeRepository) GetLikesByPost(ctx context.Context, postID uuid.UUID) ([]*models.Like, error) {
	query := `
		SELECT id, post_id, user_id, created_at
		FROM like_service_likes
		WHERE post_id = $1
		ORDER BY created_at DESC
	`

	var likes []*models.Like
	err := r.db.SelectContext(ctx, &likes, query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to get likes by post: %w", err)
	}

	return likes, nil
}
