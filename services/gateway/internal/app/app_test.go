package app_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/app"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

func TestApp_NoDB_Health200_Chats404(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	// 1. GET /health should succeed (200)
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200, got %d, body: %s", recHealth.Code, recHealth.Body.String())
	}

	var healthResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recHealth.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("failed to parse /health response JSON: %v", err)
	}
	if healthResp.Status != "ok" {
		t.Fatalf("expected health status 'ok', got %q", healthResp.Status)
	}

	// 2. Unregistered routes (when DB is unconfigured) must return 404
	unregisteredPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/chats"},
		{http.MethodPost, "/api/v1/chats"},
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodPatch, "/api/v1/users/me"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/password/forgot"},
		{http.MethodPost, "/api/v1/auth/password/reset"},
		{http.MethodPost, "/api/v1/auth/password/change"},
		{http.MethodGet, "/api/v1/admin/stats"},
		{http.MethodPost, "/api/v1/admin/index/reload"},
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/admin/rag/config"},
	}

	for _, up := range unregisteredPaths {
		req := httptest.NewRequest(up.method, up.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %s %s to return 404 when DB is unconfigured, got %d", up.method, up.path, rec.Code)
		}
	}

	// 3. Public routes must be accessible even when DB is unconfigured
	publicRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/tags", ""},
		{http.MethodGet, "/api/v1/status", ""},
		{http.MethodPost, "/api/v1/search", `{"query":"syntax"}`},
		{http.MethodPost, "/api/v1/search/rag", `{"query":"syntax"}`},
		{http.MethodGet, "/api/v1/graph/threads?doc_ids=27539308", ""},
		{http.MethodGet, "/api/v1/graph/threads/27539308", ""},
	}

	for _, pr := range publicRoutes {
		var req *http.Request
		if pr.body != "" {
			req = httptest.NewRequest(pr.method, pr.path, strings.NewReader(pr.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(pr.method, pr.path, nil)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected public route %s %s to return 200 without DB, got %d, body: %s", pr.method, pr.path, rec.Code, rec.Body.String())
		}
	}
}

func TestApp_CORS_Headers(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}

	methods := rec.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST, PATCH, DELETE, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods 'GET, POST, PATCH, DELETE, OPTIONS', got %q", methods)
	}

	headers := rec.Header().Get("Access-Control-Allow-Headers")
	if headers != "Authorization, Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers 'Authorization, Content-Type', got %q", headers)
	}
}

func TestApp_MCP_RoutesRegistered(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
	}

	application := app.New(cfg)
	handler := application.Handler()

	// 1. POST /mcp/messages without sessionId should return 404 {"error":"session not found"} (route is registered!)
	reqMessages := httptest.NewRequest(http.MethodPost, "/mcp/messages", strings.NewReader(`{}`))
	recMessages := httptest.NewRecorder()
	handler.ServeHTTP(recMessages, reqMessages)

	if recMessages.Code != http.StatusNotFound {
		t.Errorf("expected POST /mcp/messages to return 404, got %d", recMessages.Code)
	}

	// 2. GET /sse opens stream with endpoint event
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqSSE := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)
	recSSE := httptest.NewRecorder()
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	handler.ServeHTTP(recSSE, reqSSE)

	if recSSE.Code != http.StatusOK {
		t.Errorf("expected GET /sse to return 200, got %d", recSSE.Code)
	}
	if !strings.Contains(recSSE.Body.String(), "event: endpoint") {
		t.Errorf("expected event: endpoint in /sse body, got %s", recSSE.Body.String())
	}
}

func TestApp_CORS_OPTIONS_204(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/chats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected OPTIONS to return 204 No Content, got %d", rec.Code)
	}

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}
}

func TestApp_Shutdown(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		t.Fatalf("expected graceful shutdown without error, got: %v", err)
	}
}

