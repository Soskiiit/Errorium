// Package repository defines storage interfaces for the Gateway service.
// Concrete implementations (backed by PostgreSQL) are provided by db-layer.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
)

// UserStore is the interface auth-layer uses to persist and query users.
// The concrete *UserRepository struct from internal/repository/user_repo.go
// must satisfy this interface.
type UserStore interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateProfile(ctx context.Context, u *domain.User) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error
}

// SessionStore is the interface auth-layer uses to manage refresh-token sessions.
type SessionStore interface {
	Create(ctx context.Context, s *domain.UserSession) error
	GetByTokenHash(ctx context.Context, hash string) (*domain.UserSession, error)
	RevokeByTokenHash(ctx context.Context, hash string) error
	CleanupExpired(ctx context.Context) error
}

// ChatStore abstracts persistence for Chat and ChatMessage entities.
type ChatStore interface {
	Create(ctx context.Context, c *domain.Chat) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, search string) ([]*domain.Chat, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error)
	GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error)
	Update(ctx context.Context, c *domain.Chat) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMessage(ctx context.Context, msg *domain.ChatMessage) error
	GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error)
}

// TagStore abstracts persistence for Tag entities.
type TagStore interface {
	GetOrCreate(ctx context.Context, name string) (*domain.Tag, error)
	AttachToMessage(ctx context.Context, chatMessageID uuid.UUID, tagNames []string) error
	GetByMessage(ctx context.Context, chatMessageID uuid.UUID) ([]*domain.Tag, error)
}
