package repository

import (
	"context"
	"fmt"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// RagConfigRepository provides operations on the rag_config singleton table.
type RagConfigRepository struct {
	db *db.DB
}

var _ RagConfigStore = (*RagConfigRepository)(nil)

// NewRagConfigRepository constructs a RagConfigRepository.
func NewRagConfigRepository(database *db.DB) *RagConfigRepository {
	return &RagConfigRepository{db: database}
}

// Get retrieves the singleton RAG configuration (id=1).
func (r *RagConfigRepository) Get(ctx context.Context) (*domain.RagConfig, error) {
	const q = `
		SELECT id, provider, base_url, model, system_prompt, temperature, max_tokens, updated_at
		FROM rag_config
		WHERE id = 1`

	row := r.db.QueryRowContext(ctx, q)

	cfg := &domain.RagConfig{}
	var providerStr string
	err := row.Scan(
		&cfg.ID, &providerStr, &cfg.BaseURL, &cfg.Model,
		&cfg.SystemPrompt, &cfg.Temperature, &cfg.MaxTokens, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("rag_config_repo.Get: %w", err)
	}
	cfg.Provider = domain.LLMProvider(providerStr)
	return cfg, nil
}

// Update updates the singleton RAG configuration (id=1).
func (r *RagConfigRepository) Update(ctx context.Context, cfg *domain.RagConfig) error {
	const q = `
		UPDATE rag_config
		SET provider = $1, base_url = $2, model = $3, system_prompt = $4, temperature = $5, max_tokens = $6, updated_at = NOW()
		WHERE id = 1`

	_, err := r.db.ExecContext(ctx, q,
		string(cfg.Provider), cfg.BaseURL, cfg.Model,
		cfg.SystemPrompt, cfg.Temperature, cfg.MaxTokens,
	)
	if err != nil {
		return fmt.Errorf("rag_config_repo.Update: %w", err)
	}
	cfg.ID = 1
	return nil
}
