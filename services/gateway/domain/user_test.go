package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUser_JSONSerialization(t *testing.T) {
	pw := "super-secret-bcrypt-hash"
	avatar := "https://storage.errorium.org/avatars/test.png"
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

	u := &User{
		ID:           uuid.MustParse("01934272-3564-77e4-b7e3-77dd7fc7c13a"),
		Email:        "developer@example.com",
		Username:     "coder123",
		AvatarURL:    &avatar,
		PasswordHash: &pw,
		Role:         RoleUser,
		Profession:   "Backend developer",
		Gender:       "Not specified",
		Language:     "English",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("failed to marshal User: %v", err)
	}

	jsonStr := string(data)

	// Check password_hash is omitted
	if strings.Contains(jsonStr, "password_hash") || strings.Contains(jsonStr, pw) {
		t.Errorf("User JSON must not contain password_hash or password value: %s", jsonStr)
	}

	// Check profession, gender, language are present
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal User JSON: %v", err)
	}

	if val, ok := parsed["profession"]; !ok || val != "Backend developer" {
		t.Errorf("expected profession 'Backend developer', got %v", val)
	}
	if val, ok := parsed["gender"]; !ok || val != "Not specified" {
		t.Errorf("expected gender 'Not specified', got %v", val)
	}
	if val, ok := parsed["language"]; !ok || val != "English" {
		t.Errorf("expected language 'English', got %v", val)
	}
	if val, ok := parsed["email"]; !ok || val != "developer@example.com" {
		t.Errorf("expected email 'developer@example.com', got %v", val)
	}
	if val, ok := parsed["username"]; !ok || val != "coder123" {
		t.Errorf("expected username 'coder123', got %v", val)
	}
	if val, ok := parsed["role"]; !ok || val != "user" {
		t.Errorf("expected role 'user', got %v", val)
	}
}
