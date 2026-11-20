package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// ── In-memory Fake Stores ─────────────────────────────────────────────────────

type fakeUserStore struct {
	mu           sync.Mutex
	users        map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
	rehashedIDs  map[uuid.UUID]string
	lastLogin    map[uuid.UUID]time.Time
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users:        make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
		rehashedIDs:  make(map[uuid.UUID]string),
		lastLogin:    make(map[uuid.UUID]time.Time),
	}
}

func (f *fakeUserStore) Create(ctx context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.usersByEmail[u.Email]; exists {
		return errors.New("pq: duplicate key value violates unique constraint \"users_email_key\"")
	}
	cp := *u
	f.users[u.ID] = &cp
	f.usersByEmail[u.Email] = &cp
	return nil
}

func (f *fakeUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.users[u.ID]; ok {
		*existing = *u
		f.usersByEmail[u.Email] = existing
	}
	return nil
}

func (f *fakeUserStore) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[id]; ok {
		u.PasswordHash = &hash
		f.rehashedIDs[id] = hash
	}
	return nil
}

func (f *fakeUserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastLogin[id] = t
	if u, ok := f.users[id]; ok {
		u.LastLoginAt = &t
	}
	return nil
}

type fakeSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*domain.UserSession // tokenHash -> session
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions: make(map[string]*domain.UserSession),
	}
}

func (f *fakeSessionStore) Create(ctx context.Context, s *domain.UserSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *s
	f.sessions[s.RefreshTokenHash] = &cp
	return nil
}

func (f *fakeSessionStore) GetByTokenHash(ctx context.Context, hash string) (*domain.UserSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[hash]
	if !ok {
		return nil, errors.New("session not found")
	}
	cp := *s
	return &cp, nil
}

func (f *fakeSessionStore) RevokeByTokenHash(ctx context.Context, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.sessions[hash]; ok {
		s.IsRevoked = true
	}
	return nil
}

func (f *fakeSessionStore) RevokeByUser(ctx context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if s.UserID == userID {
			s.IsRevoked = true
		}
	}
	return nil
}

func (f *fakeSessionStore) CleanupExpired(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for k, s := range f.sessions {
		if now.After(s.ExpiresAt) {
			delete(f.sessions, k)
		}
	}
	return nil
}

// ── Test Harness Setup ────────────────────────────────────────────────────────

type testHarness struct {
	users    *fakeUserStore
	sessions *fakeSessionStore
	jwtMgr   *jwtpkg.Manager
	handler  *AuthHandler
	mux      *http.ServeMux
}

func newTestHarness(t *testing.T) *testHarness {
	users := newFakeUserStore()
	sessions := newFakeSessionStore()
	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:              "super-secret-test-key-32-bytes-long!",
		AccessTokenExpireMin:   15,
		RefreshTokenExpireDays: 30,
	})
	handler := NewAuthHandler(users, sessions, jwtMgr, 30)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return &testHarness{
		users:    users,
		sessions: sessions,
		jwtMgr:   jwtMgr,
		handler:  handler,
		mux:      mux,
	}
}

// ── Register Tests ────────────────────────────────────────────────────────────

