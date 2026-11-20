package repository_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
)

// ── Compile-time interface checks ─────────────────────────────────────────────

var (
	_ repository.PasswordResetStore = (*repository.PasswordResetRepository)(nil)
	_ repository.SsoStateStore      = (*repository.SsoStateRepository)(nil)
	_ repository.RagConfigStore     = (*repository.RagConfigRepository)(nil)
	_ repository.SessionStore       = (*repository.SessionRepository)(nil)
	_ repository.SSOAccountStore    = (*repository.SSOAccountRepository)(nil)
	_ repository.AuditStore         = (*repository.AuditRepository)(nil)
	_ repository.UserStore          = (*repository.UserRepository)(nil)
)

// ── Mock SQL Driver ───────────────────────────────────────────────────────────

type queryRecord struct {
	query string
	args  []driver.NamedValue
}

type queryResult struct {
	cols []string
	vals [][]driver.Value
}

type mockDriver struct {
	mu           sync.Mutex
	queries      []queryRecord
	nextRowsCols []string
	nextRowsVals [][]driver.Value
	resultQueue  []queryResult
	execErr      error
	queryErr     error
}

var (
	testDriverInstance *mockDriver
	registerOnce       sync.Once
)

func getMockDriver() *mockDriver {
	registerOnce.Do(func() {
		testDriverInstance = &mockDriver{}
		sql.Register("mock_sql_driver", testDriverInstance)
	})
	return testDriverInstance
}

func (d *mockDriver) reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queries = nil
	d.nextRowsCols = nil
	d.nextRowsVals = nil
	d.resultQueue = nil
	d.execErr = nil
	d.queryErr = nil
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{d: d}, nil
}

type mockConn struct {
	d *mockDriver
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{conn: c, query: query}, nil
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{}, nil
}

func (c *mockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.d.mu.Lock()
	defer c.d.mu.Unlock()
	c.d.queries = append(c.d.queries, queryRecord{query: query, args: args})
	if c.d.execErr != nil {
		return nil, c.d.execErr
	}
	return driver.RowsAffected(1), nil
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.d.mu.Lock()
	defer c.d.mu.Unlock()
	c.d.queries = append(c.d.queries, queryRecord{query: query, args: args})
	if c.d.queryErr != nil {
		return nil, c.d.queryErr
	}
	var cols []string
	var vals [][]driver.Value
	if len(c.d.resultQueue) > 0 {
		res := c.d.resultQueue[0]
		c.d.resultQueue = c.d.resultQueue[1:]
		cols = res.cols
		vals = res.vals
	} else {
		cols = c.d.nextRowsCols
		vals = c.d.nextRowsVals
	}
	return &mockRows{cols: cols, rows: vals, idx: 0}, nil
}

type mockStmt struct {
	conn  *mockConn
	query string
}

func (s *mockStmt) Close() error  { return nil }
func (s *mockStmt) NumInput() int { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.conn.ExecContext(context.Background(), s.query, named)
}
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.conn.QueryContext(context.Background(), s.query, named)
}

type mockTx struct{}

func (t *mockTx) Commit() error   { return nil }
func (t *mockTx) Rollback() error { return nil }

type mockRows struct {
	cols []string
	rows [][]driver.Value
	idx  int
}

func (r *mockRows) Columns() []string { return r.cols }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i, val := range r.rows[r.idx] {
		dest[i] = val
	}
	r.idx++
	return nil
}

func setupMockDB(t *testing.T) (*db.DB, *mockDriver) {
	d := getMockDriver()
	d.reset()

	rawDB, err := sql.Open("mock_sql_driver", "test")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	return &db.DB{DB: rawDB}, d
}

// ── Tests for SessionRepository.RevokeByUser ─────────────────────────────────

func TestSessionRepository_RevokeByUser(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewSessionRepository(database)

	ctx := context.Background()
	userID := uuid.MustParse("019348b0-0000-7000-8000-000000000001")

	err := repo.RevokeByUser(ctx, userID)
	if err != nil {
		t.Fatalf("RevokeByUser failed: %v", err)
	}

	if len(mock.queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mock.queries))
	}

	q := mock.queries[0]
	if !strings.Contains(q.query, "UPDATE user_sessions SET is_revoked = TRUE WHERE user_id = $1") {
		t.Errorf("unexpected query: %s", q.query)
	}
	if len(q.args) != 1 || fmt.Sprintf("%v", q.args[0].Value) != userID.String() {
		t.Errorf("expected arg %v, got %v", userID, q.args)
	}
}

// ── Tests for PasswordResetRepository ────────────────────────────────────────

