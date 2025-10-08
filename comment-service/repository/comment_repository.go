package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"comment-service/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *models.Comment) error
	GetByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error)
	GetPostComments(ctx context.Context, postID uuid.UUID, first int32, after *string) (*models.CommentConnection, error)
	Update(ctx context.Context, comment *models.Comment) error
	Delete(ctx context.Context, commentID uuid.UUID) error
	GetTotalCountByPost(ctx context.Context, postID uuid.UUID) (int32, error)
	CheckOwnership(ctx context.Context, commentID, userID uuid.UUID) (bool, error)
}

type commentRepository struct {
	db *sqlx.DB
}

func NewCommentRepository(db *sqlx.DB) CommentRepository {
	return &commentRepository{db: db}
}

// Create inserts a new comment into the database
func (r *commentRepository) Create(ctx context.Context, comment *models.Comment) error {
	query := `
		INSERT INTO comment_service_comments (id, post_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, post_id, user_id, content, created_at, updated_at
	`

	err := r.db.QueryRowxContext(
		ctx,
		query,
		comment.ID,
		comment.PostID,
		comment.UserID,
		comment.Content,
		comment.CreatedAt,
		comment.UpdatedAt,
	).StructScan(comment)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// GetByID retrieves a comment by its ID
func (r *commentRepository) GetByID(ctx context.Context, commentID uuid.UUID) (*models.Comment, error) {
	query := `
		SELECT id, post_id, user_id, content, created_at, updated_at
		FROM comment_service_comments
		WHERE id = $1
	`

	var comment models.Comment
	err := r.db.GetContext(ctx, &comment, query, commentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("comment not found")
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return &comment, nil
}

// GetPostComments retrieves comments for a post with cursor-based pagination
func (r *commentRepository) GetPostComments(ctx context.Context, postID uuid.UUID, first int32, after *string) (*models.CommentConnection, error) {
	// Default pagination limit
	if first <= 0 || first > 100 {
		first = 10
	}

	var comments []models.Comment
	var query string
	var args []interface{}

	if after != nil && *after != "" {
		// Decode cursor
		cursorTime, err := decodeCursor(*after)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		query = `
			SELECT id, post_id, user_id, content, created_at, updated_at
			FROM comment_service_comments
			WHERE post_id = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{postID, cursorTime, first + 1}
	} else {
		query = `
			SELECT id, post_id, user_id, content, created_at, updated_at
			FROM comment_service_comments
			WHERE post_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{postID, first + 1}
	}

	err := r.db.SelectContext(ctx, &comments, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	hasNextPage := len(comments) > int(first)
	if hasNextPage {
		comments = comments[:first]
	}

	edges := make([]models.CommentEdge, len(comments))
	for i, comment := range comments {
		edges[i] = models.CommentEdge{
			Cursor: encodeCursor(comment.CreatedAt),
			Node:   comment,
		}
	}

	totalCount, err := r.GetTotalCountByPost(ctx, postID)
	if err != nil {
		totalCount = 0
	}

	pageInfo := models.PageInfo{
		HasNextPage:     hasNextPage,
		HasPreviousPage: after != nil && *after != "",
	}

	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor
	}

	return &models.CommentConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: totalCount,
	}, nil
}

// Update updates an existing comment
func (r *commentRepository) Update(ctx context.Context, comment *models.Comment) error {
	query := `
		UPDATE comment_service_comments
		SET content = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, post_id, user_id, content, created_at, updated_at
	`

	err := r.db.QueryRowxContext(
		ctx,
		query,
		comment.Content,
		comment.UpdatedAt,
		comment.ID,
	).StructScan(comment)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("comment not found")
		}
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// Delete removes a comment from the database
func (r *commentRepository) Delete(ctx context.Context, commentID uuid.UUID) error {
	query := `DELETE FROM comment_service_comments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, commentID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("comment not found")
	}

	return nil
}

// GetTotalCountByPost returns the total number of comments for a post
func (r *commentRepository) GetTotalCountByPost(ctx context.Context, postID uuid.UUID) (int32, error) {
	query := `SELECT COUNT(*) FROM comment_service_comments WHERE post_id = $1`

	var count int32
	err := r.db.GetContext(ctx, &count, query, postID)
	if err != nil {
		return 0, fmt.Errorf("failed to get comment count: %w", err)
	}

	return count, nil
}

// CheckOwnership verifies if a user owns a specific comment
func (r *commentRepository) CheckOwnership(ctx context.Context, commentID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM comment_service_comments WHERE id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, commentID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check comment ownership: %w", err)
	}

	return exists, nil
}

// encodeCursor encodes a timestamp into a base64 cursor
func encodeCursor(t time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(t.Format(time.RFC3339Nano)))
}

// decodeCursor decodes a base64 cursor into a timestamp
func decodeCursor(cursor string) (time.Time, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.Parse(time.RFC3339Nano, string(decoded))
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}
