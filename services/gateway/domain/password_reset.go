package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a password reset token in the database.
// The TokenHash field is marked with `json:"-"` to ensure the secret hash is never exposed.
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
