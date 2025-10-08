package models

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	ID          uuid.UUID `json:"id" db:"id"`
	FollowerID  uuid.UUID `json:"follower_id" db:"follower_id"`   // User who is following
	FollowingID uuid.UUID `json:"following_id" db:"following_id"` // User being followed
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type FollowEdge struct {
	Cursor     string    `json:"cursor"`
	UserID     uuid.UUID `json:"user_id"`
	FollowedAt time.Time `json:"followed_at"`
}

type PageInfo struct {
	EndCursor       *string `json:"end_cursor,omitempty"`
	HasNextPage     bool    `json:"has_next_page"`
	StartCursor     *string `json:"start_cursor,omitempty"`
	HasPreviousPage bool    `json:"has_previous_page"`
}

type FollowConnection struct {
	Edges      []FollowEdge `json:"edges"`
	PageInfo   PageInfo     `json:"page_info"`
	TotalCount int32        `json:"total_count"`
}

type FollowStatus struct {
	UserID      uuid.UUID `json:"user_id"`
	IsFollowing bool      `json:"is_following"`
}

type UserFollowCounts struct {
	UserID         uuid.UUID `json:"user_id"`
	FollowersCount int32     `json:"followers_count"`
	FollowingCount int32     `json:"following_count"`
}