func TestPasswordResetRepository_Create(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewPasswordResetRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	token := &domain.PasswordResetToken{
		ID:        uuid.MustParse("019348b0-0000-7000-8000-000000000001"),
		UserID:    uuid.MustParse("019348b0-0000-7000-8000-000000000002"),
		TokenHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}

	err := repo.Create(ctx, token)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(mock.queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mock.queries))
	}
	q := mock.queries[0]
	if !strings.Contains(q.query, "INSERT INTO password_reset_tokens") {
		t.Errorf("unexpected query: %s", q.query)
	}
}

func TestPasswordResetRepository_GetByTokenHash(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewPasswordResetRepository(database)

	ctx := context.Background()
	tokenID := uuid.MustParse("019348b0-0000-7000-8000-000000000001")
	userID := uuid.MustParse("019348b0-0000-7000-8000-000000000002")
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	now := time.Now().UTC()
	expires := now.Add(time.Hour)

	mock.nextRowsCols = []string{"id", "user_id", "token_hash", "expires_at", "used_at", "created_at"}
	mock.nextRowsVals = [][]driver.Value{
		{tokenID.String(), userID.String(), hash, expires, nil, now},
	}

	tok, err := repo.GetByTokenHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByTokenHash failed: %v", err)
	}
	if tok.ID != tokenID {
		t.Errorf("expected ID %v, got %v", tokenID, tok.ID)
	}
	if tok.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, tok.UserID)
	}
	if tok.TokenHash != hash {
		t.Errorf("expected hash %v, got %v", hash, tok.TokenHash)
	}
	if tok.UsedAt != nil {
		t.Errorf("expected nil UsedAt, got %v", tok.UsedAt)
	}
}

func TestPasswordResetRepository_MarkUsed(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewPasswordResetRepository(database)

	ctx := context.Background()
	tokenID := uuid.MustParse("019348b0-0000-7000-8000-000000000001")

	err := repo.MarkUsed(ctx, tokenID)
	if err != nil {
		t.Fatalf("MarkUsed failed: %v", err)
	}

	if len(mock.queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mock.queries))
	}
	q := mock.queries[0]
	if !strings.Contains(q.query, "UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1") {
		t.Errorf("unexpected query: %s", q.query)
	}
}

func TestPasswordResetRepository_DeleteExpired(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewPasswordResetRepository(database)

	ctx := context.Background()
	err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}

	if len(mock.queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mock.queries))
	}
	q := mock.queries[0]
	if !strings.Contains(q.query, "DELETE FROM password_reset_tokens WHERE expires_at < NOW()") {
		t.Errorf("unexpected query: %s", q.query)
	}
}

// ── Tests for SsoStateRepository ─────────────────────────────────────────────

func TestSsoStateRepository_CreateAndGet(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewSsoStateRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	redirect := "http://localhost:3000/callback"
	state := &domain.SSOAuthState{
		State:      "test-state-string-xyz",
		Provider:   domain.SSOProviderGitHub,
		RedirectTo: &redirect,
		ExpiresAt:  now.Add(15 * time.Minute),
		CreatedAt:  now,
	}

	err := repo.Create(ctx, state)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(mock.queries) != 1 || !strings.Contains(mock.queries[0].query, "INSERT INTO sso_auth_states") {
		t.Errorf("expected insert query, got %v", mock.queries)
	}

	// Test Get
	mock.nextRowsCols = []string{"state", "provider", "redirect_to", "expires_at", "created_at"}
	mock.nextRowsVals = [][]driver.Value{
		{"test-state-string-xyz", "github", redirect, state.ExpiresAt, now},
	}

	got, err := repo.Get(ctx, "test-state-string-xyz")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.State != state.State || got.Provider != state.Provider {
		t.Errorf("got %v, want %v", got, state)
	}

	// Test Delete
	err = repo.Delete(ctx, "test-state-string-xyz")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

// ── Tests for RagConfigRepository ────────────────────────────────────────────

func TestRagConfigRepository_GetAndUpdate(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewRagConfigRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	baseURL := "https://api.openai.com/v1"

	mock.nextRowsCols = []string{"id", "provider", "base_url", "model", "system_prompt", "temperature", "max_tokens", "updated_at"}
	mock.nextRowsVals = [][]driver.Value{
		{1, "openai", baseURL, "gpt-4o-mini", "Ты Errorium...", 0.2, 1500, now},
	}

	cfg, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if cfg.ID != 1 || cfg.Provider != domain.LLMProviderOpenAI || cfg.Model != "gpt-4o-mini" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}

	// Test Update
	cfg.Temperature = 0.5
	cfg.MaxTokens = 2000
	err = repo.Update(ctx, cfg)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if len(mock.queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(mock.queries))
	}
	q := mock.queries[1]
	if !strings.Contains(q.query, "UPDATE rag_config") || !strings.Contains(q.query, "WHERE id = 1") {
		t.Errorf("unexpected update query: %s", q.query)
	}
}

