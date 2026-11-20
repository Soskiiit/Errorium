package domain

import (
	"time"
)

// SSOAuthState represents the state tracked during an OAuth2 authorization flow.
type SSOAuthState struct {
	State      string      `json:"state" db:"state"`
	Provider   SSOProvider `json:"provider" db:"provider"`
	RedirectTo *string     `json:"redirect_to,omitempty" db:"redirect_to"`
	ExpiresAt  time.Time   `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
}
