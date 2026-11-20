package repository

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// SessionRepository provides operations on user_sessions.
type SessionRepository struct {
	db *db.DB
}

var _ SessionStore = (*SessionRepository)(nil)

// NewSessionRepository constructs a SessionRepository.
func NewSessionRepository(database *db.DB) *SessionRepository {
	return &SessionRepository{db: database}
}

// Create inserts a new session record.
func (r *SessionRepository) Create(ctx context.Context, s *domain.UserSession) error {
	const q = `
		INSERT INTO user_sessions
			(id, user_id, refresh_token_hash, user_agent, client_ip, is_revoked, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	var clientIP *string
	if s.ClientIP != nil {
		str := s.ClientIP.String()
		clientIP = &str
	}

	_, err := r.db.ExecContext(ctx, q,
		s.ID, s.UserID, s.RefreshTokenHash, s.UserAgent, clientIP,
		s.IsRevoked, s.ExpiresAt, s.CreatedAt, s.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("session_repo.Create: %w", err)
	}
	return nil
}

// GetByTokenHash retrieves a session by its refresh token hash.
func (r *SessionRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.UserSession, error) {
	const q = `
		SELECT id, user_id, refresh_token_hash, user_agent, client_ip, is_revoked, expires_at, created_at, last_used_at
		FROM user_sessions
		WHERE refresh_token_hash = $1`

	row := r.db.QueryRowContext(ctx, q, hash)

	s := &domain.UserSession{}
	var clientIPStr *string

	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &clientIPStr,
		&s.IsRevoked, &s.ExpiresAt, &s.CreatedAt, &s.LastUsedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("session_repo.GetByTokenHash: %w", err)
	}

	if clientIPStr != nil {
		addr, parseErr := netip.ParseAddr(*clientIPStr)
		if parseErr == nil {
			s.ClientIP = &addr
		}
	}

	return s, nil
}

// RevokeByTokenHash marks a session as revoked.
func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, hash string) error {
	const q = `UPDATE user_sessions SET is_revoked = TRUE WHERE refresh_token_hash = $1`

	_, err := r.db.ExecContext(ctx, q, hash)
	if err != nil {
		return fmt.Errorf("session_repo.RevokeByTokenHash: %w", err)
	}
	return nil
}

// RevokeByUser marks all sessions for a user as revoked.
func (r *SessionRepository) RevokeByUser(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE user_sessions SET is_revoked = TRUE WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("session_repo.RevokeByUser: %w", err)
	}
	return nil
}

// CleanupExpired deletes sessions that are expired or already revoked.
func (r *SessionRepository) CleanupExpired(ctx context.Context) error {
	const q = `DELETE FROM user_sessions WHERE expires_at < NOW() OR is_revoked = TRUE`

	_, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("session_repo.CleanupExpired: %w", err)
	}
	return nil
}
