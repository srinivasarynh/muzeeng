package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"auth-service/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type authRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO auth_users (id, username, email, password_hash, bio, created_at, updated_at, 
		                   followers_count, following_count, posts_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Bio,
		user.CreatedAt, user.UpdatedAt, user.FollowersCount, user.FollowingCount, user.PostsCount,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *authRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, password_hash, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM auth_users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, password_hash, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM auth_users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *authRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, password_hash, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM auth_users
		WHERE username = $1
	`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *authRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE auth_users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *authRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE auth_users
		SET username = $1, email = $2, bio = $3, updated_at = $4,
		    followers_count = $5, following_count = $6, posts_count = $7
		WHERE id = $8
	`

	user.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(
		ctx, query,
		user.Username, user.Email, user.Bio, user.UpdatedAt,
		user.FollowersCount, user.FollowingCount, user.PostsCount, user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Refresh token operations

func (r *authRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO auth_refresh_tokens (id, user_id, token, expires_at, created_at, is_revoked)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		token.ID, token.UserID, token.Token, token.ExpiresAt, token.CreatedAt, token.IsRevoked,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *authRepository) GetRefreshToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	query := `
		SELECT id, user_id, token, expires_at, created_at, is_revoked
		FROM auth_refresh_tokens
		WHERE token = $1 AND is_revoked = false AND expires_at > NOW()
	`

	err := r.db.GetContext(ctx, &refreshToken, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("refresh token not found or expired")
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return &refreshToken, nil
}

func (r *authRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `
		UPDATE auth_refresh_tokens
		SET is_revoked = true
		WHERE token = $1
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}

func (r *authRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE auth_refresh_tokens
		SET is_revoked = true
		WHERE user_id = $1 AND is_revoked = false
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user refresh tokens: %w", err)
	}
	return nil
}

func (r *authRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	query := `
		DELETE FROM auth_refresh_tokens
		WHERE expires_at < NOW() OR is_revoked = true
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired refresh tokens: %w", err)
	}
	return nil
}

// Token blacklist operations

func (r *authRepository) AddTokenToBlacklist(ctx context.Context, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO auth_token_blacklist (id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), token, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}
	return nil
}

func (r *authRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM auth_token_blacklist
			WHERE token = $1 AND expires_at > NOW()
		)
	`

	err := r.db.GetContext(ctx, &exists, query, token)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	return exists, nil
}

func (r *authRepository) DeleteExpiredBlacklistedTokens(ctx context.Context) error {
	query := `
		DELETE FROM auth_token_blacklist
		WHERE expires_at < NOW()
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired blacklisted tokens: %w", err)
	}
	return nil
}

// User role operations

func (r *authRepository) CreateUserRole(ctx context.Context, userRole *models.UserRole) error {
	query := `
		INSERT INTO auth_user_roles (id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query, userRole.ID, userRole.UserID, userRole.Role, userRole.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user role: %w", err)
	}
	return nil
}

func (r *authRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	var roles []models.Role
	query := `
		SELECT role
		FROM auth_user_roles
		WHERE user_id = $1
	`

	err := r.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

func (r *authRepository) HasRole(ctx context.Context, userID uuid.UUID, role models.Role) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM auth_user_roles
			WHERE user_id = $1 AND role = $2
		)
	`

	err := r.db.GetContext(ctx, &exists, query, userID, role)
	if err != nil {
		return false, fmt.Errorf("failed to check user role: %w", err)
	}
	return exists, nil
}
