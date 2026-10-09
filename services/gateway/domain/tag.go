package domain

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ChatMessageTag struct {
	ChatMessageID uuid.UUID `json:"chat_message_id" db:"chat_message_id"`
	TagID         uuid.UUID `json:"tag_id" db:"tag_id"`
}
