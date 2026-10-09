package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SSOProvider string

const (
	SSOProviderGoogle SSOProvider = "google"
	SSOProviderYandex SSOProvider = "yandex"
	SSOProviderGitHub SSOProvider = "github"
)

type UserSSOAccount struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	Provider       SSOProvider     `json:"provider" db:"provider"`
	ProviderUserID string          `json:"provider_user_id" db:"provider_user_id"`
	Email          *string         `json:"email,omitempty" db:"email"`
	ProfileData    json.RawMessage `json:"profile_data" db:"profile_data"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
}