func TestAuthRegister_Success(t *testing.T) {
	h := newTestHarness(t)

	body := `{"username":"alice","email":"alice@example.com","password":"securepassword123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var res tokenPair
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Errorf("missing tokens in response")
	}

	if res.User == nil {
		t.Fatalf("missing user in response")
	}
	if res.User.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", res.User.Username)
	}
	if res.User.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", res.User.Email)
	}
	if res.User.Profession != "Backend developer" {
		t.Errorf("expected default profession 'Backend developer', got %q", res.User.Profession)
	}
	if res.User.Gender != "Not specified" {
		t.Errorf("expected default gender 'Not specified', got %q", res.User.Gender)
	}
	if res.User.Language != "English" {
		t.Errorf("expected default language 'English', got %q", res.User.Language)
	}
	if !res.User.IsActive {
		t.Errorf("expected user to be active")
	}
}

func TestAuthRegister_128CharPassword(t *testing.T) {
	h := newTestHarness(t)

	password128 := strings.Repeat("A", 128)
	body := fmt.Sprintf(`{"username":"longpassuser","email":"longpass@example.com","password":"%s"}`, password128)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for 128-char password, got %d: %s", rec.Code, rec.Body.String())
	}

	// Now verify subsequent login succeeds with the 128-char password
	loginBody := fmt.Sprintf(`{"email":"longpass@example.com","password":"%s"}`, password128)
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(loginBody))
	loginRec := httptest.NewRecorder()

	h.mux.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login with 128-char password, got %d: %s", loginRec.Code, loginRec.Body.String())
	}

	var res tokenPair
	if err := json.Unmarshal(loginRec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Errorf("missing tokens in login response")
	}
}

func TestAuthRegister_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{"empty username", `{"username":"","email":"user@example.com","password":"password123"}`},
		{"whitespace username", `{"username":"   ","email":"user@example.com","password":"password123"}`},
		{"username > 100", fmt.Sprintf(`{"username":"%s","email":"user@example.com","password":"password123"}`, strings.Repeat("a", 101))},
		{"invalid email missing at", `{"username":"user","email":"userexample.com","password":"password123"}`},
		{"invalid email missing domain", `{"username":"user","email":"user@","password":"password123"}`},
		{"password < 8", `{"username":"user","email":"user@example.com","password":"1234567"}`},
		{"password > 128", fmt.Sprintf(`{"username":"user","email":"user@example.com","password":"%s"}`, strings.Repeat("p", 129))},
		{"malformed JSON", `{"username":`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHarness(t)
			req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthRegister_EmailConflict(t *testing.T) {
	h := newTestHarness(t)

	body := `{"username":"user1","email":"conflict@example.com","password":"password123"}`
	req1 := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
	rec1 := httptest.NewRecorder()
	h.mux.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first register failed: %d", rec1.Code)
	}

	body2 := `{"username":"user2","email":"conflict@example.com","password":"password123"}`
	req2 := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body2))
	rec2 := httptest.NewRecorder()
	h.mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var errResp map[string]string
	_ = json.Unmarshal(rec2.Body.Bytes(), &errResp)
	if errResp["error"] != "email already registered" {
		t.Errorf("expected 'email already registered', got %q", errResp["error"])
	}
}

// ── Login Tests ───────────────────────────────────────────────────────────────

func TestAuthLogin_Success(t *testing.T) {
	h := newTestHarness(t)

	hash, _ := hashPassword("mysecret123")
	u := &domain.User{
		ID:           uuid.New(),
		Email:        "bob@example.com",
		Username:     "bob",
		PasswordHash: &hash,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	body := `{"email":"bob@example.com","password":"mysecret123","remember_me":false}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var res tokenPair
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Errorf("missing tokens in login response")
	}
	if res.User.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %q", res.User.Email)
	}
}

func TestAuthLogin_InvalidCredentials_Identical401(t *testing.T) {
	h := newTestHarness(t)

	hash, _ := hashPassword("correctpassword")
	u := &domain.User{
		ID:           uuid.New(),
		Email:        "existing@example.com",
		Username:     "existing",
		PasswordHash: &hash,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	// Case 1: Wrong password for existing user
	req1 := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"existing@example.com","password":"wrongpassword"}`))
	rec1 := httptest.NewRecorder()
	h.mux.ServeHTTP(rec1, req1)

	// Case 2: Non-existent email
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"unknown@example.com","password":"anypassword"}`))
	rec2 := httptest.NewRecorder()
	h.mux.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad password, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown email, got %d", rec2.Code)
	}

	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("expected identical response bodies for bad password and unknown email:\ncase 1: %s\ncase 2: %s",
			rec1.Body.String(), rec2.Body.String())
	}

	var errResp map[string]string
	_ = json.Unmarshal(rec1.Body.Bytes(), &errResp)
	if errResp["error"] != "invalid credentials" {
		t.Errorf("expected 'invalid credentials', got %q", errResp["error"])
	}
}

