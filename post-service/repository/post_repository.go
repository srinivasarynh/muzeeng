package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"post-service/model"
)

type PostRepository interface {
	Create(ctx context.Context, post *models.Post) error
	GetByID(ctx context.Context, postID uuid.UUID, requestingUserID *uuid.UUID) (*models.PostWithLikeStatus, error)
	Update(ctx context.Context, post *models.Post) error
	Delete(ctx context.Context, postID uuid.UUID) error
	GetUserPosts(ctx context.Context, userID uuid.UUID, first int32, after *string, requestingUserID *uuid.UUID) (*models.PostConnection, error)
	IncrementCommentsCount(ctx context.Context, postID uuid.UUID) error
	DecrementCommentsCount(ctx context.Context, postID uuid.UUID) error
	IncrementLikesCount(ctx context.Context, postID uuid.UUID) error
	DecrementLikesCount(ctx context.Context, postID uuid.UUID) error
}

type postRepository struct {
	db *sqlx.DB
}

func NewPostRepository(db *sqlx.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *models.Post) error {
	query := `
		INSERT INTO post_service_posts (id, user_id, content, created_at, updated_at, likes_count, comments_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		post.ID,
		post.UserID,
		post.Content,
		post.CreatedAt,
		post.UpdatedAt,
		post.LikesCount,
		post.CommentsCount,
	)
	return err
}

func (r *postRepository) GetByID(ctx context.Context, postID uuid.UUID, requestingUserID *uuid.UUID) (*models.PostWithLikeStatus, error) {
	var post models.Post
	var isLiked *bool

	if requestingUserID != nil {
		query := `
			SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at, 
			       p.likes_count, p.comments_count,
			       EXISTS(SELECT 1 FROM likes WHERE post_id = p.id AND user_id = $2) as is_liked
			FROM post_service_posts p
			WHERE p.id = $1
		`
		var liked bool
		err := r.db.GetContext(ctx, &struct {
			models.Post
			IsLikedVal bool `db:"is_liked"`
		}{
			Post: post,
		}, query, postID, requestingUserID)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("post not found")
			}
			return nil, err
		}

		// Query again with proper scanning
		row := r.db.QueryRowContext(ctx, query, postID, requestingUserID)
		err = row.Scan(
			&post.ID,
			&post.UserID,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.LikesCount,
			&post.CommentsCount,
			&liked,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("post not found")
			}
			return nil, err
		}
		isLiked = &liked
	} else {
		query := `
			SELECT id, user_id, content, created_at, updated_at, likes_count, comments_count
			FROM post_service_posts
			WHERE id = $1
		`
		err := r.db.GetContext(ctx, &post, query, postID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("post not found")
			}
			return nil, err
		}
	}

	return &models.PostWithLikeStatus{
		Post:    post,
		IsLiked: isLiked,
	}, nil
}

func (r *postRepository) Update(ctx context.Context, post *models.Post) error {
	query := `
		UPDATE post_service_posts 
		SET content = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`
	result, err := r.db.ExecContext(ctx, query, post.Content, post.UpdatedAt, post.ID, post.UserID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found or unauthorized")
	}

	return nil
}

func (r *postRepository) Delete(ctx context.Context, postID uuid.UUID) error {
	query := `DELETE FROM post_service_posts WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, postID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

func (r *postRepository) GetUserPosts(ctx context.Context, userID uuid.UUID, first int32, after *string, requestingUserID *uuid.UUID) (*models.PostConnection, error) {
	var afterTime time.Time
	var afterID uuid.UUID

	if after != nil && *after != "" {
		decoded, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		afterTime = decoded.Timestamp
		afterID = decoded.ID
	}

	// Get total count
	var totalCount int32
	countQuery := `SELECT COUNT(*) FROM post_service_posts WHERE user_id = $1`
	err := r.db.GetContext(ctx, &totalCount, countQuery, userID)
	if err != nil {
		return nil, err
	}

	// Build main query
	var posts []models.Post
	var query string
	var args []interface{}

	if requestingUserID != nil {
		if after != nil && *after != "" {
			query = `
				SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at, 
				       p.likes_count, p.comments_count
				FROM post_service_posts p
				WHERE p.user_id = $1 
				  AND (p.created_at, p.id) < ($2, $3)
				ORDER BY p.created_at DESC, p.id DESC
				LIMIT $4
			`
			args = []interface{}{userID, afterTime, afterID, first + 1}
		} else {
			query = `
				SELECT p.id, p.user_id, p.content, p.created_at, p.updated_at, 
				       p.likes_count, p.comments_count
				FROM post_service_posts p
				WHERE p.user_id = $1
				ORDER BY p.created_at DESC, p.id DESC
				LIMIT $2
			`
			args = []interface{}{userID, first + 1}
		}
	} else {
		if after != nil && *after != "" {
			query = `
				SELECT id, user_id, content, created_at, updated_at, likes_count, comments_count
				FROM post_service_posts
				WHERE user_id = $1 
				  AND (created_at, id) < ($2, $3)
				ORDER BY created_at DESC, id DESC
				LIMIT $4
			`
			args = []interface{}{userID, afterTime, afterID, first + 1}
		} else {
			query = `
				SELECT id, user_id, content, created_at, updated_at, likes_count, comments_count
				FROM post_service_posts
				WHERE user_id = $1
				ORDER BY created_at DESC, id DESC
				LIMIT $2
			`
			args = []interface{}{userID, first + 1}
		}
	}

	err = r.db.SelectContext(ctx, &posts, query, args...)
	if err != nil {
		return nil, err
	}

	hasNextPage := len(posts) > int(first)
	if hasNextPage {
		posts = posts[:first]
	}

	likeStatusMap := make(map[uuid.UUID]bool)
	if requestingUserID != nil && len(posts) > 0 {
		postIDs := make([]uuid.UUID, len(posts))
		for i, post := range posts {
			postIDs[i] = post.ID
		}

		likeQuery := `
			SELECT post_id 
			FROM post_service_likes 
			WHERE user_id = $1 AND post_id = ANY($2)
		`
		var likedPostIDs []uuid.UUID
		err = r.db.SelectContext(ctx, &likedPostIDs, likeQuery, requestingUserID, postIDs)
		if err != nil {
			return nil, err
		}

		for _, postID := range likedPostIDs {
			likeStatusMap[postID] = true
		}
	}

	edges := make([]models.PostEdge, len(posts))
	for i, post := range posts {
		cursor := encodeCursor(post.CreatedAt, post.ID)
		edges[i] = models.PostEdge{
			Cursor: cursor,
			Node:   post,
		}
	}

	var endCursor *string
	var startCursor *string
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

	return &models.PostConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}

func (r *postRepository) IncrementCommentsCount(ctx context.Context, postID uuid.UUID) error {
	query := `UPDATE post_service_posts SET comments_count = comments_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, postID)
	return err
}

func (r *postRepository) DecrementCommentsCount(ctx context.Context, postID uuid.UUID) error {
	query := `UPDATE post_service_posts SET comments_count = GREATEST(comments_count - 1, 0) WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, postID)
	return err
}

func (r *postRepository) IncrementLikesCount(ctx context.Context, postID uuid.UUID) error {
	query := `UPDATE post_service_posts SET likes_count = likes_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, postID)
	return err
}

func (r *postRepository) DecrementLikesCount(ctx context.Context, postID uuid.UUID) error {
	query := `UPDATE post_service_posts SET likes_count = GREATEST(likes_count - 1, 0) WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, postID)
	return err
}

// Cursor encoding/decoding helpers
type Cursor struct {
	Timestamp time.Time
	ID        uuid.UUID
}

func encodeCursor(timestamp time.Time, id uuid.UUID) string {
	cursorStr := fmt.Sprintf("%d:%s", timestamp.Unix(), id.String())
	return base64.StdEncoding.EncodeToString([]byte(cursorStr))
}

func decodeCursor(cursor string) (*Cursor, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	var timestamp int64
	var idStr string
	_, err = fmt.Sscanf(string(decoded), "%d:%s", &timestamp, &idStr)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	return &Cursor{
		Timestamp: time.Unix(timestamp, 0),
		ID:        id,
	}, nil
}
