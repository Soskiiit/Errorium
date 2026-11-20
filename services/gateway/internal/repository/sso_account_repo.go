package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// SSOAccountRepository provides persistence operations for the user_sso_accounts table.
type SSOAccountRepository struct {
	db *db.DB
}

var _ SSOAccountStore = (*SSOAccountRepository)(nil)

// NewSSOAccountRepository constructs an SSOAccountRepository.
func NewSSOAccountRepository(database *db.DB) *SSOAccountRepository {
	return &SSOAccountRepository{db: database}
}

// Create inserts a new linked SSO account record.
func (r *SSOAccountRepository) Create(ctx context.Context, acc *domain.UserSSOAccount) error {
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	now := time.Now()
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = now
	}
	if acc.UpdatedAt.IsZero() {
		acc.UpdatedAt = now
	}
	if acc.ProfileData == nil {
		acc.ProfileData = json.RawMessage("{}")
	}

	const q = `
		INSERT INTO user_sso_accounts (id, user_id, provider, provider_user_id, email, profile_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, q,
		acc.ID, acc.UserID, string(acc.Provider), acc.ProviderUserID,
		acc.Email, acc.ProfileData, acc.CreatedAt, acc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("sso_account_repo.Create: %w", err)
	}
	return nil
}

// GetByProviderAndUserID finds a linked SSO account by provider and external user ID.
func (r *SSOAccountRepository) GetByProviderAndUserID(ctx context.Context, provider domain.SSOProvider, providerUserID string) (*domain.UserSSOAccount, error) {
	const q = `
		SELECT id, user_id, provider, provider_user_id, email, profile_data, created_at, updated_at
		FROM user_sso_accounts
		WHERE provider = $1 AND provider_user_id = $2`

	row := r.db.QueryRowContext(ctx, q, string(provider), providerUserID)

	acc := &domain.UserSSOAccount{}
	var providerStr string
	err := row.Scan(
		&acc.ID, &acc.UserID, &providerStr, &acc.ProviderUserID,
		&acc.Email, &acc.ProfileData, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("sso_account_repo.GetByProviderAndUserID: %w", err)
	}
	acc.Provider = domain.SSOProvider(providerStr)
	return acc, nil
}
