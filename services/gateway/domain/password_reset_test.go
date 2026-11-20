package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
)

func TestPasswordResetToken_JSONSerialization(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	expires := now.Add(1 * time.Hour)
	used := now.Add(10 * time.Minute)
	secretHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	token := &domain.PasswordResetToken{
		ID:        uuid.MustParse("019348b0-0000-7000-8000-000000000001"),
		UserID:    uuid.MustParse("019348b0-0000-7000-8000-000000000002"),
		TokenHash: secretHash,
		ExpiresAt: expires,
		UsedAt:    &used,
		CreatedAt: now,
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("failed to marshal PasswordResetToken: %v", err)
	}

	jsonStr := string(data)

	// TokenHash MUST NOT be serialized into JSON
	if strings.Contains(jsonStr, secretHash) || strings.Contains(jsonStr, "token_hash") {
		t.Errorf("PasswordResetToken JSON must not contain token_hash or its secret value: %s", jsonStr)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if val, ok := parsed["id"]; !ok || val != token.ID.String() {
		t.Errorf("expected id %q, got %v", token.ID.String(), val)
	}
	if val, ok := parsed["user_id"]; !ok || val != token.UserID.String() {
		t.Errorf("expected user_id %q, got %v", token.UserID.String(), val)
	}
	if _, ok := parsed["expires_at"]; !ok {
		t.Errorf("expected expires_at in JSON")
	}
	if _, ok := parsed["used_at"]; !ok {
		t.Errorf("expected used_at in JSON")
	}
	if _, ok := parsed["created_at"]; !ok {
		t.Errorf("expected created_at in JSON")
	}

	// Test when UsedAt is nil, used_at is omitted
	tokenWithoutUsed := &domain.PasswordResetToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: secretHash,
		ExpiresAt: expires,
		UsedAt:    nil,
		CreatedAt: now,
	}
	dataWithoutUsed, err := json.Marshal(tokenWithoutUsed)
	if err != nil {
		t.Fatalf("failed to marshal token without used_at: %v", err)
	}
	if strings.Contains(string(dataWithoutUsed), "used_at") {
		t.Errorf("used_at should be omitted when nil: %s", string(dataWithoutUsed))
	}
}
