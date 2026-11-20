package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// PasswordResetRepository provides operations on password_reset_tokens.
type PasswordResetRepository struct {
	db *db.DB
}

var _ PasswordResetStore = (*PasswordResetRepository)(nil)

// NewPasswordResetRepository constructs a PasswordResetRepository.
func NewPasswordResetRepository(database *db.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: database}
}

// Create inserts a new password reset token record.
func (r *PasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	if token.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			id = uuid.New()
		}
		token.ID = id
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	const q = `
		INSERT INTO password_reset_tokens
			(id, user_id, token_hash, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		token.ID, token.UserID, token.TokenHash,
		token.ExpiresAt, token.UsedAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("password_reset_repo.Create: %w", err)
	}
	return nil
}

// GetByTokenHash retrieves a reset token by its SHA-256 hash.
func (r *PasswordResetRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1`

	row := r.db.QueryRowContext(ctx, q, tokenHash)

	tok := &domain.PasswordResetToken{}
	err := row.Scan(&tok.ID, &tok.UserID, &tok.TokenHash, &tok.ExpiresAt, &tok.UsedAt, &tok.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("password_reset_repo.GetByTokenHash: %w", err)
	}
	return tok, nil
}

// MarkUsed marks a reset token as used by setting used_at to NOW().
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`

	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("password_reset_repo.MarkUsed: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired tokens from the table.
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM password_reset_tokens WHERE expires_at < NOW()`

	_, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("password_reset_repo.DeleteExpired: %w", err)
	}
	return nil
}

// DeleteByUser removes all reset tokens for a given user.
func (r *PasswordResetRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	const q = `DELETE FROM password_reset_tokens WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("password_reset_repo.DeleteByUser: %w", err)
	}
	return nil
}
