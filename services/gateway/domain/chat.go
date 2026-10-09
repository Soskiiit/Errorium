package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	IsPinned  bool      `json:"is_pinned" db:"is_pinned"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type SenderRole string

const (
	SenderRoleUser      SenderRole = "user"
	SenderRoleAssistant SenderRole = "assistant"
	SenderRoleSystem    SenderRole = "system"
)

type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeErrorReport MessageType = "error_report"
	MessageTypeRAGSolution MessageType = "rag_solution"
)

type ErrorContext struct {
	Language   string   `json:"language,omitempty"`
	StackTrace string   `json:"stack_trace,omitempty"`
	ErrorType  string   `json:"error_type,omitempty"`
	RawQuery   string   `json:"raw_query,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type RAGSource struct {
	DocID      uint32  `json:"doc_id"`
	Title      string  `json:"title"`
	Score      float64 `json:"score"`
	URL        string  `json:"url"`
	Snippet    string  `json:"snippet,omitempty"`
	IsAccepted bool    `json:"is_accepted"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatMessage struct {
	ID           uuid.UUID        `json:"id" db:"id"`
	ChatID       uuid.UUID        `json:"chat_id" db:"chat_id"`
	SenderRole   SenderRole       `json:"sender_role" db:"sender_role"`
	MessageType  MessageType      `json:"message_type" db:"message_type"`
	ErrorContext *json.RawMessage `json:"error_context,omitempty" db:"error_context"`
	RAGSources   *json.RawMessage `json:"rag_sources,omitempty" db:"rag_sources"`
	TokenUsage   *json.RawMessage `json:"token_usage,omitempty" db:"token_usage"`
	CreatedAt    time.Time        `json:"created_at" db:"created_at"`

	Tags []Tag `json:"tags,omitempty" db:"-"`
}
