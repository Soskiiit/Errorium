package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// UserRepository provides CRUD operations for the users table.
type UserRepository struct {
	db *db.DB
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(database *db.DB) *UserRepository {
	return &UserRepository{db: database}
}

// Create inserts a new user record.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (id, email, username, avatar_url, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, q,
		u.ID, u.Email, u.Username, u.AvatarURL, u.PasswordHash,
		string(u.Role), u.IsActive, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("user_repo.Create: %w", err)
	}
	return nil
}

// GetByID fetches a user by primary key.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, email, username, avatar_url, password_hash, role, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1`

	return r.scanUser(r.db.QueryRowContext(ctx, q, id))
}

// GetByEmail fetches a user by email address.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, username, avatar_url, password_hash, role, is_active, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1`

	return r.scanUser(r.db.QueryRowContext(ctx, q, email))
}

// UpdateProfile updates mutable profile fields: username, avatar_url, updated_at.
func (r *UserRepository) UpdateProfile(ctx context.Context, u *domain.User) error {
	const q = `
		UPDATE users
		SET username = $1, avatar_url = $2, updated_at = $3
		WHERE id = $4`

	_, err := r.db.ExecContext(ctx, q, u.Username, u.AvatarURL, u.UpdatedAt, u.ID)
	if err != nil {
		return fmt.Errorf("user_repo.UpdateProfile: %w", err)
	}
	return nil
}

// UpdateLastLogin sets the last_login_at timestamp for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	const q = `UPDATE users SET last_login_at = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, q, t, id)
	if err != nil {
		return fmt.Errorf("user_repo.UpdateLastLogin: %w", err)
	}
	return nil
}

// scanUser reads a *domain.User from a single *sql.Row.
func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	var roleStr string
	err := row.Scan(
		&u.ID, &u.Email, &u.Username, &u.AvatarURL, &u.PasswordHash,
		&roleStr, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user_repo.scan: %w", err)
	}
	u.Role = domain.Role(roleStr)
	return u, nil
}
