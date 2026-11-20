package handler_test

import (
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
	"golang.org/x/crypto/bcrypt"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// ── Fakes and Mocks ──────────────────────────────────────────────────────────

type mockUserStore struct {
	mu            sync.Mutex
	users         map[uuid.UUID]*domain.User
	byEmail       map[string]*domain.User
	updatePassErr error
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users:   make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *mockUserStore) add(u *domain.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users[u.ID] = &cp
	m.byEmail[strings.ToLower(u.Email)] = &cp
}

func (m *mockUserStore) Create(ctx context.Context, u *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users[u.ID] = &cp
	m.byEmail[strings.ToLower(u.Email)] = &cp
	return nil
}

func (m *mockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	return nil
}

func (m *mockUserStore) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updatePassErr != nil {
		return m.updatePassErr
	}
	u, ok := m.users[id]
	if !ok {
		return errors.New("user not found")
	}
	u.PasswordHash = &hash
	return nil
}

func (m *mockUserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}

type mockPasswordResetStore struct {
	mu             sync.Mutex
	tokensByHash   map[string]*domain.PasswordResetToken
	tokensByID     map[uuid.UUID]*domain.PasswordResetToken
	deleteExpiredN int
}

func newMockPasswordResetStore() *mockPasswordResetStore {
	return &mockPasswordResetStore{
		tokensByHash: make(map[string]*domain.PasswordResetToken),
		tokensByID:   make(map[uuid.UUID]*domain.PasswordResetToken),
	}
}

func (m *mockPasswordResetStore) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	cp := *token
	m.tokensByHash[token.TokenHash] = &cp
	m.tokensByID[token.ID] = &cp
	return nil
}

func (m *mockPasswordResetStore) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokensByHash[tokenHash]
	if !ok {
		return nil, errors.New("token not found")
	}
	cp := *tok
	return &cp, nil
}

func (m *mockPasswordResetStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokensByID[id]
	if !ok {
		return errors.New("token not found")
	}
	now := time.Now()
	tok.UsedAt = &now
	if byHash, exists := m.tokensByHash[tok.TokenHash]; exists {
		byHash.UsedAt = &now
	}
	return nil
}

func (m *mockPasswordResetStore) DeleteExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteExpiredN++
	now := time.Now()
	for h, t := range m.tokensByHash {
		if now.After(t.ExpiresAt) {
			delete(m.tokensByHash, h)
			delete(m.tokensByID, t.ID)
		}
	}
	return nil
}

type mockSessionStore struct {
	mu          sync.Mutex
	revokedUser []uuid.UUID
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{}
}

func (m *mockSessionStore) Create(ctx context.Context, s *domain.UserSession) error {
	return nil
}

func (m *mockSessionStore) GetByTokenHash(ctx context.Context, hash string) (*domain.UserSession, error) {
	return nil, errors.New("not implemented")
}

func (m *mockSessionStore) RevokeByTokenHash(ctx context.Context, hash string) error {
	return nil
}

func (m *mockSessionStore) RevokeByUser(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokedUser = append(m.revokedUser, userID)
	return nil
}

func (m *mockSessionStore) CleanupExpired(ctx context.Context) error {
	return nil
}

type spyMailer struct {
	mu      sync.Mutex
	calls   []mailerCall
	sendErr error
}

type mailerCall struct {
	to   string
	link string
}

func (s *spyMailer) SendReset(ctx context.Context, to, link string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sendErr != nil {
		return s.sendErr
	}
	s.calls = append(s.calls, mailerCall{to: to, link: link})
	return nil
}

// ── Test Fixture ─────────────────────────────────────────────────────────────

type testFixture struct {
	userStore  *mockUserStore
	resetStore *mockPasswordResetStore
	sessStore  *mockSessionStore
	mailer     *spyMailer
	jwtMgr     *jwtpkg.Manager
	mux        *http.ServeMux
	alice      *domain.User
	aliceToken string
}

func setupFixture(t *testing.T) *testFixture {
	t.Helper()

	userStore := newMockUserStore()
	resetStore := newMockPasswordResetStore()
	sessStore := newMockSessionStore()
	mailerSpy := &spyMailer{}

	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:            "supersecretkeyforpasswordresettests123",
		AccessTokenExpireMin: 60,
	})

	pwHash, err := bcrypt.GenerateFromPassword([]byte("oldPassword123"), 12)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	pwHashStr := string(pwHash)

	alice := &domain.User{
		ID:           uuid.New(),
		Email:        "alice@example.com",
		Username:     "alice",
		PasswordHash: &pwHashStr,
		Role:         domain.RoleUser,
		IsActive:     true,
	}
	userStore.add(alice)

	aliceToken, err := jwtMgr.GenerateAccess(alice)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	h := handler.NewPasswordResetHandler(
		userStore,
		resetStore,
		sessStore,
		mailerSpy,
		"http://localhost:3000",
	)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux, middleware.Auth(jwtMgr))

	return &testFixture{
		userStore:  userStore,
		resetStore: resetStore,
		sessStore:  sessStore,
		mailer:     mailerSpy,
		jwtMgr:     jwtMgr,
		mux:        mux,
		alice:      alice,
		aliceToken: aliceToken,
	}
}

