package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
)

func TestSSOAuthState_JSONSerialization(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	expires := now.Add(15 * time.Minute)
	redirect := "http://localhost:3000/profile"

	state := &domain.SSOAuthState{
		State:      "random-oauth-state-string-12345",
		Provider:   domain.SSOProviderGitHub,
		RedirectTo: &redirect,
		ExpiresAt:  expires,
		CreatedAt:  now,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal SSOAuthState: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if val, ok := parsed["state"]; !ok || val != "random-oauth-state-string-12345" {
		t.Errorf("expected state, got %v", val)
	}
	if val, ok := parsed["provider"]; !ok || val != "github" {
		t.Errorf("expected provider github, got %v", val)
	}
	if val, ok := parsed["redirect_to"]; !ok || val != redirect {
		t.Errorf("expected redirect_to %q, got %v", redirect, val)
	}

	// Test nil redirect_to is omitted
	stateNilRedirect := &domain.SSOAuthState{
		State:     "random-state",
		Provider:  domain.SSOProviderGoogle,
		ExpiresAt: expires,
		CreatedAt: now,
	}
	dataNil, err := json.Marshal(stateNilRedirect)
	if err != nil {
		t.Fatalf("failed to marshal SSOAuthState with nil redirect: %v", err)
	}
	if strings.Contains(string(dataNil), "redirect_to") {
		t.Errorf("redirect_to should be omitted when nil: %s", string(dataNil))
	}
}
