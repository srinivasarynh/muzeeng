package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Username       string    `json:"username" db:"username"`
	Email          string    `json:"email" db:"email"`
	Bio            *string   `json:"bio,omitempty" db:"bio"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	FollowersCount int32     `json:"followers_count" db:"followers_count"`
	FollowingCount int32     `json:"following_count" db:"following_count"`
	PostsCount     int32     `json:"posts_count" db:"posts_count"`
}

type UserProfile struct {
	User
	IsFollowing *bool `json:"is_following,omitempty"`
}

type UpdateUserInput struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Bio      *string `json:"bio,omitempty"`
}
