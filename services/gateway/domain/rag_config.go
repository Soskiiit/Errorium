package domain

import (
	"time"
)

// LLMProvider defines the supported RAG/LLM backend providers.
type LLMProvider string

const (
	LLMProviderOpenAI   LLMProvider = "openai"
	LLMProviderGigaChat LLMProvider = "gigachat"
	LLMProviderStub     LLMProvider = "stub"
)

// RagConfig represents the RAG/LLM configuration stored in the database.
// Field names and types match the OpenAPI RagConfig schema.
// Internal fields ID and UpdatedAt are hidden from client JSON serialization (`json:"-"`).
type RagConfig struct {
	ID           int         `json:"-" db:"id"`
	Provider     LLMProvider `json:"provider" db:"provider"`
	BaseURL      *string     `json:"base_url,omitempty" db:"base_url"`
	Model        string      `json:"model" db:"model"`
	SystemPrompt string      `json:"system_prompt" db:"system_prompt"`
	Temperature  float64     `json:"temperature" db:"temperature"`
	MaxTokens    int         `json:"max_tokens" db:"max_tokens"`
	UpdatedAt    time.Time   `json:"-" db:"updated_at"`
}
