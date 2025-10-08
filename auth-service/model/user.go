package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Username       string    `json:"username" db:"username"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	Bio            *string   `json:"bio,omitempty" db:"bio"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	FollowersCount int32     `json:"followers_count" db:"followers_count"`
	FollowingCount int32     `json:"following_count" db:"following_count"`
	PostsCount     int32     `json:"posts_count" db:"posts_count"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
}

type TokenBlacklist struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type UserRole struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      Role      `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
