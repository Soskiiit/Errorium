package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/sso"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// ── In-memory Test Fakes ─────────────────────────────────────────────────────

type ssoFakeStateStore struct {
	mu     sync.Mutex
	states map[string]*domain.SSOAuthState
}

func newSsoFakeStateStore() *ssoFakeStateStore {
	return &ssoFakeStateStore{
		states: make(map[string]*domain.SSOAuthState),
	}
}

func (s *ssoFakeStateStore) Create(ctx context.Context, state *domain.SSOAuthState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *state
	s.states[state.State] = &cp
	return nil
}

func (s *ssoFakeStateStore) Get(ctx context.Context, stateStr string) (*domain.SSOAuthState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.states[stateStr]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *st
	return &cp, nil
}

func (s *ssoFakeStateStore) Delete(ctx context.Context, stateStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, stateStr)
	return nil
}

type ssoFakeAccountStore struct {
	mu       sync.Mutex
	accounts map[string]*domain.UserSSOAccount
}

func newSsoFakeAccountStore() *ssoFakeAccountStore {
	return &ssoFakeAccountStore{
		accounts: make(map[string]*domain.UserSSOAccount),
	}
}

func (s *ssoFakeAccountStore) key(provider domain.SSOProvider, providerUserID string) string {
	return string(provider) + ":" + providerUserID
}

func (s *ssoFakeAccountStore) Create(ctx context.Context, acc *domain.UserSSOAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(acc.Provider, acc.ProviderUserID)
	if _, exists := s.accounts[k]; exists {
		return errors.New("duplicate sso account")
	}
	cp := *acc
	s.accounts[k] = &cp
	return nil
}

func (s *ssoFakeAccountStore) GetByProviderAndUserID(ctx context.Context, provider domain.SSOProvider, providerUserID string) (*domain.UserSSOAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := s.key(provider, providerUserID)
	acc, ok := s.accounts[k]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *acc
	return &cp, nil
}

type ssoFakeUserStore struct {
	mu           sync.Mutex
	users        map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
}

func newSsoFakeUserStore() *ssoFakeUserStore {
	return &ssoFakeUserStore{
		users:        make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

func (f *ssoFakeUserStore) Create(ctx context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.usersByEmail[u.Email]; exists {
		return errors.New("duplicate email")
	}
	cp := *u
	f.users[u.ID] = &cp
	f.usersByEmail[u.Email] = &cp
	return nil
}

func (f *ssoFakeUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *u
	return &cp, nil
}

func (f *ssoFakeUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *u
	return &cp, nil
}

func (f *ssoFakeUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.users[u.ID]; ok {
		existing.Username = u.Username
		existing.Email = u.Email
	}
	return nil
}

func (f *ssoFakeUserStore) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return nil
}

func (f *ssoFakeUserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}

func (f *ssoFakeUserStore) List(ctx context.Context, limit, offset int, search string) ([]*domain.User, int, error) {
	return nil, 0, nil
}

func (f *ssoFakeUserStore) UpdateRoleAndStatus(ctx context.Context, id uuid.UUID, role *domain.Role, isActive *bool) (*domain.User, error) {
	return nil, nil
}

type ssoFakeSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*domain.UserSession
}

func newSsoFakeSessionStore() *ssoFakeSessionStore {
	return &ssoFakeSessionStore{
		sessions: make(map[string]*domain.UserSession),
	}
}

func (f *ssoFakeSessionStore) Create(ctx context.Context, s *domain.UserSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *s
	f.sessions[s.RefreshTokenHash] = &cp
	return nil
}

func (f *ssoFakeSessionStore) GetByTokenHash(ctx context.Context, hash string) (*domain.UserSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[hash]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *s
	return &cp, nil
}

func (f *ssoFakeSessionStore) RevokeByTokenHash(ctx context.Context, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sessions[hash]; ok {
		s.IsRevoked = true
	}
	return nil
}

func (f *ssoFakeSessionStore) RevokeByUser(ctx context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if s.UserID == userID {
			s.IsRevoked = true
		}
	}
	return nil
}

func (f *ssoFakeSessionStore) CleanupExpired(ctx context.Context) error {
	return nil
}

// ── Test Setup Helper ────────────────────────────────────────────────────────

