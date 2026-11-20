package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// SsoStateRepository provides operations on sso_auth_states.
type SsoStateRepository struct {
	db *db.DB
}

var _ SsoStateStore = (*SsoStateRepository)(nil)

// NewSsoStateRepository constructs an SsoStateRepository.
func NewSsoStateRepository(database *db.DB) *SsoStateRepository {
	return &SsoStateRepository{db: database}
}

// Create inserts a new SSO state record.
func (r *SsoStateRepository) Create(ctx context.Context, state *domain.SSOAuthState) error {
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}

	const q = `
		INSERT INTO sso_auth_states (state, provider, redirect_to, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.ExecContext(ctx, q,
		state.State, string(state.Provider), state.RedirectTo,
		state.ExpiresAt, state.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("sso_state_repo.Create: %w", err)
	}
	return nil
}

// Get retrieves an SSO auth state by state string.
func (r *SsoStateRepository) Get(ctx context.Context, state string) (*domain.SSOAuthState, error) {
	const q = `
		SELECT state, provider, redirect_to, expires_at, created_at
		FROM sso_auth_states
		WHERE state = $1`

	row := r.db.QueryRowContext(ctx, q, state)

	s := &domain.SSOAuthState{}
	var providerStr string
	err := row.Scan(&s.State, &providerStr, &s.RedirectTo, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("sso_state_repo.Get: %w", err)
	}
	s.Provider = domain.SSOProvider(providerStr)
	return s, nil
}

// Delete removes an SSO auth state by state string.
func (r *SsoStateRepository) Delete(ctx context.Context, state string) error {
	const q = `DELETE FROM sso_auth_states WHERE state = $1`

	_, err := r.db.ExecContext(ctx, q, state)
	if err != nil {
		return fmt.Errorf("sso_state_repo.Delete: %w", err)
	}
	return nil
}

// DeleteExpired removes expired SSO auth states.
func (r *SsoStateRepository) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM sso_auth_states WHERE expires_at < NOW()`

	_, err := r.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("sso_state_repo.DeleteExpired: %w", err)
	}
	return nil
}