// ── Forgot Password Tests ────────────────────────────────────────────────────

func TestPasswordForgot_ExistingUser_Returns202_AndSendsEmail(t *testing.T) {
	f := setupFixture(t)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/forgot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	if len(f.mailer.calls) != 1 {
		t.Fatalf("expected 1 mailer call, got %d", len(f.mailer.calls))
	}

	call := f.mailer.calls[0]
	if call.to != "alice@example.com" {
		t.Errorf("expected email to %q, got %q", "alice@example.com", call.to)
	}
	if !strings.HasPrefix(call.link, "http://localhost:3000/reset-password?token=") {
		t.Errorf("expected link starting with frontend URL, got %q", call.link)
	}

	// Extract raw token from link
	rawToken := strings.TrimPrefix(call.link, "http://localhost:3000/reset-password?token=")
	if rawToken == "" {
		t.Fatal("empty raw token in reset link")
	}

	// Verify SHA-256 hash was stored in resetStore
	hash := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(hash[:])

	tok, err := f.resetStore.GetByTokenHash(context.Background(), expectedHash)
	if err != nil {
		t.Fatalf("token hash not found in store: %v", err)
	}
	if tok.UserID != f.alice.ID {
		t.Errorf("expected token for user %v, got %v", f.alice.ID, tok.UserID)
	}
	if tok.UsedAt != nil {
		t.Errorf("expected used_at to be nil, got %v", tok.UsedAt)
	}
	if time.Until(tok.ExpiresAt) < 55*time.Minute || time.Until(tok.ExpiresAt) > 65*time.Minute {
		t.Errorf("expected token TTL around 1 hour, got expires_at=%v", tok.ExpiresAt)
	}
}

func TestPasswordForgot_NonExistentUser_Returns202_NoEmail(t *testing.T) {
	f := setupFixture(t)

	body := `{"email":"unknown@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/forgot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}

	if len(f.mailer.calls) != 0 {
		t.Fatalf("mailer should NOT be called for non-existent user, got %d calls", len(f.mailer.calls))
	}
	if len(f.resetStore.tokensByHash) != 0 {
		t.Fatalf("no tokens should be stored for non-existent user")
	}
}

func TestPasswordForgot_MailerError_Returns500(t *testing.T) {
	f := setupFixture(t)
	f.mailer.sendErr = errors.New("smtp connection failed")

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/forgot", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when mailer fails, got %d", rec.Code)
	}
}

func TestPasswordForgot_InvalidBody_Returns400(t *testing.T) {
	f := setupFixture(t)

	testCases := []string{
		`{invalid json`,
		`{}`,
		`{"email":""}`,
		`{"email":"not-an-email"}`,
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("POST", "/api/v1/auth/password/forgot", strings.NewReader(tc))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		f.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for payload %q, got %d", tc, rec.Code)
		}
	}
}

// ── Reset Password Tests ─────────────────────────────────────────────────────

func TestPasswordReset_Success(t *testing.T) {
	f := setupFixture(t)

	rawToken := "random-test-reset-token-32bytes-long"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	tok := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    f.alice.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	_ = f.resetStore.Create(context.Background(), tok)

	body := fmt.Sprintf(`{"token":%q,"new_password":"newSecurePassword123"}`, rawToken)
	req := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify user password hash was updated
	updatedUser, err := f.userStore.GetByID(context.Background(), f.alice.ID)
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}
	if updatedUser.PasswordHash == nil {
		t.Fatal("user password hash is nil")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*updatedUser.PasswordHash), []byte("newSecurePassword123")); err != nil {
		t.Errorf("new password hash does not match new password: %v", err)
	}

	// Verify token is marked used
	storedTok, err := f.resetStore.GetByTokenHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("token not found: %v", err)
	}
	if storedTok.UsedAt == nil {
		t.Errorf("expected token to be marked used")
	}

	// Verify user sessions were revoked
	f.sessStore.mu.Lock()
	revoked := len(f.sessStore.revokedUser)
	f.sessStore.mu.Unlock()
	if revoked != 1 {
		t.Errorf("expected RevokeByUser to be called once, got %d", revoked)
	}
}

func TestPasswordReset_ReusedToken_Returns401(t *testing.T) {
	f := setupFixture(t)

	rawToken := "reused-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	now := time.Now()
	tok := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    f.alice.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &now,
		CreatedAt: time.Now().Add(-10 * time.Minute),
	}
	_ = f.resetStore.Create(context.Background(), tok)

	body := fmt.Sprintf(`{"token":%q,"new_password":"newPassword123"}`, rawToken)
	req := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for reused token, got %d", rec.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid or expired reset token" {
		t.Errorf("expected error %q, got %q", "invalid or expired reset token", resp["error"])
	}
}

func TestPasswordReset_ExpiredToken_Returns401(t *testing.T) {
	f := setupFixture(t)

	rawToken := "expired-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	tok := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    f.alice.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(-5 * time.Minute), // expired
		CreatedAt: time.Now().Add(-65 * time.Minute),
	}
	_ = f.resetStore.Create(context.Background(), tok)

	body := fmt.Sprintf(`{"token":%q,"new_password":"newPassword123"}`, rawToken)
	req := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for expired token, got %d", rec.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid or expired reset token" {
		t.Errorf("expected error %q, got %q", "invalid or expired reset token", resp["error"])
	}
}

func TestPasswordReset_InvalidToken_Returns401(t *testing.T) {
	f := setupFixture(t)

	body := `{"token":"non-existent-token","new_password":"newPassword123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unknown token, got %d", rec.Code)
	}
}

