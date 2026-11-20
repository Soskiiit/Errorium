package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/metrics"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	searchpb "github.com/soskiiit/errorium/services/gateway/internal/pb/search"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// ── Mock Core Client ──────────────────────────────────────────────────────────

type mockCoreAdminClient struct {
	statsResp  *searchpb.GetStatsResponse
	statsErr   error
	reloadResp *searchpb.ReloadIndexResponse
	reloadErr  error
}

func (m *mockCoreAdminClient) GetStats(ctx context.Context) (*searchpb.GetStatsResponse, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.statsResp, nil
}

func (m *mockCoreAdminClient) ReloadIndex(ctx context.Context) (*searchpb.ReloadIndexResponse, error) {
	if m.reloadErr != nil {
		return nil, m.reloadErr
	}
	return m.reloadResp, nil
}

// ── Mock User Store for Admin ─────────────────────────────────────────────────

type mockAdminUserStore struct {
	users map[uuid.UUID]*domain.User
}

func newMockAdminUserStore() *mockAdminUserStore {
	return &mockAdminUserStore{
		users: make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockAdminUserStore) Create(ctx context.Context, u *domain.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockAdminUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

func (m *mockAdminUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockAdminUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockAdminUserStore) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if u, ok := m.users[id]; ok {
		u.PasswordHash = &hash
		return nil
	}
	return sql.ErrNoRows
}

func (m *mockAdminUserStore) UpdateLastLogin(ctx context.Context, id uuid.UUID, t time.Time) error {
	return nil
}

func (m *mockAdminUserStore) List(ctx context.Context, limit, offset int, search string) ([]*domain.User, int, error) {
	var matched []*domain.User
	searchLower := strings.ToLower(search)
	for _, u := range m.users {
		if searchLower == "" || strings.Contains(strings.ToLower(u.Email), searchLower) || strings.Contains(strings.ToLower(u.Username), searchLower) {
			matched = append(matched, u)
		}
	}
	total := len(matched)
	if offset >= total {
		return []*domain.User{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matched[offset:end], total, nil
}

func (m *mockAdminUserStore) UpdateRoleAndStatus(ctx context.Context, id uuid.UUID, role *domain.Role, isActive *bool) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	if role != nil {
		u.Role = *role
	}
	if isActive != nil {
		u.IsActive = *isActive
	}
	u.UpdatedAt = time.Now()
	return u, nil
}

// ── Mock Rag Config Store ─────────────────────────────────────────────────────

type mockAdminRagConfigStore struct {
	cfg *domain.RagConfig
}

func (m *mockAdminRagConfigStore) Get(ctx context.Context) (*domain.RagConfig, error) {
	return m.cfg, nil
}

func (m *mockAdminRagConfigStore) Update(ctx context.Context, cfg *domain.RagConfig) error {
	m.cfg = cfg
	return nil
}

// ── Mock Audit Store ──────────────────────────────────────────────────────────

type auditEntry struct {
	UserID   *uuid.UUID
	Action   string
	ClientIP *netip.Addr
	Details  string
}

type mockAuditStore struct {
	entries []auditEntry
}

func (m *mockAuditStore) Create(ctx context.Context, l *domain.AuditLog) error {
	m.entries = append(m.entries, auditEntry{
		UserID:   l.UserID,
		Action:   l.Action,
		ClientIP: l.ClientIP,
		Details:  string(l.Details),
	})
	return nil
}

func (m *mockAuditStore) Log(ctx context.Context, userID *uuid.UUID, action string, clientIP *netip.Addr, details json.RawMessage) error {
	return m.Create(ctx, &domain.AuditLog{
		UserID:   userID,
		Action:   action,
		ClientIP: clientIP,
		Details:  details,
	})
}

// ── Setup helper ──────────────────────────────────────────────────────────────

type testAdminEnv struct {
	mux         *http.ServeMux
	jwtManager  *jwtpkg.Manager
	coreClient  *mockCoreAdminClient
	metrics     *metrics.Metrics
	userStore   *mockAdminUserStore
	ragStore    *mockAdminRagConfigStore
	auditStore  *mockAuditStore
	adminUser   *domain.User
	regularUser *domain.User
}

func setupAdminTestEnv() *testAdminEnv {
	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:            "test-secret-key-at-least-32-chars-long!",
		AccessTokenExpireMin: 60,
	})

	coreClient := &mockCoreAdminClient{
		statsResp: &searchpb.GetStatsResponse{
			TotalDocuments: 1000,
			AvgDocLength:   50.5,
			VocabularySize: 500,
			IndexSizeBytes: 10240,
			IsReady:        true,
			BuildTimestamp: 1750000000,
		},
		reloadResp: &searchpb.ReloadIndexResponse{
			Ok:             true,
			BuildTimestamp: 1750001111,
		},
	}

	metricsInst := metrics.New()
	userStore := newMockAdminUserStore()
	ragStore := &mockAdminRagConfigStore{
		cfg: &domain.RagConfig{
			ID:           1,
			Provider:     domain.LLMProviderStub,
			Model:        "",
			SystemPrompt: "Default prompt",
			Temperature:  0.2,
			MaxTokens:    1500,
		},
	}
	auditStore := &mockAuditStore{}

	adminID := uuid.New()
	adminUser := &domain.User{
		ID:        adminID,
		Email:     "admin@errorium.ru",
		Username:  "admin",
		Role:      domain.RoleAdmin,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = userStore.Create(context.Background(), adminUser)

	regID := uuid.New()
	regUser := &domain.User{
		ID:        regID,
		Email:     "user@errorium.ru",
		Username:  "user",
		Role:      domain.RoleUser,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = userStore.Create(context.Background(), regUser)

	h := handler.NewAdminHandler(coreClient, metricsInst, userStore, ragStore, auditStore)
	mux := http.NewServeMux()

	authMW := middleware.Auth(jwtMgr)
	adminMW := func(next http.Handler) http.Handler {
		return authMW(middleware.RequireAdmin(next))
	}
	h.RegisterRoutes(mux, adminMW)

	return &testAdminEnv{
		mux:         mux,
		jwtManager:  jwtMgr,
		coreClient:  coreClient,
		metrics:     metricsInst,
		userStore:   userStore,
		ragStore:    ragStore,
		auditStore:  auditStore,
		adminUser:   adminUser,
		regularUser: regUser,
	}
}

func (e *testAdminEnv) makeToken(u *domain.User) string {
	token, _ := e.jwtManager.GenerateAccess(u)
	return token
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAdminHandler_RBACMatrix(t *testing.T) {
	env := setupAdminTestEnv()
	adminToken := env.makeToken(env.adminUser)
	userToken := env.makeToken(env.regularUser)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/admin/stats", ""},
		{http.MethodPost, "/api/v1/admin/index/reload", ""},
		{http.MethodGet, "/api/v1/admin/users", ""},
		{http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%s", env.regularUser.ID), `{"role":"admin"}`},
		{http.MethodGet, "/api/v1/admin/rag/config", ""},
		{http.MethodPut, "/api/v1/admin/rag/config", `{"provider":"stub","temperature":0.5,"max_tokens":1000}`},
	}

	for _, ep := range endpoints {
		t.Run(fmt.Sprintf("%s %s anonymous -> 401", ep.method, ep.path), func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			env.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for anonymous, got %d", rec.Code)
			}
		})

		t.Run(fmt.Sprintf("%s %s regular user -> 403", ep.method, ep.path), func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Authorization", "Bearer "+userToken)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			env.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("expected 403 for regular user, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "forbidden: admin role required") {
				t.Errorf("expected forbidden message, got %s", rec.Body.String())
			}
		})

		t.Run(fmt.Sprintf("%s %s admin user -> 200", ep.method, ep.path), func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			env.mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 for admin user, got %d, body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminHandler_Stats(t *testing.T) {
	env := setupAdminTestEnv()
	adminToken := env.makeToken(env.adminUser)

	t.Run("empty metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var stats struct {
			TotalDocuments int64   `json:"total_documents"`
			AvgDocLength   float64 `json:"avg_doc_length"`
			VocabularySize int64   `json:"vocabulary_size"`
			IndexSizeBytes int64   `json:"index_size_bytes"`
			IsReady        bool    `json:"is_ready"`
			LatencyMs      struct {
				P50 float64 `json:"p50"`
				P95 float64 `json:"p95"`
				P99 float64 `json:"p99"`
			} `json:"latency_ms"`
			QueriesTotal int64 `json:"queries_total"`
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if stats.TotalDocuments != 1000 || stats.AvgDocLength != 50.5 || stats.VocabularySize != 500 || stats.IndexSizeBytes != 10240 || !stats.IsReady {
			t.Errorf("unexpected core stats: %+v", stats)
		}
		if stats.LatencyMs.P50 != 0 || stats.LatencyMs.P95 != 0 || stats.LatencyMs.P99 != 0 || stats.QueriesTotal != 0 {
			t.Errorf("expected 0 for empty metrics, got %+v", stats.LatencyMs)
		}
	})

	t.Run("with recorded percentiles", func(t *testing.T) {
		for i := 1; i <= 100; i++ {
			env.metrics.Record(float64(i))
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		lat := resp["latency_ms"].(map[string]interface{})

		if lat["p50"].(float64) != 50 || lat["p95"].(float64) != 95 || lat["p99"].(float64) != 99 {
			t.Errorf("unexpected latency percentiles: %+v", lat)
		}
		if int64(resp["queries_total"].(float64)) != 100 {
			t.Errorf("expected queries_total 100, got %v", resp["queries_total"])
		}
	})

	t.Run("core unavailable -> 502", func(t *testing.T) {
		env.coreClient.statsErr = errors.New("connection refused")
		defer func() { env.coreClient.statsErr = nil }()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "search backend unavailable") {
			t.Errorf("unexpected error: %s", rec.Body.String())
		}
	})
}

func TestAdminHandler_ReloadIndex(t *testing.T) {
	env := setupAdminTestEnv()
	adminToken := env.makeToken(env.adminUser)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/index/reload", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("X-Real-IP", "192.168.1.10")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Ok             bool  `json:"ok"`
			BuildTimestamp int64 `json:"build_timestamp"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Ok || resp.BuildTimestamp != 1750001111 {
			t.Errorf("unexpected reload response: %+v", resp)
		}

		// Verify audit
		if len(env.auditStore.entries) == 0 {
			t.Fatal("expected audit entry recorded")
		}
		last := env.auditStore.entries[len(env.auditStore.entries)-1]
		if last.Action != "admin_index_reload" || *last.UserID != env.adminUser.ID {
			t.Errorf("unexpected audit log: %+v", last)
		}
		if last.ClientIP == nil || last.ClientIP.String() != "192.168.1.10" {
			t.Errorf("unexpected audit client ip: %v", last.ClientIP)
		}
	})

	t.Run("core reload ok=false -> 502", func(t *testing.T) {
		env.coreClient.reloadResp = &searchpb.ReloadIndexResponse{Ok: false}
		defer func() {
			env.coreClient.reloadResp = &searchpb.ReloadIndexResponse{Ok: true, BuildTimestamp: 1750001111}
		}()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/index/reload", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", rec.Code)
		}
	})

	t.Run("core reload error -> 502", func(t *testing.T) {
		env.coreClient.reloadErr = errors.New("rpc failed")
		defer func() { env.coreClient.reloadErr = nil }()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/index/reload", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", rec.Code)
		}
	})
}

func TestAdminHandler_Users(t *testing.T) {
	env := setupAdminTestEnv()
	adminToken := env.makeToken(env.adminUser)

	// Add more users for pagination & search
	for i := 1; i <= 25; i++ {
		u := &domain.User{
			ID:        uuid.New(),
			Email:     fmt.Sprintf("user%02d@example.com", i),
			Username:  fmt.Sprintf("user%02d", i),
			Role:      domain.RoleUser,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = env.userStore.Create(context.Background(), u)
	}

	t.Run("list with default pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Items []*domain.User `json:"items"`
			Total int            `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Total != 27 { // 25 + 2 initial
			t.Errorf("expected total 27, got %d", resp.Total)
		}
		if len(resp.Items) != 20 { // default limit 20
			t.Errorf("expected 20 items, got %d", len(resp.Items))
		}
	})

	t.Run("list with search and custom pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=5&offset=0&search=user0", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Items []*domain.User `json:"items"`
			Total int            `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		// user01..user09 -> 9 users match
		if resp.Total != 9 {
			t.Errorf("expected 9 matched users, got %d", resp.Total)
		}
		if len(resp.Items) != 5 {
			t.Errorf("expected 5 items, got %d", len(resp.Items))
		}
	})

	t.Run("patch self -> 400 forbidden modification", func(t *testing.T) {
		body := `{"role":"user"}`
		path := fmt.Sprintf("/api/v1/admin/users/%s", env.adminUser.ID)
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 when modifying own account, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "cannot modify own admin account") {
			t.Errorf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("patch other user -> 200 + audit", func(t *testing.T) {
		body := `{"role":"admin","is_active":false}`
		path := fmt.Sprintf("/api/v1/admin/users/%s", env.regularUser.ID)
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var updated domain.User
		_ = json.Unmarshal(rec.Body.Bytes(), &updated)
		if updated.Role != domain.RoleAdmin || updated.IsActive != false {
			t.Errorf("unexpected updated user: %+v", updated)
		}

		// Verify audit
		last := env.auditStore.entries[len(env.auditStore.entries)-1]
		if last.Action != "admin_user_patch" {
			t.Errorf("expected audit action admin_user_patch, got %s", last.Action)
		}
		if !strings.Contains(last.Details, env.regularUser.ID.String()) {
			t.Errorf("expected details to contain target id, got %s", last.Details)
		}
	})

	t.Run("patch non-existent user -> 404", func(t *testing.T) {
		randomID := uuid.New()
		body := `{"role":"admin"}`
		path := fmt.Sprintf("/api/v1/admin/users/%s", randomID)
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "user not found") {
			t.Errorf("unexpected error message: %s", rec.Body.String())
		}
	})

	t.Run("patch with invalid role -> 400", func(t *testing.T) {
		body := `{"role":"superadmin"}`
		path := fmt.Sprintf("/api/v1/admin/users/%s", env.regularUser.ID)
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid role, got %d", rec.Code)
		}
	})
}

func TestAdminHandler_RagConfig(t *testing.T) {
	env := setupAdminTestEnv()
	adminToken := env.makeToken(env.adminUser)

	t.Run("GET config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/rag/config", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var cfg domain.RagConfig
		_ = json.Unmarshal(rec.Body.Bytes(), &cfg)
		if cfg.Provider != domain.LLMProviderStub {
			t.Errorf("expected stub provider, got %s", cfg.Provider)
		}
	})

	t.Run("PUT valid openai config", func(t *testing.T) {
		body := `{
			"provider": "openai",
			"base_url": "https://api.openai.com/v1",
			"model": "gpt-4o",
			"system_prompt": "Custom system prompt",
			"temperature": 0.7,
			"max_tokens": 2048
		}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var updated domain.RagConfig
		_ = json.Unmarshal(rec.Body.Bytes(), &updated)
		if updated.Provider != domain.LLMProviderOpenAI || updated.Model != "gpt-4o" || updated.Temperature != 0.7 {
			t.Errorf("unexpected updated config: %+v", updated)
		}

		// Verify audit
		last := env.auditStore.entries[len(env.auditStore.entries)-1]
		if last.Action != "admin_rag_config_update" {
			t.Errorf("expected audit action admin_rag_config_update, got %s", last.Action)
		}
	})

	t.Run("PUT invalid provider -> 400", func(t *testing.T) {
		body := `{"provider":"claude","model":"claude-3","temperature":0.5,"max_tokens":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT temperature > 2 -> 400", func(t *testing.T) {
		body := `{"provider":"stub","temperature":2.5,"max_tokens":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT temperature < 0 -> 400", func(t *testing.T) {
		body := `{"provider":"stub","temperature":-0.1,"max_tokens":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT max_tokens < 1 -> 400", func(t *testing.T) {
		body := `{"provider":"stub","temperature":0.5,"max_tokens":0}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT non-stub with empty model -> 400", func(t *testing.T) {
		body := `{"provider":"openai","base_url":"https://api.openai.com/v1","model":"","temperature":0.5,"max_tokens":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT openai with invalid base_url -> 400", func(t *testing.T) {
		body := `{"provider":"openai","base_url":"not-a-valid-url","model":"gpt-4o","temperature":0.5,"max_tokens":1000}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PUT stub with empty model and nil base_url -> 200", func(t *testing.T) {
		body := `{"provider":"stub","temperature":0.2,"max_tokens":1500}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/rag/config", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
		}
	})
}
