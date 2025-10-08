package models

import (
	"time"

	"github.com/google/uuid"
)

// FeedCache represents a cached feed item for a user
type FeedCache struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Post represents a post in the feed
type Post struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Content       string    `json:"content" db:"content"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	LikesCount    int32     `json:"likes_count" db:"likes_count"`
	CommentsCount int32     `json:"comments_count" db:"comments_count"`
}

// PostWithLikeStatus extends Post with user-specific like status
type PostWithLikeStatus struct {
	Post
	IsLiked *bool `json:"is_liked,omitempty"`
}

// FeedItem represents a single item in the user's feed
type FeedItem struct {
	Post     Post      `json:"post"`
	AuthorID uuid.UUID `json:"author_id"`
	IsLiked  *bool     `json:"is_liked,omitempty"`
	RankedAt time.Time `json:"ranked_at"`
	Score    float64   `json:"score"`
}

// PostEdge represents an edge in the connection
type PostEdge struct {
	Cursor string `json:"cursor"`
	Node   Post   `json:"node"`
}

// PageInfo contains pagination information
type PageInfo struct {
	EndCursor       *string `json:"end_cursor,omitempty"`
	HasNextPage     bool    `json:"has_next_page"`
	StartCursor     *string `json:"start_cursor,omitempty"`
	HasPreviousPage bool    `json:"has_previous_page"`
}

// PostConnection represents a paginated list of posts
type PostConnection struct {
	Edges      []PostEdge `json:"edges"`
	PageInfo   PageInfo   `json:"page_info"`
	TotalCount int32      `json:"total_count"`
}

// FeedStats contains statistics about a user's feed
type FeedStats struct {
	UserID          uuid.UUID `json:"user_id"`
	TotalPosts      int32     `json:"total_posts"`
	UnreadPosts     int32     `json:"unread_posts"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
	FollowingCount  int32     `json:"following_count"`
}
