package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CommentEdge struct {
	Cursor string  `json:"cursor"`
	Node   Comment `json:"node"`
}

type PageInfo struct {
	EndCursor       *string `json:"end_cursor,omitempty"`
	HasNextPage     bool    `json:"has_next_page"`
	StartCursor     *string `json:"start_cursor,omitempty"`
	HasPreviousPage bool    `json:"has_previous_page"`
}

type CommentConnection struct {
	Edges      []CommentEdge `json:"edges"`
	PageInfo   PageInfo      `json:"page_info"`
	TotalCount int32         `json:"total_count"`
}