func TestPasswordReset_PasswordValidation(t *testing.T) {
	f := setupFixture(t)

	rawToken := "valid-token-for-validation"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	tok := &domain.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    f.alice.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}
	_ = f.resetStore.Create(context.Background(), tok)

	shortPass := "short"
	bodyShort := fmt.Sprintf(`{"token":%q,"new_password":%q}`, rawToken, shortPass)
	reqShort := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(bodyShort))
	reqShort.Header.Set("Content-Type", "application/json")
	recShort := httptest.NewRecorder()

	f.mux.ServeHTTP(recShort, reqShort)

	if recShort.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", recShort.Code)
	}
	var respShort map[string]string
	_ = json.Unmarshal(recShort.Body.Bytes(), &respShort)
	if respShort["error"] != "password must be between 8 and 128 characters" {
		t.Errorf("expected error 'password must be between 8 and 128 characters', got %q", respShort["error"])
	}

	longPass := strings.Repeat("a", 129)
	bodyLong := fmt.Sprintf(`{"token":%q,"new_password":%q}`, rawToken, longPass)
	reqLong := httptest.NewRequest("POST", "/api/v1/auth/password/reset", strings.NewReader(bodyLong))
	reqLong.Header.Set("Content-Type", "application/json")
	recLong := httptest.NewRecorder()

	f.mux.ServeHTTP(recLong, reqLong)

	if recLong.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long password, got %d", recLong.Code)
	}
}

// ── Change Password Tests ────────────────────────────────────────────────────

func TestPasswordChange_Success(t *testing.T) {
	f := setupFixture(t)

	body := `{"current_password":"oldPassword123","new_password":"brandNewPassword789"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/change", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify updated password
	user, err := f.userStore.GetByID(context.Background(), f.alice.ID)
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("brandNewPassword789")); err != nil {
		t.Errorf("new password hash does not match: %v", err)
	}

	// Verify sessions revoked
	f.sessStore.mu.Lock()
	revoked := len(f.sessStore.revokedUser)
	f.sessStore.mu.Unlock()
	if revoked != 1 {
		t.Errorf("expected sessions to be revoked, got %d calls", revoked)
	}
}

func TestPasswordChange_InvalidCurrentPassword(t *testing.T) {
	f := setupFixture(t)

	body := `{"current_password":"wrongOldPassword","new_password":"brandNewPassword789"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/change", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for wrong current password, got %d", rec.Code)
	}

	var resp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid current password" {
		t.Errorf("expected error %q, got %q", "invalid current password", resp["error"])
	}
}

func TestPasswordChange_PasswordValidation(t *testing.T) {
	f := setupFixture(t)

	// Short password (<8)
	bodyShort := `{"current_password":"oldPassword123","new_password":"short"}`
	reqShort := httptest.NewRequest("POST", "/api/v1/auth/password/change", strings.NewReader(bodyShort))
	reqShort.Header.Set("Content-Type", "application/json")
	reqShort.Header.Set("Authorization", "Bearer "+f.aliceToken)
	recShort := httptest.NewRecorder()

	f.mux.ServeHTTP(recShort, reqShort)

	if recShort.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d", recShort.Code)
	}
	var respShort map[string]string
	_ = json.Unmarshal(recShort.Body.Bytes(), &respShort)
	if respShort["error"] != "password must be between 8 and 128 characters" {
		t.Errorf("expected error 'password must be between 8 and 128 characters', got %q", respShort["error"])
	}

	// Long password (>128)
	bodyLong := fmt.Sprintf(`{"current_password":"oldPassword123","new_password":%q}`, strings.Repeat("a", 129))
	reqLong := httptest.NewRequest("POST", "/api/v1/auth/password/change", strings.NewReader(bodyLong))
	reqLong.Header.Set("Content-Type", "application/json")
	reqLong.Header.Set("Authorization", "Bearer "+f.aliceToken)
	recLong := httptest.NewRecorder()

	f.mux.ServeHTTP(recLong, reqLong)

	if recLong.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long password, got %d", recLong.Code)
	}
}

func TestPasswordChange_Unauthenticated(t *testing.T) {
	f := setupFixture(t)

	body := `{"current_password":"oldPassword123","new_password":"brandNewPassword789"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/password/change", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	rec := httptest.NewRecorder()

	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", rec.Code)
	}
}
