package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// TagRepository handles tag lookups and message-tag associations.
type TagRepository struct {
	db *db.DB
}

// NewTagRepository constructs a TagRepository.
func NewTagRepository(database *db.DB) *TagRepository {
	return &TagRepository{db: database}
}

// GetOrCreate upserts a tag by name and returns the resulting record.
func (r *TagRepository) GetOrCreate(ctx context.Context, name string) (*domain.Tag, error) {
	const q = `
		INSERT INTO tags (id, name, created_at)
		VALUES (gen_random_uuid(), $1, NOW())
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name, created_at`

	t := &domain.Tag{}
	err := r.db.QueryRowContext(ctx, q, name).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("tag_repo.GetOrCreate: %w", err)
	}
	return t, nil
}

// AttachToMessage upserts rows in chat_message_tags for each tag name.
// Tags are created via GetOrCreate first.
func (r *TagRepository) AttachToMessage(ctx context.Context, chatMessageID uuid.UUID, tagNames []string) error {
	for _, name := range tagNames {
		tag, err := r.GetOrCreate(ctx, name)
		if err != nil {
			return fmt.Errorf("tag_repo.AttachToMessage get_or_create %q: %w", name, err)
		}

		const q = `
			INSERT INTO chat_message_tags (chat_message_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`

		if _, err := r.db.ExecContext(ctx, q, chatMessageID, tag.ID); err != nil {
			return fmt.Errorf("tag_repo.AttachToMessage insert %q: %w", name, err)
		}
	}
	return nil
}

// GetByMessage returns all tags associated with the given chat message.
func (r *TagRepository) GetByMessage(ctx context.Context, chatMessageID uuid.UUID) ([]*domain.Tag, error) {
	const q = `
		SELECT t.id, t.name, t.created_at
		FROM tags t
		JOIN chat_message_tags cmt ON cmt.tag_id = t.id
		WHERE cmt.chat_message_id = $1
		ORDER BY t.name ASC`

	rows, err := r.db.QueryContext(ctx, q, chatMessageID)
	if err != nil {
		return nil, fmt.Errorf("tag_repo.GetByMessage query: %w", err)
	}
	defer rows.Close()

	var tags []*domain.Tag
	for rows.Next() {
		t := &domain.Tag{}
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("tag_repo.GetByMessage scan: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tag_repo.GetByMessage rows: %w", err)
	}

	return tags, nil
}