func TestAuthLogin_InactiveUser_403(t *testing.T) {
	h := newTestHarness(t)

	hash, _ := hashPassword("password123")
	u := &domain.User{
		ID:           uuid.New(),
		Email:        "disabled@example.com",
		Username:     "disabled",
		PasswordHash: &hash,
		Role:         domain.RoleUser,
		IsActive:     false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"disabled@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp["error"] != "account is disabled" {
		t.Errorf("expected 'account is disabled', got %q", errResp["error"])
	}
}

func TestAuthLogin_LegacyHashAutoRehash(t *testing.T) {
	h := newTestHarness(t)

	saltHex := "0123456789abcdef0123456789abcdef"
	password := "legacyPassword123"
	sum := sha256.Sum256([]byte(saltHex + password))
	legacyHash := saltHex + ":" + hex.EncodeToString(sum[:])

	userID := uuid.New()
	u := &domain.User{
		ID:           userID,
		Email:        "legacy@example.com",
		Username:     "legacyuser",
		PasswordHash: &legacyHash,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	body := fmt.Sprintf(`{"email":"legacy@example.com","password":"%s"}`, password)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for legacy login, got %d: %s", rec.Code, rec.Body.String())
	}

	rehashed, wasRehashed := h.users.rehashedIDs[userID]
	if !wasRehashed {
		t.Fatalf("expected UpdatePasswordHash to be called for legacy user")
	}

	if !strings.HasPrefix(rehashed, "$2") {
		t.Errorf("expected rehashed password to start with $2, got %q", rehashed)
	}

	// Verify the new bcrypt hash works with the password
	if !checkPassword(rehashed, password) {
		t.Errorf("expected new hash to match password")
	}
}

func TestAuthLogin_SessionTTL_RememberMe(t *testing.T) {
	// 1. remember_me = true -> TTL ~30 days
	{
		h := newTestHarness(t)
		hash, _ := hashPassword("mypassword")
		u := &domain.User{
			ID:           uuid.New(),
			Email:        "remember@example.com",
			Username:     "remember",
			PasswordHash: &hash,
			Role:         domain.RoleUser,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_ = h.users.Create(context.Background(), u)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"remember@example.com","password":"mypassword","remember_me":true}`))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var res tokenPair
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		hashToken := h.jwtMgr.RefreshTokenHash(res.RefreshToken)

		sess, err := h.sessions.GetByTokenHash(context.Background(), hashToken)
		if err != nil {
			t.Fatalf("session not found: %v", err)
		}

		expectedDuration := 30 * 24 * time.Hour
		actualDuration := time.Until(sess.ExpiresAt)
		diff := actualDuration - expectedDuration
		if diff < -time.Minute || diff > time.Minute {
			t.Errorf("expected session TTL ~30 days, got duration %v (diff %v)", actualDuration, diff)
		}
	}

	// 2. remember_me = false -> TTL ~24 hours
	{
		h := newTestHarness(t)
		hash, _ := hashPassword("mypassword")
		u := &domain.User{
			ID:           uuid.New(),
			Email:        "noremeber@example.com",
			Username:     "noremeber",
			PasswordHash: &hash,
			Role:         domain.RoleUser,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_ = h.users.Create(context.Background(), u)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"noremeber@example.com","password":"mypassword","remember_me":false}`))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var res tokenPair
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		hashToken := h.jwtMgr.RefreshTokenHash(res.RefreshToken)

		sess, err := h.sessions.GetByTokenHash(context.Background(), hashToken)
		if err != nil {
			t.Fatalf("session not found: %v", err)
		}

		expectedDuration := 24 * time.Hour
		actualDuration := time.Until(sess.ExpiresAt)
		diff := actualDuration - expectedDuration
		if diff < -time.Minute || diff > time.Minute {
			t.Errorf("expected session TTL ~24 hours, got duration %v (diff %v)", actualDuration, diff)
		}
	}
}