type testRig struct {
	mux          *http.ServeMux
	states       *ssoFakeStateStore
	ssoAccounts  *ssoFakeAccountStore
	users        *ssoFakeUserStore
	sessions     *ssoFakeSessionStore
	registry     *sso.Registry
	jwtMgr       *jwtpkg.Manager
	redirectBase string
	frontendURL  string
}

func newTestRig() *testRig {
	states := newSsoFakeStateStore()
	ssoAccounts := newSsoFakeAccountStore()
	users := newSsoFakeUserStore()
	sessions := newSsoFakeSessionStore()
	registry := sso.NewRegistry()

	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:              "test-jwt-secret-key-32-chars-long!",
		AccessTokenExpireMin:   60,
		RefreshTokenExpireDays: 30,
	})

	redirectBase := "http://localhost:8000"
	frontendURL := "http://localhost:3000"

	h := handler.NewSSOHandler(
		registry,
		states,
		ssoAccounts,
		users,
		sessions,
		jwtMgr,
		redirectBase,
		frontendURL,
		30,
	)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	return &testRig{
		mux:          mux,
		states:       states,
		ssoAccounts:  ssoAccounts,
		users:        users,
		sessions:     sessions,
		registry:     registry,
		jwtMgr:       jwtMgr,
		redirectBase: redirectBase,
		frontendURL:  frontendURL,
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestSSOHandler_Login_Success(t *testing.T) {
	rig := newTestRig()

	// Register configured github provider
	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID: "github-client-id",
	})
	rig.registry.Register(p)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/login/github", nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header in 302 response")
	}

	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("failed to parse redirect Location: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != "github-client-id" {
		t.Errorf("expected client_id github-client-id, got %s", q.Get("client_id"))
	}
	expectedRedirectURI := "http://localhost:8000/api/v1/auth/sso/callback/github"
	if q.Get("redirect_uri") != expectedRedirectURI {
		t.Errorf("expected redirect_uri %s, got %s", expectedRedirectURI, q.Get("redirect_uri"))
	}
	state := q.Get("state")
	if state == "" {
		t.Fatal("expected state parameter in redirect URL")
	}

	// Verify state saved in store with valid TTL
	savedState, err := rig.states.Get(context.Background(), state)
	if err != nil {
		t.Fatalf("expected state %s in store: %v", state, err)
	}
	if savedState.Provider != domain.SSOProviderGitHub {
		t.Errorf("expected provider github, got %s", savedState.Provider)
	}
	if savedState.ExpiresAt.Before(time.Now()) {
		t.Errorf("expected state expires_at in future, got %v", savedState.ExpiresAt)
	}
}

func TestSSOHandler_Login_UnknownProvider(t *testing.T) {
	rig := newTestRig()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/login/unknown_idp", nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "unknown provider" {
		t.Errorf("expected error 'unknown provider', got %q", resp["error"])
	}
}

func TestSSOHandler_Login_ProviderNotConfigured(t *testing.T) {
	rig := newTestRig()

	// Provider registered with empty ClientID
	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID: "",
	})
	rig.registry.Register(p)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/login/github", nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "provider not configured" {
		t.Errorf("expected error 'provider not configured', got %q", resp["error"])
	}
}

