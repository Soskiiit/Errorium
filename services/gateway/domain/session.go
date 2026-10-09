package domain

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	UserID           uuid.UUID   `json:"user_id" db:"user_id"`
	RefreshTokenHash string      `json:"-" db:"refresh_token_hash"`
	UserAgent        *string     `json:"user_agent,omitempty" db:"user_agent"`
	ClientIP         *netip.Addr `json:"client_ip,omitempty" db:"client_ip"`
	IsRevoked        bool        `json:"is_revoked" db:"is_revoked"`
	ExpiresAt        time.Time   `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	LastUsedAt       time.Time   `json:"last_used_at" db:"last_used_at"`
}