// ── Refresh Tests ─────────────────────────────────────────────────────────────

func TestAuthRefresh_RotationSuccess(t *testing.T) {
	h := newTestHarness(t)

	hash, _ := hashPassword("refreshpass")
	u := &domain.User{
		ID:           uuid.New(),
		Email:        "refresh@example.com",
		Username:     "refreshuser",
		PasswordHash: &hash,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	// Login to get tokens
	loginReq := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"refresh@example.com","password":"refreshpass"}`))
	loginRec := httptest.NewRecorder()
	h.mux.ServeHTTP(loginRec, loginReq)

	var loginRes tokenPair
	_ = json.Unmarshal(loginRec.Body.Bytes(), &loginRes)
	oldRefreshToken := loginRes.RefreshToken
	oldHash := h.jwtMgr.RefreshTokenHash(oldRefreshToken)

	// Call refresh
	refreshBody := fmt.Sprintf(`{"refresh_token":"%s"}`, oldRefreshToken)
	refreshReq := httptest.NewRequest("POST", "/api/v1/auth/refresh", strings.NewReader(refreshBody))
	refreshRec := httptest.NewRecorder()
	h.mux.ServeHTTP(refreshRec, refreshReq)

	if refreshRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on refresh, got %d: %s", refreshRec.Code, refreshRec.Body.String())
	}

	var refreshRes tokenPair
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshRes); err != nil {
		t.Fatalf("failed to decode refresh response: %v", err)
	}

	if refreshRes.AccessToken == "" || refreshRes.RefreshToken == "" {
		t.Errorf("missing new tokens")
	}
	if refreshRes.RefreshToken == oldRefreshToken {
		t.Errorf("expected refreshed token to be rotated, got same token")
	}

	// Verify old session is revoked
	oldSess, err := h.sessions.GetByTokenHash(context.Background(), oldHash)
	if err != nil {
		t.Fatalf("failed to fetch old session: %v", err)
	}
	if !oldSess.IsRevoked {
		t.Errorf("expected old session to be revoked")
	}

	// Verify new session is active
	newHash := h.jwtMgr.RefreshTokenHash(refreshRes.RefreshToken)
	newSess, err := h.sessions.GetByTokenHash(context.Background(), newHash)
	if err != nil {
		t.Fatalf("failed to fetch new session: %v", err)
	}
	if newSess.IsRevoked {
		t.Errorf("expected new session to not be revoked")
	}
}

func TestAuthRefresh_Errors(t *testing.T) {
	h := newTestHarness(t)

	userID := uuid.New()
	u := &domain.User{
		ID:        userID,
		Email:     "user@example.com",
		Username:  "user",
		Role:      domain.RoleUser,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	// 1. Unknown token
	{
		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"nonexistent_token"}`))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for unknown token, got %d", rec.Code)
		}
		var errResp map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp["error"] != "invalid refresh token" {
			t.Errorf("expected 'invalid refresh token', got %q", errResp["error"])
		}
	}

	// 2. Revoked token
	{
		revokedToken := "revoked-token-value"
		revokedHash := h.jwtMgr.RefreshTokenHash(revokedToken)
		_ = h.sessions.Create(context.Background(), &domain.UserSession{
			ID:               uuid.New(),
			UserID:           userID,
			RefreshTokenHash: revokedHash,
			IsRevoked:        true,
			ExpiresAt:        time.Now().Add(24 * time.Hour),
		})

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":"%s"}`, revokedToken))))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for revoked token, got %d", rec.Code)
		}
		var errResp map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp["error"] != "refresh token has been revoked" {
			t.Errorf("expected 'refresh token has been revoked', got %q", errResp["error"])
		}
	}

	// 3. Expired token
	{
		expiredToken := "expired-token-value"
		expiredHash := h.jwtMgr.RefreshTokenHash(expiredToken)
		_ = h.sessions.Create(context.Background(), &domain.UserSession{
			ID:               uuid.New(),
			UserID:           userID,
			RefreshTokenHash: expiredHash,
			IsRevoked:        false,
			ExpiresAt:        time.Now().Add(-1 * time.Hour),
		})

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":"%s"}`, expiredToken))))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for expired token, got %d", rec.Code)
		}
		var errResp map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp["error"] != "refresh token expired" {
			t.Errorf("expected 'refresh token expired', got %q", errResp["error"])
		}
	}

	// 4. Inactive user (IsActive: false) -> 403 Forbidden
	{
		inactiveUserID := uuid.New()
		inactiveUser := &domain.User{
			ID:        inactiveUserID,
			Email:     "inactive@example.com",
			Username:  "inactive",
			Role:      domain.RoleUser,
			IsActive:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = h.users.Create(context.Background(), inactiveUser)

		inactiveToken := "inactive-refresh-token"
		inactiveHash := h.jwtMgr.RefreshTokenHash(inactiveToken)
		_ = h.sessions.Create(context.Background(), &domain.UserSession{
			ID:               uuid.New(),
			UserID:           inactiveUserID,
			RefreshTokenHash: inactiveHash,
			IsRevoked:        false,
			ExpiresAt:        time.Now().Add(24 * time.Hour),
		})

		req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":"%s"}`, inactiveToken))))
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for inactive user refresh, got %d", rec.Code)
		}
		var errResp map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp["error"] != "account is disabled" {
			t.Errorf("expected 'account is disabled', got %q", errResp["error"])
		}
	}
}

