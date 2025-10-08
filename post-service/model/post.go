package models

import (
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Content       string    `json:"content" db:"content"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	LikesCount    int32     `json:"likes_count" db:"likes_count"`
	CommentsCount int32     `json:"comments_count" db:"comments_count"`
}

type PostWithLikeStatus struct {
	Post
	IsLiked *bool `json:"is_liked,omitempty"`
}

type PostEdge struct {
	Cursor string `json:"cursor"`
	Node   Post   `json:"node"`
}

type PageInfo struct {
	EndCursor       *string `json:"end_cursor,omitempty"`
	HasNextPage     bool    `json:"has_next_page"`
	StartCursor     *string `json:"start_cursor,omitempty"`
	HasPreviousPage bool    `json:"has_previous_page"`
}

type PostConnection struct {
	Edges      []PostEdge `json:"edges"`
	PageInfo   PageInfo   `json:"page_info"`
	TotalCount int32      `json:"total_count"`
}
