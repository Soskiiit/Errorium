package sso_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/sso"
)

func TestGitHubProvider_AuthURL(t *testing.T) {
	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID: "test-gh-client",
	})

	redirectURI := "http://localhost:8000/api/v1/auth/sso/callback/github"
	state := "state-12345"

	authURL := p.AuthURL(state, redirectURI)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse authURL: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != "test-gh-client" {
		t.Errorf("expected client_id test-gh-client, got %s", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != redirectURI {
		t.Errorf("expected redirect_uri %s, got %s", redirectURI, q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("expected response_type code, got %s", q.Get("response_type"))
	}
	if q.Get("state") != state {
		t.Errorf("expected state %s, got %s", state, q.Get("state"))
	}
	if !strings.Contains(q.Get("scope"), "read:user") || !strings.Contains(q.Get("scope"), "user:email") {
		t.Errorf("expected github scopes, got %s", q.Get("scope"))
	}
}

func TestGoogleProvider_AuthURL(t *testing.T) {
	p := sso.NewGoogleProvider(sso.ProviderConfig{
		ClientID: "test-google-client",
	})

	redirectURI := "http://localhost:8000/api/v1/auth/sso/callback/google"
	state := "state-google"

	authURL := p.AuthURL(state, redirectURI)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse authURL: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != "test-google-client" {
		t.Errorf("expected client_id test-google-client, got %s", q.Get("client_id"))
	}
	if !strings.Contains(q.Get("scope"), "openid") || !strings.Contains(q.Get("scope"), "email") {
		t.Errorf("expected google scopes, got %s", q.Get("scope"))
	}
}

func TestYandexProvider_AuthURL(t *testing.T) {
	p := sso.NewYandexProvider(sso.ProviderConfig{
		ClientID: "test-ya-client",
	})

	redirectURI := "http://localhost:8000/api/v1/auth/sso/callback/yandex"
	state := "state-yandex"

	authURL := p.AuthURL(state, redirectURI)
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse authURL: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != "test-ya-client" {
		t.Errorf("expected client_id test-ya-client, got %s", q.Get("client_id"))
	}
	if !strings.Contains(q.Get("scope"), "login:email") || !strings.Contains(q.Get("scope"), "login:info") {
		t.Errorf("expected yandex scopes, got %s", q.Get("scope"))
	}
}

func TestGitHubProvider_Exchange_Success(t *testing.T) {
	var receivedContentType, receivedAccept string
	var receivedBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedAccept = r.Header.Get("Accept")

		_ = r.ParseForm()
		receivedBody = r.PostForm.Encode()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "gh_secret_token",
			"token_type":   "bearer",
			"scope":        "read:user,user:email",
		})
	}))
	defer ts.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     ts.URL,
		HTTPClient:   ts.Client(),
	})

	token, err := p.Exchange(context.Background(), "auth_code_123", "http://redirect.test")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}
	if token != "gh_secret_token" {
		t.Errorf("expected token gh_secret_token, got %s", token)
	}
	if !strings.Contains(receivedContentType, "application/x-www-form-urlencoded") {
		t.Errorf("expected form Content-Type, got %s", receivedContentType)
	}
	if receivedAccept != "application/json" {
		t.Errorf("expected Accept application/json, got %s", receivedAccept)
	}
	if !strings.Contains(receivedBody, "grant_type=authorization_code") ||
		!strings.Contains(receivedBody, "code=auth_code_123") ||
		!strings.Contains(receivedBody, "client_id=gh-client") ||
		!strings.Contains(receivedBody, "client_secret=gh-secret") {
		t.Errorf("unexpected body: %s", receivedBody)
	}
}

func TestGitHubProvider_Exchange_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "bad_verification_code",
			"error_description": "The code passed is incorrect or expired.",
		})
	}))
	defer ts.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     ts.URL,
		HTTPClient:   ts.Client(),
	})

	_, err := p.Exchange(context.Background(), "bad_code", "http://redirect.test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGitHubProvider_UserInfo_DirectEmail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid_gh_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    123456,
			"login": "octocat",
			"name":  "The Octocat",
			"email": "octocat@github.com",
		})
	}))
	defer ts.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:    "gh-client",
		UserInfoURL: ts.URL,
		HTTPClient:  ts.Client(),
	})

	info, err := p.FetchUserInfo(context.Background(), "valid_gh_token")
	if err != nil {
		t.Fatalf("FetchUserInfo failed: %v", err)
	}
	if info.ProviderUserID != "123456" {
		t.Errorf("expected ProviderUserID 123456, got %s", info.ProviderUserID)
	}
	if info.Email != "octocat@github.com" {
		t.Errorf("expected Email octocat@github.com, got %s", info.Email)
	}
	if info.Username != "octocat" {
		t.Errorf("expected Username octocat, got %s", info.Username)
	}
}

