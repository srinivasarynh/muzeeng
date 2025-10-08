package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"user-service/model"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, userID uuid.UUID, input *models.UpdateUserInput) (*models.User, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*models.User, error)
	IncrementPostsCount(ctx context.Context, userID uuid.UUID) error
	DecrementPostsCount(ctx context.Context, userID uuid.UUID) error
	CheckFollowStatus(ctx context.Context, userID, followerID uuid.UUID) (bool, error)
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, username, email, bio, created_at, updated_at, 
		       followers_count, following_count, posts_count
		FROM user_service_users
		WHERE id = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM user_service_users
		WHERE email = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM user_service_users
		WHERE username = $1
	`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, userID uuid.UUID, input *models.UpdateUserInput) (*models.User, error) {
	query := "UPDATE user_service_users SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if input.Username != nil {
		query += fmt.Sprintf(", username = $%d", argCount)
		args = append(args, *input.Username)
		argCount++
	}

	if input.Email != nil {
		query += fmt.Sprintf(", email = $%d", argCount)
		args = append(args, *input.Email)
		argCount++
	}

	if input.Bio != nil {
		query += fmt.Sprintf(", bio = $%d", argCount)
		args = append(args, *input.Bio)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, username, email, bio, created_at, updated_at, followers_count, following_count, posts_count", argCount)
	args = append(args, userID)

	var user models.User
	err := r.db.GetContext(ctx, &user, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*models.User, error) {
	if len(userIDs) == 0 {
		return []*models.User{}, nil
	}

	query := `
		SELECT id, username, email, bio, created_at, updated_at,
		       followers_count, following_count, posts_count
		FROM user_service_users
		WHERE id = ANY($1)
	`

	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}

	return users, nil
}

func (r *userRepository) IncrementPostsCount(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_service_users
		SET posts_count = posts_count + 1, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment posts count: %w", err)
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

func (r *userRepository) DecrementPostsCount(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_service_users
		SET posts_count = GREATEST(posts_count - 1, 0), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to decrement posts count: %w", err)
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

func (r *userRepository) CheckFollowStatus(ctx context.Context, userID, followerID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_service_follows
			WHERE follower_id = $1 AND following_id = $2
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, followerID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check follow status: %w", err)
	}

	return exists, nil
}
