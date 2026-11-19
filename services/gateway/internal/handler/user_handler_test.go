package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
	"github.com/soskiiit/errorium/services/gateway/pkg/storage"
)

type mockUserRepo struct {
	users     map[uuid.UUID]*domain.User
	byEmail   map[string]*domain.User
	updateErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) add(u *domain.User) {
	m.users[u.ID] = u
	m.byEmail[strings.ToLower(u.Email)] = u
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, errors.New("user not found")
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) UpdateProfile(ctx context.Context, u *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if existing, ok := m.users[u.ID]; ok {
		delete(m.byEmail, strings.ToLower(existing.Email))
	}
	cp := *u
	m.users[u.ID] = &cp
	m.byEmail[strings.ToLower(u.Email)] = &cp
	return nil
}

type mockAvatarStore struct {
	data         map[string][]byte
	contentTypes map[string]string
	putErr       error
	getErr       error
}

func newMockAvatarStore() *mockAvatarStore {
	return &mockAvatarStore{
		data:         make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

func (m *mockAvatarStore) Put(ctx context.Context, key, contentType string, data []byte) error {
	if m.putErr != nil {
		return m.putErr
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.data[key] = cp
	m.contentTypes[key] = contentType
	return nil
}

func (m *mockAvatarStore) Get(ctx context.Context, key string) ([]byte, string, error) {
	if m.getErr != nil {
		return nil, "", m.getErr
	}
	data, ok := m.data[key]
	if !ok {
		return nil, "", storage.ErrNotFound
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	return cp, m.contentTypes[key], nil
}

type testUserFixture struct {
	userRepo    *mockUserRepo
	avatarStore *mockAvatarStore
	jwtMgr      *jwtpkg.Manager
	mux         *http.ServeMux
	alice       *domain.User
	bob         *domain.User
	aliceToken  string
	bobToken    string
}

func setupUserTestFixture(t *testing.T) *testUserFixture {
	t.Helper()

	userRepo := newMockUserRepo()
	avatarStore := newMockAvatarStore()

	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:              "super-secret-test-key-32-bytes-long!",
		AccessTokenExpireMin:   15,
		RefreshTokenExpireDays: 7,
	})

	now := time.Now().UTC()
	alice := &domain.User{
		ID:         uuid.New(),
		Email:      "alice@example.com",
		Username:   "alice",
		Role:       domain.RoleUser,
		Profession: "Backend developer",
		Gender:     "Female",
		Language:   "English",
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	bob := &domain.User{
		ID:         uuid.New(),
		Email:      "bob@example.com",
		Username:   "bob",
		Role:       domain.RoleUser,
		Profession: "Frontend developer",
		Gender:     "Male",
		Language:   "Русский",
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	userRepo.add(alice)
	userRepo.add(bob)

	aliceToken, err := jwtMgr.GenerateAccess(alice)
	if err != nil {
		t.Fatalf("failed to generate alice token: %v", err)
	}
	bobToken, err := jwtMgr.GenerateAccess(bob)
	if err != nil {
		t.Fatalf("failed to generate bob token: %v", err)
	}

	mux := http.NewServeMux()
	userHandler := handler.NewUserHandler(userRepo, avatarStore)
	userHandler.RegisterRoutes(mux, middleware.Auth(jwtMgr))

	return &testUserFixture{
		userRepo:    userRepo,
		avatarStore: avatarStore,
		jwtMgr:      jwtMgr,
		mux:         mux,
		alice:       alice,
		bob:         bob,
		aliceToken:  aliceToken,
		bobToken:    bobToken,
	}
}

func createMultipartAvatar(t *testing.T, fieldName, filename, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

// ---------------------------------------------------------------------------
// GET /api/v1/users/me Tests
// ---------------------------------------------------------------------------

func TestUserHandler_GetMe(t *testing.T) {
	f := setupUserTestFixture(t)

	// 1. Authorized GET /me returns 200 and user profile
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.User
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal user response: %v", err)
	}
	if resp.ID != f.alice.ID {
		t.Errorf("expected user ID %s, got %s", f.alice.ID, resp.ID)
	}
	if resp.Email != f.alice.Email {
		t.Errorf("expected email %s, got %s", f.alice.Email, resp.Email)
	}
	if resp.PasswordHash != nil {
		t.Errorf("password_hash must be omitted from JSON response")
	}

	// 2. Unauthorized request returns 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	unauthRec := httptest.NewRecorder()
	f.mux.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthRec.Code)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/users/me Tests
// ---------------------------------------------------------------------------

func TestUserHandler_UpdateMe_RoleRejection(t *testing.T) {
	f := setupUserTestFixture(t)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "only role in body",
			body: `{"role": "admin"}`,
		},
		{
			name: "role with valid username",
			body: `{"username": "alice_admin", "role": "admin"}`,
		},
		{
			name: "role with null",
			body: `{"role": null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+f.aliceToken)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			f.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request on role edit, got %d: %s", rec.Code, rec.Body.String())
			}

			if !strings.Contains(rec.Body.String(), "role is not editable") {
				t.Errorf("expected error message 'role is not editable', got %s", rec.Body.String())
			}

			// Ensure role remains RoleUser in repository
			u, _ := f.userRepo.GetByID(context.Background(), f.alice.ID)
			if u.Role != domain.RoleUser {
				t.Errorf("expected user role to remain 'user', got %s", u.Role)
			}
		})
	}
}

func TestUserHandler_UpdateMe_EmptyBody(t *testing.T) {
	f := setupUserTestFixture(t)

	emptyBodies := []string{
		`{}`,
		`{"unknown_field": "val"}`,
	}

	for _, body := range emptyBodies {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+f.aliceToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		f.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 on empty update body, got %d: %s", rec.Code, rec.Body.String())
		}
	}
}

func TestUserHandler_UpdateMe_ValidProfileUpdate(t *testing.T) {
	f := setupUserTestFixture(t)

	// Update username
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(`{"username": "alice_updated"}`))
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp domain.User
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Username != "alice_updated" {
		t.Errorf("expected username 'alice_updated', got %q", resp.Username)
	}

	// Update all profile fields simultaneously
	fullBody := `{
		"username": "alice_full",
		"email": "alice_new@example.com",
		"gender": "Not specified",
		"language": "Русский",
		"profession": "Lead System Architect"
	}`
	reqFull := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(fullBody))
	reqFull.Header.Set("Authorization", "Bearer "+f.aliceToken)
	reqFull.Header.Set("Content-Type", "application/json")
	recFull := httptest.NewRecorder()
	f.mux.ServeHTTP(recFull, reqFull)

	if recFull.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recFull.Code, recFull.Body.String())
	}

	var fullResp domain.User
	if err := json.Unmarshal(recFull.Body.Bytes(), &fullResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if fullResp.Username != "alice_full" {
		t.Errorf("expected username 'alice_full', got %q", fullResp.Username)
	}
	if fullResp.Email != "alice_new@example.com" {
		t.Errorf("expected email 'alice_new@example.com', got %q", fullResp.Email)
	}
	if fullResp.Gender != "Not specified" {
		t.Errorf("expected gender 'Not specified', got %q", fullResp.Gender)
	}
	if fullResp.Language != "Русский" {
		t.Errorf("expected language 'Русский', got %q", fullResp.Language)
	}
	if fullResp.Profession != "Lead System Architect" {
		t.Errorf("expected profession 'Lead System Architect', got %q", fullResp.Profession)
	}
}

func TestUserHandler_UpdateMe_EmailCollision(t *testing.T) {
	f := setupUserTestFixture(t)

	// Alice tries to take Bob's email
	body := fmt.Sprintf(`{"email": "%s"}`, f.bob.Email)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserHandler_UpdateMe_KeepingOwnEmail(t *testing.T) {
	f := setupUserTestFixture(t)

	// Alice keeps her own email (case-insensitive test)
	body := fmt.Sprintf(`{"email": "%s", "username": "alice_same_email"}`, strings.ToUpper(f.alice.Email))
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when keeping own email, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserHandler_UpdateMe_ValidationErrors(t *testing.T) {
	f := setupUserTestFixture(t)

	testCases := []struct {
		name string
		body string
	}{
		{"empty username", `{"username": ""}`},
		{"whitespace username", `{"username": "   "}`},
		{"username too long", fmt.Sprintf(`{"username": "%s"}`, strings.Repeat("a", 101))},
		{"invalid email missing at", `{"email": "invalid-email"}`},
		{"invalid email missing domain", `{"email": "alice@"}`},
		{"invalid gender", `{"gender": "Other"}`},
		{"invalid language", `{"language": "Français"}`},
		{"empty profession", `{"profession": ""}`},
		{"whitespace profession", `{"profession": "   "}`},
		{"profession too long", fmt.Sprintf(`{"profession": "%s"}`, strings.Repeat("p", 101))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+f.aliceToken)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			f.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for %s, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/users/avatar Tests
// ---------------------------------------------------------------------------

var minimalPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func TestUserHandler_UploadAvatar_Success(t *testing.T) {
	f := setupUserTestFixture(t)

	body, contentType := createMultipartAvatar(t, "avatar", "avatar.png", "image/png", minimalPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", body)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedURL := fmt.Sprintf("/api/v1/users/%s/avatar", f.alice.ID)
	if resp.AvatarURL != expectedURL {
		t.Errorf("expected avatar_url %q, got %q", expectedURL, resp.AvatarURL)
	}

	// Verify avatar was stored in AvatarStore
	key := fmt.Sprintf("avatars/%s", f.alice.ID)
	data, ct, err := f.avatarStore.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("expected avatar to be stored in AvatarStore: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("expected stored content-type 'image/png', got %q", ct)
	}
	if !bytes.Equal(data, minimalPNG) {
		t.Errorf("stored data does not match uploaded data")
	}

	// Verify user's avatar_url was updated in repository
	u, _ := f.userRepo.GetByID(context.Background(), f.alice.ID)
	if u.AvatarURL == nil || *u.AvatarURL != expectedURL {
		t.Errorf("expected user AvatarURL to be %q, got %v", expectedURL, u.AvatarURL)
	}
}

func TestUserHandler_UploadAvatar_UnsupportedMIME(t *testing.T) {
	f := setupUserTestFixture(t)

	// Upload text file
	textContent := []byte("this is plain text, not an image file")
	body, contentType := createMultipartAvatar(t, "avatar", "avatar.txt", "text/plain", textContent)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", body)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on unsupported MIME, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unsupported image format") {
		t.Errorf("expected 'unsupported image format', got %s", rec.Body.String())
	}
}

func TestUserHandler_UploadAvatar_MissingFile(t *testing.T) {
	f := setupUserTestFixture(t)

	// Multipart without "avatar" field
	body, contentType := createMultipartAvatar(t, "photo", "photo.png", "image/png", minimalPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", body)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on missing avatar field, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUserHandler_UploadAvatar_FileExceeds5MB(t *testing.T) {
	f := setupUserTestFixture(t)

	// Create payload larger than 5 MiB (5*1024*1024 + 512 bytes)
	largeData := make([]byte, 5<<20+512)
	copy(largeData, minimalPNG) // prepend PNG header so MIME check wouldn't fail first

	body, contentType := createMultipartAvatar(t, "avatar", "large.png", "image/png", largeData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", body)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on file > 5 MiB, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "file size exceeds 5MB limit") {
		t.Errorf("expected 'file size exceeds 5MB limit', got %s", rec.Body.String())
	}
}

func TestUserHandler_UploadAvatar_S3Failure(t *testing.T) {
	f := setupUserTestFixture(t)
	f.avatarStore.putErr = errors.New("s3 endpoint connection timeout")

	body, contentType := createMultipartAvatar(t, "avatar", "avatar.png", "image/png", minimalPNG)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", body)
	req.Header.Set("Authorization", "Bearer "+f.aliceToken)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway on S3 failure, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/users/{id}/avatar Tests (Public Route)
// ---------------------------------------------------------------------------

func TestUserHandler_GetAvatar_PublicSuccess(t *testing.T) {
	f := setupUserTestFixture(t)

	// Pre-populate avatar in store
	key := fmt.Sprintf("avatars/%s", f.alice.ID)
	_ = f.avatarStore.Put(context.Background(), key, "image/png", minimalPNG)

	// Public GET request without Authorization header
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/avatar", f.alice.ID), nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type 'image/png', got %q", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Length") != fmt.Sprintf("%d", len(minimalPNG)) {
		t.Errorf("expected Content-Length '%d', got %q", len(minimalPNG), rec.Header().Get("Content-Length"))
	}
	if rec.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("expected Cache-Control 'public, max-age=3600', got %q", rec.Header().Get("Cache-Control"))
	}
	etag := rec.Header().Get("ETag")
	if etag == "" || !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Errorf("expected quote-wrapped ETag header, got %q", etag)
	}

	if !bytes.Equal(rec.Body.Bytes(), minimalPNG) {
		t.Errorf("returned body does not match stored image data")
	}
}

func TestUserHandler_GetAvatar_NotFound(t *testing.T) {
	f := setupUserTestFixture(t)

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/avatar", nonExistentID), nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "avatar not found") {
		t.Errorf("expected 'avatar not found', got %s", rec.Body.String())
	}
}

func TestUserHandler_GetAvatar_InvalidUUID(t *testing.T) {
	f := setupUserTestFixture(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-valid-uuid/avatar", nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on invalid UUID, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid user id") {
		t.Errorf("expected 'invalid user id', got %s", rec.Body.String())
	}
}

func TestUserHandler_GetAvatar_S3Failure(t *testing.T) {
	f := setupUserTestFixture(t)
	f.avatarStore.getErr = errors.New("s3 internal connection reset")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/avatar", f.alice.ID), nil)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway on S3 failure, got %d: %s", rec.Code, rec.Body.String())
	}
}