func TestSSOHandler_Callback_NewUser_Success(t *testing.T) {
	rig := newTestRig()

	// Mock IdP server
	idpMux := http.NewServeMux()
	idpMux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock_idp_access_token",
			"token_type":   "bearer",
		})
	})
	idpMux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "idp_user_100",
			"login": "new_sso_user",
			"email": "new_sso@example.com",
		})
	})
	idpServer := httptest.NewServer(idpMux)
	defer idpServer.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     idpServer.URL + "/token",
		UserInfoURL:  idpServer.URL + "/user",
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	// Pre-create state
	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderGitHub,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=valid_code&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d: %s", rr.Code, rr.Body.String())
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header in 302 response")
	}

	// Must redirect to {FRONTEND_URL}/auth/callback#access_token=...&refresh_token=... (fragment #, not query)
	if !strings.HasPrefix(loc, "http://localhost:3000/auth/callback#") {
		t.Fatalf("expected prefix http://localhost:3000/auth/callback#, got %s", loc)
	}

	fragment := strings.TrimPrefix(loc, "http://localhost:3000/auth/callback#")
	fragValues, err := url.ParseQuery(fragment)
	if err != nil {
		t.Fatalf("failed to parse fragment: %v", err)
	}
	accessToken := fragValues.Get("access_token")
	refreshToken := fragValues.Get("refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("expected access_token and refresh_token in fragment, got %s", fragment)
	}

	// Verify user created in DB
	u, err := rig.users.GetByEmail(context.Background(), "new_sso@example.com")
	if err != nil {
		t.Fatalf("user was not created in users store: %v", err)
	}
	if u.Username != "new_sso_user" {
		t.Errorf("expected username new_sso_user, got %s", u.Username)
	}
	if u.PasswordHash != nil {
		t.Errorf("expected password_hash to be nil for SSO user, got %v", u.PasswordHash)
	}

	// Verify SSO account link
	ssoAcc, err := rig.ssoAccounts.GetByProviderAndUserID(context.Background(), domain.SSOProviderGitHub, "idp_user_100")
	if err != nil {
		t.Fatalf("sso account was not created in ssoAccounts store: %v", err)
	}
	if ssoAcc.UserID != u.ID {
		t.Errorf("expected linked user ID %s, got %s", u.ID, ssoAcc.UserID)
	}

	// Verify state was deleted (single-use)
	_, err = rig.states.Get(context.Background(), stateStr)
	if err == nil {
		t.Errorf("expected state %s to be deleted after use", stateStr)
	}
}

func TestSSOHandler_Callback_ExistingSSOAccount(t *testing.T) {
	rig := newTestRig()

	// Pre-create user and linked account
	existingUser := &domain.User{
		ID:        uuid.New(),
		Email:     "existing@example.com",
		Username:  "existing_guy",
		Role:      domain.RoleUser,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = rig.users.Create(context.Background(), existingUser)

	_ = rig.ssoAccounts.Create(context.Background(), &domain.UserSSOAccount{
		ID:             uuid.New(),
		UserID:         existingUser.ID,
		Provider:       domain.SSOProviderGoogle,
		ProviderUserID: "goog_sub_existing",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	idpMux := http.NewServeMux()
	idpMux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "goog_token"})
	})
	idpMux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":   "goog_sub_existing",
			"email": "different_email_now@example.com",
			"name":  "Existing Guy",
		})
	})
	idpServer := httptest.NewServer(idpMux)
	defer idpServer.Close()

	p := sso.NewGoogleProvider(sso.ProviderConfig{
		ClientID:     "goog-client",
		ClientSecret: "goog-secret",
		TokenURL:     idpServer.URL + "/token",
		UserInfoURL:  idpServer.URL + "/userinfo",
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderGoogle,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/google?code=code123&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}

	loc := rr.Header().Get("Location")
	fragment := strings.TrimPrefix(loc, "http://localhost:3000/auth/callback#")
	fragValues, _ := url.ParseQuery(fragment)
	accessToken := fragValues.Get("access_token")

	// Validate access token belongs to existingUser
	claims, err := rig.jwtMgr.ParseAccess(accessToken)
	if err != nil {
		t.Fatalf("failed to verify access token: %v", err)
	}
	if claims.UserID != existingUser.ID.String() {
		t.Errorf("expected user ID %s in token, got %s", existingUser.ID.String(), claims.UserID)
	}
}

func TestSSOHandler_Callback_LinkToExistingUserByEmail(t *testing.T) {
	rig := newTestRig()

	// Pre-create user with password (email already registered)
	existingUser := &domain.User{
		ID:        uuid.New(),
		Email:     "registered@errorium.ru",
		Username:  "registered_user",
		Role:      domain.RoleUser,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = rig.users.Create(context.Background(), existingUser)

	idpMux := http.NewServeMux()
	idpMux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ya_token"})
	})
	idpMux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            "yandex_new_id_888",
			"default_email": "registered@errorium.ru",
			"default_login": "ya_registered",
		})
	})
	idpServer := httptest.NewServer(idpMux)
	defer idpServer.Close()

	p := sso.NewYandexProvider(sso.ProviderConfig{
		ClientID:     "ya-client",
		ClientSecret: "ya-secret",
		TokenURL:     idpServer.URL + "/token",
		UserInfoURL:  idpServer.URL + "/info",
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderYandex,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/yandex?code=ya_code&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}

	// Verify SSO account was linked to existingUser
	ssoAcc, err := rig.ssoAccounts.GetByProviderAndUserID(context.Background(), domain.SSOProviderYandex, "yandex_new_id_888")
	if err != nil {
		t.Fatalf("failed to find linked sso account: %v", err)
	}
	if ssoAcc.UserID != existingUser.ID {
		t.Errorf("expected linked userID %s, got %s", existingUser.ID, ssoAcc.UserID)
	}
}