// ── Logout Tests ──────────────────────────────────────────────────────────────

func TestAuthLogout(t *testing.T) {
	h := newTestHarness(t)

	userID := uuid.New()
	u := &domain.User{
		ID:        userID,
		Email:     "logout@example.com",
		Username:  "logoutuser",
		Role:      domain.RoleUser,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	accessToken, err := h.jwtMgr.GenerateAccess(u)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	refreshToken := "logout-refresh-token"
	refreshHash := h.jwtMgr.RefreshTokenHash(refreshToken)
	_ = h.sessions.Create(context.Background(), &domain.UserSession{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		IsRevoked:        false,
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	})

	// 1. Logout without Bearer token -> 401
	{
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without Bearer, got %d", rec.Code)
		}
	}

	// 2. Logout with Bearer and refresh token -> 204 and session revoked
	{
		body := fmt.Sprintf(`{"refresh_token":"%s"}`, refreshToken)
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
		}

		sess, _ := h.sessions.GetByTokenHash(context.Background(), refreshHash)
		if !sess.IsRevoked {
			t.Errorf("expected session to be revoked after logout")
		}
	}

	// 3. Logout with Bearer and empty body -> 204
	{
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content with empty body, got %d", rec.Code)
		}
	}
}

// ── Me Tests ──────────────────────────────────────────────────────────────────

func TestAuthMe(t *testing.T) {
	h := newTestHarness(t)

	userID := uuid.New()
	u := &domain.User{
		ID:         userID,
		Email:      "me@example.com",
		Username:   "meuser",
		Role:       domain.RoleUser,
		Profession: "Backend developer",
		Gender:     "Male",
		Language:   "English",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	_ = h.users.Create(context.Background(), u)

	accessToken, err := h.jwtMgr.GenerateAccess(u)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	// 1. Success 200
	{
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var res domain.User
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.ID != userID || res.Email != "me@example.com" {
			t.Errorf("unexpected user in response: %+v", res)
		}
	}

	// 2. Unauthorized 401 without Bearer token
	{
		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	}

	// 3. User not found 404
	{
		deletedUser := &domain.User{
			ID:       uuid.New(),
			Email:    "ghost@example.com",
			Username: "ghost",
			Role:     domain.RoleUser,
			IsActive: true,
		}
		ghostToken, _ := h.jwtMgr.GenerateAccess(deletedUser)

		req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+ghostToken)
		rec := httptest.NewRecorder()
		h.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for non-existent user, got %d", rec.Code)
		}
	}
}
