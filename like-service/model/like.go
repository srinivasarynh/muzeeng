package models

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PostID    uuid.UUID `json:"post_id" db:"post_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type LikeInfo struct {
	Count                int32       `json:"count"`
	IsLikedByCurrentUser *bool       `json:"is_liked_by_current_user,omitempty"`
	RecentLikerIDs       []uuid.UUID `json:"recent_liker_ids"`
}

type PostLikeStatus struct {
	PostID  uuid.UUID `json:"post_id"`
	IsLiked bool      `json:"is_liked"`
}
