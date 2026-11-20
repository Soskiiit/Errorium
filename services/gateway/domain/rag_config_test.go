package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
)

func TestRagConfig_JSONSerialization(t *testing.T) {
	baseURL := "https://api.openai.com/v1"
	cfg := &domain.RagConfig{
		ID:           1,
		Provider:     domain.LLMProviderOpenAI,
		BaseURL:      &baseURL,
		Model:        "gpt-4o-mini",
		SystemPrompt: "Ты Errorium...",
		Temperature:  0.7,
		MaxTokens:    2000,
		UpdatedAt:    time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal RagConfig: %v", err)
	}

	jsonStr := string(data)

	// ID and UpdatedAt must NOT be serialized into JSON
	if strings.Contains(jsonStr, "\"id\"") {
		t.Errorf("RagConfig JSON must not contain id: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "updated_at") {
		t.Errorf("RagConfig JSON must not contain updated_at: %s", jsonStr)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if val, ok := parsed["provider"]; !ok || val != "openai" {
		t.Errorf("expected provider openai, got %v", val)
	}
	if val, ok := parsed["base_url"]; !ok || val != baseURL {
		t.Errorf("expected base_url %q, got %v", baseURL, val)
	}
	if val, ok := parsed["model"]; !ok || val != "gpt-4o-mini" {
		t.Errorf("expected model gpt-4o-mini, got %v", val)
	}
	if val, ok := parsed["system_prompt"]; !ok || val != "Ты Errorium..." {
		t.Errorf("expected system_prompt, got %v", val)
	}
	if val, ok := parsed["temperature"]; !ok || val != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", val)
	}
	if val, ok := parsed["max_tokens"]; !ok || val != float64(2000) {
		t.Errorf("expected max_tokens 2000, got %v", val)
	}

	// Test Unmarshal from JSON (client request format)
	reqJSON := `{"provider":"gigachat","model":"GigaChat-Pro","system_prompt":"Промпт","temperature":0.3,"max_tokens":1000}`
	var unmarshaled domain.RagConfig
	if err := json.Unmarshal([]byte(reqJSON), &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal request JSON: %v", err)
	}
	if unmarshaled.Provider != domain.LLMProviderGigaChat {
		t.Errorf("expected provider gigachat, got %v", unmarshaled.Provider)
	}
	if unmarshaled.BaseURL != nil {
		t.Errorf("expected nil base_url, got %v", unmarshaled.BaseURL)
	}
	if unmarshaled.Model != "GigaChat-Pro" {
		t.Errorf("expected model GigaChat-Pro, got %v", unmarshaled.Model)
	}
	if unmarshaled.Temperature != 0.3 {
		t.Errorf("expected temperature 0.3, got %v", unmarshaled.Temperature)
	}
	if unmarshaled.MaxTokens != 1000 {
		t.Errorf("expected max_tokens 1000, got %v", unmarshaled.MaxTokens)
	}
}