func TestSSOHandler_Callback_InvalidOrExpiredState(t *testing.T) {
	rig := newTestRig()

	p := sso.NewGitHubProvider(sso.ProviderConfig{ClientID: "gh-client"})
	rig.registry.Register(p)

	// Case 1: State not found
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/callback/github?code=abc&state=nonexistent", nil)
	rr1 := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for nonexistent state, got %d", rr1.Code)
	}

	// Case 2: State expired
	expiredState := "expired-state-xyz"
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     expiredState,
		Provider:  domain.SSOProviderGitHub,
		ExpiresAt: time.Now().Add(-1 * time.Minute), // expired
	})
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=abc&state=%s", expiredState), nil)
	rr2 := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired state, got %d", rr2.Code)
	}

	// Case 3: Mismatched provider
	mismatchedState := "mismatched-provider-state"
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     mismatchedState,
		Provider:  domain.SSOProviderGoogle, // Created for Google
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	// But used on GitHub callback
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=abc&state=%s", mismatchedState), nil)
	rr3 := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for mismatched provider state, got %d", rr3.Code)
	}
}

func TestSSOHandler_Callback_ExchangeFailed(t *testing.T) {
	rig := newTestRig()

	idpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer idpServer.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     idpServer.URL,
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderGitHub,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=bad_code&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on exchange failure, got %d", rr.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "sso exchange failed" {
		t.Errorf("expected 'sso exchange failed', got %q", resp["error"])
	}
}

func TestSSOHandler_Callback_DisabledUser(t *testing.T) {
	rig := newTestRig()

	disabledUser := &domain.User{
		ID:        uuid.New(),
		Email:     "banned@errorium.ru",
		Username:  "banned_user",
		Role:      domain.RoleUser,
		IsActive:  false, // disabled!
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = rig.users.Create(context.Background(), disabledUser)

	_ = rig.ssoAccounts.Create(context.Background(), &domain.UserSSOAccount{
		ID:             uuid.New(),
		UserID:         disabledUser.ID,
		Provider:       domain.SSOProviderGitHub,
		ProviderUserID: "banned_gh_id",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	idpMux := http.NewServeMux()
	idpMux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token"})
	})
	idpMux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "banned_gh_id",
			"login": "banned_user",
		})
	})
	idpServer := httptest.NewServer(idpMux)
	defer idpServer.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     idpServer.URL + "/token",
		UserInfoURL:  idpServer.URL + "/user",
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderGitHub,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=code&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled user, got %d", rr.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp["error"] != "account is disabled" {
		t.Errorf("expected 'account is disabled', got %q", resp["error"])
	}
}

func TestSSOHandler_Callback_NewUser_WithoutEmail(t *testing.T) {
	rig := newTestRig()

	idpMux := http.NewServeMux()
	idpMux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token"})
	})
	idpMux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "no_email_id_77",
			"login": "no_email_user",
			"email": nil,
		})
	})
	idpServer := httptest.NewServer(idpMux)
	defer idpServer.Close()

	p := sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "gh-client",
		ClientSecret: "gh-secret",
		TokenURL:     idpServer.URL + "/token",
		UserInfoURL:  idpServer.URL + "/user",
		EmailsURL:    "", // no emails endpoint
		HTTPClient:   idpServer.Client(),
	})
	rig.registry.Register(p)

	stateStr := uuid.New().String()
	_ = rig.states.Create(context.Background(), &domain.SSOAuthState{
		State:     stateStr,
		Provider:  domain.SSOProviderGitHub,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/auth/sso/callback/github?code=code&state=%s", stateStr), nil)
	rr := httptest.NewRecorder()
	rig.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}

	expectedFallbackEmail := "github_no_email_id_77@sso.errorium.local"
	u, err := rig.users.GetByEmail(context.Background(), expectedFallbackEmail)
	if err != nil {
		t.Fatalf("expected user with fallback email %s, got error: %v", expectedFallbackEmail, err)
	}
	if u.Username != "no_email_user" {
		t.Errorf("expected username no_email_user, got %s", u.Username)
	}
}
