package domain

import (
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    *uuid.UUID      `json:"user_id,omitempty" db:"user_id"`
	Action    string          `json:"action" db:"action"`
	ClientIP  *netip.Addr     `json:"client_ip,omitempty" db:"client_ip"`
	Details   json.RawMessage `json:"details" db:"details"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}