func TestGitHubProvider_UserInfo_FallbackEmails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    999,
			"login": "octononame",
			"name":  "No Name",
			"email": nil,
		})
	})
	mux.HandleFunc("/user/emails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "unverified@example.com", "primary": false, "verified": false},
			{"email": "primary@example.com", "primary": true, "verified": true},
			{"email": "secondary@example.com", "primary": false, "verified": true},
		})
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:    "gh-client",
		UserInfoURL: ts.URL + "/user",
		EmailsURL:   ts.URL + "/user/emails",
		HTTPClient:  ts.Client(),
	})

	info, err := p.FetchUserInfo(context.Background(), "valid_gh_token")
	if err != nil {
		t.Fatalf("FetchUserInfo failed: %v", err)
	}
	if info.ProviderUserID != "999" {
		t.Errorf("expected ProviderUserID 999, got %s", info.ProviderUserID)
	}
	if info.Email != "primary@example.com" {
		t.Errorf("expected Email primary@example.com, got %s", info.Email)
	}
	if info.Username != "octononame" {
		t.Errorf("expected Username octononame, got %s", info.Username)
	}
}

func TestGoogleProvider_UserInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "goog-sub-987",
			"email":          "goog.user@example.com",
			"email_verified": true,
			"name":           "Google User",
		})
	}))
	defer ts.Close()

	p := sso.NewGoogleProvider(sso.ProviderConfig{
		ClientID:    "goog-client",
		UserInfoURL: ts.URL,
		HTTPClient:  ts.Client(),
	})

	info, err := p.FetchUserInfo(context.Background(), "goog_token")
	if err != nil {
		t.Fatalf("FetchUserInfo failed: %v", err)
	}
	if info.ProviderUserID != "goog-sub-987" {
		t.Errorf("expected ProviderUserID goog-sub-987, got %s", info.ProviderUserID)
	}
	if info.Email != "goog.user@example.com" {
		t.Errorf("expected Email goog.user@example.com, got %s", info.Email)
	}
	if info.Username != "Google User" {
		t.Errorf("expected Username Google User, got %s", info.Username)
	}
}

func TestYandexProvider_UserInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            "yandex-id-555",
			"default_login": "ya_login",
			"default_email": "ya.user@yandex.ru",
			"real_name":     "Ivan Ivanov",
		})
	}))
	defer ts.Close()

	p := sso.NewYandexProvider(sso.ProviderConfig{
		ClientID:    "ya-client",
		UserInfoURL: ts.URL,
		HTTPClient:  ts.Client(),
	})

	info, err := p.FetchUserInfo(context.Background(), "ya_token")
	if err != nil {
		t.Fatalf("FetchUserInfo failed: %v", err)
	}
	if info.ProviderUserID != "yandex-id-555" {
		t.Errorf("expected ProviderUserID yandex-id-555, got %s", info.ProviderUserID)
	}
	if info.Email != "ya.user@yandex.ru" {
		t.Errorf("expected Email ya.user@yandex.ru, got %s", info.Email)
	}
	if info.Username != "ya_login" {
		t.Errorf("expected Username ya_login, got %s", info.Username)
	}
}

func TestRegistry(t *testing.T) {
	reg := sso.NewRegistry()
	gh := sso.NewGitHubProvider(sso.ProviderConfig{ClientID: "gh"})
	reg.Register(gh)

	p, ok := reg.Get(domain.SSOProviderGitHub)
	if !ok || p == nil {
		t.Fatalf("expected github provider to be registered")
	}

	_, ok = reg.Get(domain.SSOProvider("unknown"))
	if ok {
		t.Fatalf("expected unknown provider to not be registered")
	}
}
