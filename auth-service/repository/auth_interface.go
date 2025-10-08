package repository

import (
	models "auth-service/model"
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	UpdateUser(ctx context.Context, user *models.User) error

	// Refresh token operations
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredRefreshTokens(ctx context.Context) error

	// Token blacklist operations
	AddTokenToBlacklist(ctx context.Context, token string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	DeleteExpiredBlacklistedTokens(ctx context.Context) error

	// User role operations
	CreateUserRole(ctx context.Context, userRole *models.UserRole) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error)
	HasRole(ctx context.Context, userID uuid.UUID, role models.Role) (bool, error)
}