func TestApp_ServerTimeouts_WriteTimeoutZero(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	server := application.Server()
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.WriteTimeout != 0 {
		t.Errorf("expected WriteTimeout to be 0 (unlimited for SSE), got %v", server.WriteTimeout)
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("expected ReadHeaderTimeout 5s, got %v", server.ReadHeaderTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout 60s, got %v", server.IdleTimeout)
	}
}

type fakeUserStore struct {
	users map[string]*domain.User
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: make(map[string]*domain.User)}
}

func (f *fakeUserStore) Create(ctx context.Context, u *domain.User) error {
	f.users[u.Email] = u
	return nil
}

func (f *fakeUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (f *fakeUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

func (f *fakeUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	f.users[u.Email] = u
	return nil
}

func (f *fakeUserStore) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return nil
}

func (f *fakeUserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}

func (f *fakeUserStore) List(ctx context.Context, limit, offset int, search string) ([]*domain.User, int, error) {
	return nil, 0, nil
}

func (f *fakeUserStore) UpdateRoleAndStatus(ctx context.Context, id uuid.UUID, role *domain.Role, isActive *bool) (*domain.User, error) {
	return nil, nil
}

func TestBootstrapFirstAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("creates first admin when none exists", func(t *testing.T) {
		store := newFakeUserStore()
		email := "admin@errorium.ru"
		pass := "AdminSecurePass123!"

		err := app.BootstrapFirstAdmin(ctx, store, email, pass)
		if err != nil {
			t.Fatalf("BootstrapFirstAdmin failed: %v", err)
		}

		u, ok := store.users[email]
		if !ok {
			t.Fatalf("expected admin user to be created")
		}
		if u.Email != email {
			t.Errorf("expected email %s, got %s", email, u.Email)
		}
		if u.Username != "admin" {
			t.Errorf("expected username 'admin', got %s", u.Username)
		}
		if u.Role != domain.RoleAdmin {
			t.Errorf("expected role 'admin', got %s", u.Role)
		}
		if !u.IsActive {
			t.Errorf("expected is_active true")
		}
		if u.PasswordHash == nil {
			t.Fatalf("expected password hash to be set")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(pass)); err != nil {
			t.Errorf("password hash verification failed: %v", err)
		}
	})

	t.Run("idempotent: does not overwrite existing admin", func(t *testing.T) {
		store := newFakeUserStore()
		email := "admin@errorium.ru"
		pass := "AdminSecurePass123!"

		_ = app.BootstrapFirstAdmin(ctx, store, email, pass)
		initialUser := store.users[email]

		// Call again with a different password
		err := app.BootstrapFirstAdmin(ctx, store, email, "DifferentPassword456!")
		if err != nil {
			t.Fatalf("second bootstrap failed: %v", err)
		}

		afterUser := store.users[email]
		if *afterUser.PasswordHash != *initialUser.PasswordHash {
			t.Errorf("expected password hash to remain unchanged")
		}
	})

	t.Run("does not overwrite regular user with same email", func(t *testing.T) {
		store := newFakeUserStore()
		email := "existing@errorium.ru"
		existingHash := "$2a$12$existingFakeHash"
		existingUser := &domain.User{
			ID:           uuid.New(),
			Email:        email,
			Username:     "existing",
			PasswordHash: &existingHash,
			Role:         domain.RoleUser,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		_ = store.Create(ctx, existingUser)

		err := app.BootstrapFirstAdmin(ctx, store, email, "NewAdminPassword!")
		if err != nil {
			t.Fatalf("bootstrap failed: %v", err)
		}

		u := store.users[email]
		if u.Role != domain.RoleUser {
			t.Errorf("expected role to remain 'user', got %s", u.Role)
		}
		if *u.PasswordHash != existingHash {
			t.Errorf("expected password hash to remain original")
		}
	})

	t.Run("empty email or password does nothing", func(t *testing.T) {
		store := newFakeUserStore()
		_ = app.BootstrapFirstAdmin(ctx, store, "", "pass")
		_ = app.BootstrapFirstAdmin(ctx, store, "admin@test.ru", "")
		if len(store.users) != 0 {
			t.Errorf("expected no users created when credentials are empty")
		}
	})
}