// ── Tests for SSOAccountRepository ───────────────────────────────────────────

func TestSSOAccountRepository_CreateAndGet(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewSSOAccountRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	email := "dev@example.com"
	accID := uuid.New()
	userID := uuid.New()

	acc := &domain.UserSSOAccount{
		ID:             accID,
		UserID:         userID,
		Provider:       domain.SSOProviderGitHub,
		ProviderUserID: "123456",
		Email:          &email,
		ProfileData:    []byte(`{"login":"octocat"}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.Create(ctx, acc)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(mock.queries) != 1 || !strings.Contains(mock.queries[0].query, "INSERT INTO user_sso_accounts") {
		t.Errorf("expected insert query, got %v", mock.queries)
	}

	// Test GetByProviderAndUserID
	mock.nextRowsCols = []string{"id", "user_id", "provider", "provider_user_id", "email", "profile_data", "created_at", "updated_at"}
	mock.nextRowsVals = [][]driver.Value{
		{accID.String(), userID.String(), "github", "123456", email, []byte(`{"login":"octocat"}`), now, now},
	}

	got, err := repo.GetByProviderAndUserID(ctx, domain.SSOProviderGitHub, "123456")
	if err != nil {
		t.Fatalf("GetByProviderAndUserID failed: %v", err)
	}
	if got.ID != accID || got.UserID != userID || got.Provider != domain.SSOProviderGitHub || got.ProviderUserID != "123456" {
		t.Errorf("unexpected account got: %+v", got)
	}
	if got.Email == nil || *got.Email != email {
		t.Errorf("unexpected email: %v", got.Email)
	}
}

// ── Tests for AuditRepository ────────────────────────────────────────────────

func TestAuditRepository_CreateAndLog(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewAuditRepository(database)

	ctx := context.Background()
	userID := uuid.New()
	addr := netip.MustParseAddr("127.0.0.1")

	err := repo.Log(ctx, &userID, "admin_test", &addr, json.RawMessage(`{"foo":"bar"}`))
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if len(mock.queries) != 1 || !strings.Contains(mock.queries[0].query, "INSERT INTO audit_logs") {
		t.Errorf("expected insert query, got %v", mock.queries)
	}
}

// ── Tests for UserRepository.List and UpdateRoleAndStatus ─────────────────────

func TestUserRepository_List(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewUserRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	uID := uuid.New()

	// Enqueue count result and rows result
	mock.resultQueue = []queryResult{
		{
			cols: []string{"count"},
			vals: [][]driver.Value{{1}},
		},
		{
			cols: []string{
				"id", "email", "username", "avatar_url", "password_hash",
				"role", "profession", "gender", "language", "is_active",
				"created_at", "updated_at", "last_login_at",
			},
			vals: [][]driver.Value{
				{uID.String(), "user@example.com", "user", nil, nil, "user", "Developer", "Not specified", "English", true, now, now, nil},
			},
		},
	}

	items, total, err := repo.List(ctx, 20, 0, "user")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Errorf("unexpected list result: total=%d, items=%d", total, len(items))
	}
	if items[0].Email != "user@example.com" {
		t.Errorf("unexpected user: %v", items[0])
	}
}

func TestUserRepository_UpdateRoleAndStatus(t *testing.T) {
	database, mock := setupMockDB(t)
	repo := repository.NewUserRepository(database)

	ctx := context.Background()
	now := time.Now().UTC()
	uID := uuid.New()

	mock.nextRowsCols = []string{
		"id", "email", "username", "avatar_url", "password_hash",
		"role", "profession", "gender", "language", "is_active",
		"created_at", "updated_at", "last_login_at",
	}
	mock.nextRowsVals = [][]driver.Value{
		{uID.String(), "user@example.com", "user", nil, nil, "admin", "Developer", "Not specified", "English", false, now, now, nil},
	}

	newRole := domain.RoleAdmin
	newActive := false
	updated, err := repo.UpdateRoleAndStatus(ctx, uID, &newRole, &newActive)
	if err != nil {
		t.Fatalf("UpdateRoleAndStatus failed: %v", err)
	}
	if updated.Role != domain.RoleAdmin || updated.IsActive != false {
		t.Errorf("unexpected updated user: %+v", updated)
	}
}
