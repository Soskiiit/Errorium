package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// AuditRepository provides operations on the audit_logs table.
type AuditRepository struct {
	db *db.DB
}

var _ AuditStore = (*AuditRepository)(nil)

// NewAuditRepository constructs an AuditRepository.
func NewAuditRepository(database *db.DB) *AuditRepository {
	return &AuditRepository{db: database}
}

// Create inserts a new audit log record.
func (r *AuditRepository) Create(ctx context.Context, l *domain.AuditLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	if len(l.Details) == 0 {
		l.Details = json.RawMessage("{}")
	}

	var ipStr *string
	if l.ClientIP != nil {
		s := l.ClientIP.String()
		ipStr = &s
	}

	const q = `
		INSERT INTO audit_logs (id, user_id, action, client_ip, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		l.ID, l.UserID, l.Action, ipStr, []byte(l.Details), l.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("audit_repo.Create: %w", err)
	}
	return nil
}

// Log is a convenience helper to insert an audit record.
func (r *AuditRepository) Log(ctx context.Context, userID *uuid.UUID, action string, clientIP *netip.Addr, details json.RawMessage) error {
	return r.Create(ctx, &domain.AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		ClientIP:  clientIP,
		Details:   details,
		CreatedAt: time.Now(),
	})
}
