package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// ChatRepository provides CRUD operations for chats and chat_messages.
type ChatRepository struct {
	db *db.DB
}

// NewChatRepository constructs a ChatRepository.
func NewChatRepository(database *db.DB) *ChatRepository {
	return &ChatRepository{db: database}
}

// Create inserts a new chat record.
func (r *ChatRepository) Create(ctx context.Context, c *domain.Chat) error {
	const q = `
		INSERT INTO chats (id, user_id, title, is_pinned, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, q,
		c.ID, c.UserID, c.Title, c.IsPinned, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("chat_repo.Create: %w", err)
	}
	return nil
}

// ListByUser returns paginated chats for a user with an optional title search.
// Returns the slice, the total count matching the filter, and any error.
func (r *ChatRepository) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
	search string,
) ([]*domain.Chat, int, error) {
	const countQ = `
		SELECT COUNT(*)
		FROM chats
		WHERE user_id = $1
		  AND ($2 = '' OR title ILIKE '%' || $2 || '%')`

	var total int
	if err := r.db.QueryRowContext(ctx, countQ, userID, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("chat_repo.ListByUser count: %w", err)
	}

	const q = `
		SELECT id, user_id, title, is_pinned, created_at, updated_at
		FROM chats
		WHERE user_id = $1
		  AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		ORDER BY is_pinned DESC, updated_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, q, userID, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("chat_repo.ListByUser query: %w", err)
	}
	defer rows.Close()

	var chats []*domain.Chat
	for rows.Next() {
		c := &domain.Chat{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.IsPinned, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("chat_repo.ListByUser scan: %w", err)
		}
		chats = append(chats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("chat_repo.ListByUser rows: %w", err)
	}

	return chats, total, nil
}

// GetByID fetches a single chat by its primary key.
func (r *ChatRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	const q = `
		SELECT id, user_id, title, is_pinned, created_at, updated_at
		FROM chats
		WHERE id = $1`

	c := &domain.Chat{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&c.ID, &c.UserID, &c.Title, &c.IsPinned, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("chat_repo.GetByID: %w", err)
	}
	return c, nil
}

// GetByIDForUser fetches a single chat by its primary key and user ID.
// If the chat does not exist or does not belong to the user, an error (wrapping sql.ErrNoRows) is returned.
func (r *ChatRepository) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error) {
	const q = `
		SELECT id, user_id, title, is_pinned, created_at, updated_at
		FROM chats
		WHERE id = $1 AND user_id = $2`

	c := &domain.Chat{}
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(
		&c.ID, &c.UserID, &c.Title, &c.IsPinned, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("chat_repo.GetByIDForUser: %w", err)
	}
	return c, nil
}

// Update modifies title, is_pinned, and updated_at of an existing chat.
func (r *ChatRepository) Update(ctx context.Context, c *domain.Chat) error {
	const q = `
		UPDATE chats
		SET title = $1, is_pinned = $2, updated_at = $3
		WHERE id = $4`

	_, err := r.db.ExecContext(ctx, q, c.Title, c.IsPinned, c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("chat_repo.Update: %w", err)
	}
	return nil
}

// Delete removes a chat and cascades to its messages.
func (r *ChatRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM chats WHERE id = $1`

	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("chat_repo.Delete: %w", err)
	}
	return nil
}

// AddMessage inserts a new message into the given chat.
func (r *ChatRepository) AddMessage(ctx context.Context, m *domain.ChatMessage) error {
	const q = `
		INSERT INTO chat_messages
			(id, chat_id, sender_role, message_type, content, code, error_context, rag_sources, token_usage, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, q,
		m.ID, m.ChatID, string(m.SenderRole), string(m.MessageType),
		m.Content, m.Code,
		m.ErrorContext, m.RAGSources, m.TokenUsage, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("chat_repo.AddMessage: %w", err)
	}
	return nil
}

// GetMessages returns all messages for the given chat ordered by creation time ascending.
func (r *ChatRepository) GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error) {
	const q = `
		SELECT id, chat_id, sender_role, message_type, content, code, error_context, rag_sources, token_usage, created_at
		FROM chat_messages
		WHERE chat_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, chatID)
	if err != nil {
		return nil, fmt.Errorf("chat_repo.GetMessages query: %w", err)
	}
	defer rows.Close()

	var messages []*domain.ChatMessage
	for rows.Next() {
		m := &domain.ChatMessage{}
		var senderRole, messageType string
		if err := rows.Scan(
			&m.ID, &m.ChatID, &senderRole, &messageType,
			&m.Content, &m.Code,
			&m.ErrorContext, &m.RAGSources, &m.TokenUsage, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("chat_repo.GetMessages scan: %w", err)
		}
		m.SenderRole = domain.SenderRole(senderRole)
		m.MessageType = domain.MessageType(messageType)
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("chat_repo.GetMessages rows: %w", err)
	}

	return messages, nil
}
