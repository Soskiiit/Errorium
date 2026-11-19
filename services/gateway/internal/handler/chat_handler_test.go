package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// ---------------------------------------------------------------------------
// In-memory fakes for ChatStore, TagStore, SearchClient
// ---------------------------------------------------------------------------

type testChatStore struct {
	mu       sync.RWMutex
	chats    map[uuid.UUID]*domain.Chat
	messages map[uuid.UUID][]*domain.ChatMessage
}

func newTestChatStore() *testChatStore {
	return &testChatStore{
		chats:    make(map[uuid.UUID]*domain.Chat),
		messages: make(map[uuid.UUID][]*domain.ChatMessage),
	}
}

func (s *testChatStore) Create(ctx context.Context, c *domain.Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *c
	s.chats[c.ID] = &cp
	return nil
}

func (s *testChatStore) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, search string) ([]*domain.Chat, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matching []*domain.Chat
	searchLower := strings.ToLower(search)
	for _, c := range s.chats {
		if c.UserID == userID {
			if search == "" || strings.Contains(strings.ToLower(c.Title), searchLower) {
				cp := *c
				matching = append(matching, &cp)
			}
		}
	}

	// Sort: is_pinned DESC, updated_at DESC
	sort.Slice(matching, func(i, j int) bool {
		if matching[i].IsPinned != matching[j].IsPinned {
			return matching[i].IsPinned
		}
		return matching[i].UpdatedAt.After(matching[j].UpdatedAt)
	})

	total := len(matching)
	if offset >= total {
		return []*domain.Chat{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return matching[offset:end], total, nil
}

func (s *testChatStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.chats[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *c
	return &cp, nil
}

func (s *testChatStore) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.chats[id]
	if !ok || c.UserID != userID {
		return nil, sql.ErrNoRows
	}
	cp := *c
	return &cp, nil
}

func (s *testChatStore) Update(ctx context.Context, c *domain.Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[c.ID]; !ok {
		return sql.ErrNoRows
	}
	cp := *c
	s.chats[c.ID] = &cp
	return nil
}

func (s *testChatStore) Delete(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.chats, id)
	delete(s.messages, id)
	return nil
}

func (s *testChatStore) AddMessage(ctx context.Context, msg *domain.ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *msg
	s.messages[msg.ChatID] = append(s.messages[msg.ChatID], &cp)
	return nil
}

func (s *testChatStore) GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*domain.ChatMessage
	for _, m := range s.messages[chatID] {
		cp := *m
		res = append(res, &cp)
	}
	return res, nil
}

type testTagStore struct {
	mu          sync.RWMutex
	tags        map[string]*domain.Tag
	messageTags map[uuid.UUID][]string
}

func newTestTagStore() *testTagStore {
	return &testTagStore{
		tags:        make(map[string]*domain.Tag),
		messageTags: make(map[uuid.UUID][]string),
	}
}

func (t *testTagStore) GetOrCreate(ctx context.Context, name string) (*domain.Tag, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	lower := strings.ToLower(name)
	if existing, ok := t.tags[lower]; ok {
		return existing, nil
	}
	tag := &domain.Tag{
		ID:        uuid.New(),
		Name:      lower,
		CreatedAt: time.Now().UTC(),
	}
	t.tags[lower] = tag
	return tag, nil
}

func (t *testTagStore) AttachToMessage(ctx context.Context, chatMessageID uuid.UUID, tagNames []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, name := range tagNames {
		lower := strings.ToLower(name)
		if _, ok := t.tags[lower]; !ok {
			t.tags[lower] = &domain.Tag{
				ID:        uuid.New(),
				Name:      lower,
				CreatedAt: time.Now().UTC(),
			}
		}
		t.messageTags[chatMessageID] = append(t.messageTags[chatMessageID], lower)
	}
	return nil
}

func (t *testTagStore) GetByMessage(ctx context.Context, chatMessageID uuid.UUID) ([]*domain.Tag, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	names := t.messageTags[chatMessageID]
	var res []*domain.Tag
	seen := make(map[string]bool)
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			if tag, ok := t.tags[name]; ok {
				cp := *tag
				res = append(res, &cp)
			}
		}
	}
	// Sort by name ASC
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res, nil
}

type testSearchClient struct {
	delegate   search.SearchClient
	failSearch bool
	failRAG    bool
	mu         sync.Mutex
	lastReq    search.SearchRequest
}

func newTestSearchClient() *testSearchClient {
	return &testSearchClient{
		delegate: search.NewStub(),
	}
}

func (c *testSearchClient) Search(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
	c.mu.Lock()
	c.lastReq = req
	fail := c.failSearch
	c.mu.Unlock()

	if fail {
		return nil, errors.New("search backend unavailable")
	}
	return c.delegate.Search(ctx, req)
}

func (c *testSearchClient) RAG(ctx context.Context, req search.SearchRequest) (*search.RAGResult, error) {
	c.mu.Lock()
	c.lastReq = req
	fail := c.failRAG
	c.mu.Unlock()

	if fail {
		return nil, errors.New("search backend unavailable")
	}
	return c.delegate.RAG(ctx, req)
}

// ---------------------------------------------------------------------------
// Test harness helpers
// ---------------------------------------------------------------------------

type testHarness struct {
	chatStore    *testChatStore
	tagStore     *testTagStore
	searchClient *testSearchClient
	jwtMgr       *jwtpkg.Manager
	mux          *http.ServeMux
	userA        *domain.User
	tokenA       string
	userB        *domain.User
	tokenB       string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:            "super-secret-gateway-test-key-32-bytes!!",
		AccessTokenExpireMin: 15,
	})
	authMW := middleware.Auth(jwtMgr)

	chats := newTestChatStore()
	tags := newTestTagStore()
	searchCli := newTestSearchClient()

	chatHandler := handler.NewChatHandler(chats, tags, searchCli)
	mux := http.NewServeMux()
	chatHandler.RegisterRoutes(mux, authMW)

	userA := &domain.User{
		ID:       uuid.New(),
		Email:    "usera@example.com",
		Username: "usera",
		Role:     domain.RoleUser,
	}
	tokenA, err := jwtMgr.GenerateAccess(userA)
	if err != nil {
		t.Fatalf("failed to generate access token A: %v", err)
	}

	userB := &domain.User{
		ID:       uuid.New(),
		Email:    "userb@example.com",
		Username: "userb",
		Role:     domain.RoleUser,
	}
	tokenB, err := jwtMgr.GenerateAccess(userB)
	if err != nil {
		t.Fatalf("failed to generate access token B: %v", err)
	}

	return &testHarness{
		chatStore:    chats,
		tagStore:     tags,
		searchClient: searchCli,
		jwtMgr:       jwtMgr,
		mux:          mux,
		userA:        userA,
		tokenA:       tokenA,
		userB:        userB,
		tokenB:       tokenB,
	}
}

func (h *testHarness) request(method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.mux.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// 1. Ownership Matrix Tests
// ---------------------------------------------------------------------------

func TestChatHandler_OwnershipMatrix(t *testing.T) {
	h := newTestHarness(t)

	// User A creates a chat
	now := time.Now().UTC()
	chatA := &domain.Chat{
		ID:        uuid.New(),
		UserID:    h.userA.ID,
		Title:     "Chat Owned By User A",
		IsPinned:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.chatStore.Create(context.Background(), chatA); err != nil {
		t.Fatalf("failed to insert test chat: %v", err)
	}

	randomID := uuid.New()

	endpoints := []struct {
		name       string
		method     string
		pathForA   string
		pathForRnd string
		body       interface{}
	}{
		{
			name:       "GetChat",
			method:     http.MethodGet,
			pathForA:   "/api/v1/chats/" + chatA.ID.String(),
			pathForRnd: "/api/v1/chats/" + randomID.String(),
			body:       nil,
		},
		{
			name:       "UpdateChat",
			method:     http.MethodPatch,
			pathForA:   "/api/v1/chats/" + chatA.ID.String(),
			pathForRnd: "/api/v1/chats/" + randomID.String(),
			body:       map[string]interface{}{"title": "Renamed Title"},
		},
		{
			name:       "SendMessage",
			method:     http.MethodPost,
			pathForA:   "/api/v1/chats/" + chatA.ID.String() + "/messages",
			pathForRnd: "/api/v1/chats/" + randomID.String() + "/messages",
			body:       map[string]interface{}{"query": "[Python] IndexError"},
		},
		{
			name:       "DeleteChat",
			method:     http.MethodDelete,
			pathForA:   "/api/v1/chats/" + chatA.ID.String(),
			pathForRnd: "/api/v1/chats/" + randomID.String(),
			body:       nil,
		},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+"_UserB_Returns404ConcealingExistence", func(t *testing.T) {
			rec := h.request(ep.method, ep.pathForA, h.tokenB, ep.body)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for User B accessing User A chat, got %d; body: %s", rec.Code, rec.Body.String())
			}
			var errResp map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "chat not found" {
				t.Fatalf("expected error 'chat not found', got body: %s", rec.Body.String())
			}
		})

		t.Run(ep.name+"_NonExistentChat_Returns404", func(t *testing.T) {
			rec := h.request(ep.method, ep.pathForRnd, h.tokenA, ep.body)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for non-existent chat, got %d; body: %s", rec.Code, rec.Body.String())
			}
			var errResp map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil || errResp["error"] != "chat not found" {
				t.Fatalf("expected error 'chat not found', got body: %s", rec.Body.String())
			}
		})
	}

	// Verify User A can access their own chat across all methods
	t.Run("UserA_CanGetTheirOwnChat", func(t *testing.T) {
		rec := h.request(http.MethodGet, "/api/v1/chats/"+chatA.ID.String(), h.tokenA, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for User A, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("UserA_CanUpdateTheirOwnChat", func(t *testing.T) {
		rec := h.request(http.MethodPatch, "/api/v1/chats/"+chatA.ID.String(), h.tokenA, map[string]interface{}{
			"title": "Updated By Owner",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for User A update, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("UserA_CanSendMessageToTheirOwnChat", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chatA.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "IndexError: list index out of range",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 for User A send message, got %d; body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("UserA_CanDeleteTheirOwnChat", func(t *testing.T) {
		rec := h.request(http.MethodDelete, "/api/v1/chats/"+chatA.ID.String(), h.tokenA, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for User A delete, got %d; body: %s", rec.Code, rec.Body.String())
		}
		// Confirm it's gone
		getRec := h.request(http.MethodGet, "/api/v1/chats/"+chatA.ID.String(), h.tokenA, nil)
		if getRec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after deletion, got %d", getRec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// 2. Bracketed Tag Extraction & Clean Query Tests
// ---------------------------------------------------------------------------

func TestChatHandler_BracketedTagExtraction(t *testing.T) {
	h := newTestHarness(t)

	chat := &domain.Chat{
		ID:        uuid.New(),
		UserID:    h.userA.ID,
		Title:     "Новый чат",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = h.chatStore.Create(context.Background(), chat)

	t.Run("SingleTagAndCleanQuery", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Python] TypeError: x",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			UserMessage domain.ChatMessage `json:"user_message"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if resp.UserMessage.Content == nil || *resp.UserMessage.Content != "[Python] TypeError: x" {
			t.Fatalf("user message content should preserve raw query with brackets, got %v", resp.UserMessage.Content)
		}

		if len(resp.UserMessage.Tags) != 1 || resp.UserMessage.Tags[0].Name != "python" {
			t.Fatalf("expected tag 'python', got %v", resp.UserMessage.Tags)
		}

		// Check search client received cleanQuery "TypeError: x"
		if h.searchClient.lastReq.Query != "TypeError: x" {
			t.Fatalf("expected search query 'TypeError: x', got %q", h.searchClient.lastReq.Query)
		}
	})

	t.Run("MultipleTags_CaseInsensitivity_Deduplication", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[C++] [c++] [GCC]   undefined reference to main   ",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			UserMessage domain.ChatMessage `json:"user_message"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		expectedTags := map[string]bool{"c++": true, "gcc": true}
		if len(resp.UserMessage.Tags) != 2 {
			t.Fatalf("expected 2 unique tags, got %d: %v", len(resp.UserMessage.Tags), resp.UserMessage.Tags)
		}
		for _, tag := range resp.UserMessage.Tags {
			if !expectedTags[tag.Name] {
				t.Fatalf("unexpected tag: %s", tag.Name)
			}
		}

		if h.searchClient.lastReq.Query != "undefined reference to main" {
			t.Fatalf("expected cleanQuery 'undefined reference to main', got %q", h.searchClient.lastReq.Query)
		}
	})

	t.Run("MergeWithExplicitTags", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Python] [Django] OperationalError",
			"tags":  []string{"django", "postgresql", "Python"},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			UserMessage domain.ChatMessage `json:"user_message"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		expectedTags := map[string]bool{"python": true, "django": true, "postgresql": true}
		if len(resp.UserMessage.Tags) != 3 {
			t.Fatalf("expected 3 unique tags, got %d: %v", len(resp.UserMessage.Tags), resp.UserMessage.Tags)
		}
		for _, tag := range resp.UserMessage.Tags {
			if !expectedTags[tag.Name] {
				t.Fatalf("unexpected tag %s", tag.Name)
			}
		}

		if h.searchClient.lastReq.Query != "OperationalError" {
			t.Fatalf("expected cleanQuery 'OperationalError', got %q", h.searchClient.lastReq.Query)
		}
	})

	t.Run("EmptyCleanQueryReturns400", func(t *testing.T) {
		cases := []string{
			"[Python]",
			"[Python] [Django]",
			"[tag]   ",
			"   ",
		}
		for _, q := range cases {
			rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
				"query": q,
			})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for query %q, got %d: %s", q, rec.Code, rec.Body.String())
			}
			var errResp map[string]string
			_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
			if errResp["error"] != "clean query is empty" {
				t.Fatalf("expected error 'clean query is empty', got %q", errResp["error"])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// 3. Auto-Titling Tests
// ---------------------------------------------------------------------------

func TestChatHandler_AutoTitling(t *testing.T) {
	h := newTestHarness(t)

	t.Run("AutoTitleOnNewChat_ShortQuery", func(t *testing.T) {
		chat := &domain.Chat{
			ID:        uuid.New(),
			UserID:    h.userA.ID,
			Title:     "Новый чат",
			CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-10 * time.Minute),
		}
		_ = h.chatStore.Create(context.Background(), chat)

		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Python] IndexError: list index out of range",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		updatedChat, err := h.chatStore.GetByID(context.Background(), chat.ID)
		if err != nil {
			t.Fatalf("failed to fetch chat: %v", err)
		}

		if updatedChat.Title != "IndexError: list index out of range" {
			t.Fatalf("expected title 'IndexError: list index out of range', got %q", updatedChat.Title)
		}
		if !updatedChat.UpdatedAt.After(chat.UpdatedAt) {
			t.Fatalf("expected UpdatedAt to be updated, got %v vs original %v", updatedChat.UpdatedAt, chat.UpdatedAt)
		}
	})

	t.Run("AutoTitleOnNewChat_80RunesMultiByteUTF8", func(t *testing.T) {
		chat := &domain.Chat{
			ID:        uuid.New(),
			UserID:    h.userA.ID,
			Title:     "Новый чат",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_ = h.chatStore.Create(context.Background(), chat)

		// 100 Cyrillic characters: each Cyrillic rune is 2 bytes in UTF-8
		// 80 runes = 160 bytes; taking raw byte slicing would corrupt characters
		longCyrillic := strings.Repeat("абвгд", 20) // 100 runes
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Тест] " + longCyrillic,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		updatedChat, err := h.chatStore.GetByID(context.Background(), chat.ID)
		if err != nil {
			t.Fatalf("failed to fetch chat: %v", err)
		}

		runes := []rune(updatedChat.Title)
		if len(runes) != 80 {
			t.Fatalf("expected exactly 80 runes, got %d", len(runes))
		}
		expectedTitle := string([]rune(longCyrillic)[:80])
		if updatedChat.Title != expectedTitle {
			t.Fatalf("expected title %q, got %q", expectedTitle, updatedChat.Title)
		}
	})

	t.Run("CustomTitlePreserved_NotOverwritten", func(t *testing.T) {
		customTitle := "Мой особенный диалог"
		chat := &domain.Chat{
			ID:        uuid.New(),
			UserID:    h.userA.ID,
			Title:     customTitle,
			CreatedAt: time.Now().UTC().Add(-5 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-5 * time.Minute),
		}
		_ = h.chatStore.Create(context.Background(), chat)

		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Python] ZeroDivisionError",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		updatedChat, err := h.chatStore.GetByID(context.Background(), chat.ID)
		if err != nil {
			t.Fatalf("failed to fetch chat: %v", err)
		}

		if updatedChat.Title != customTitle {
			t.Fatalf("custom title should be preserved, got %q", updatedChat.Title)
		}
		if !updatedChat.UpdatedAt.After(chat.UpdatedAt) {
			t.Fatalf("expected UpdatedAt to be updated, got %v vs original %v", updatedChat.UpdatedAt, chat.UpdatedAt)
		}
	})
}

// ---------------------------------------------------------------------------
// 4. Message Pipeline & Search Failure (502) Tests
// ---------------------------------------------------------------------------

func TestChatHandler_MessagePipelineAndSearchBackend(t *testing.T) {
	h := newTestHarness(t)

	chat := &domain.Chat{
		ID:        uuid.New(),
		UserID:    h.userA.ID,
		Title:     "Новый чат",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = h.chatStore.Create(context.Background(), chat)

	t.Run("SuccessfulMessagePipeline", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "[Python] SyntaxError: invalid syntax",
			"top_k": 3,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			UserMessage      domain.ChatMessage `json:"user_message"`
			AssistantMessage domain.ChatMessage `json:"assistant_message"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// User message validation
		if resp.UserMessage.SenderRole != domain.SenderRoleUser {
			t.Fatalf("expected sender_role 'user', got %s", resp.UserMessage.SenderRole)
		}
		if resp.UserMessage.MessageType != domain.MessageTypeErrorReport {
			t.Fatalf("expected message_type 'error_report', got %s", resp.UserMessage.MessageType)
		}
		if resp.UserMessage.Content == nil || *resp.UserMessage.Content != "[Python] SyntaxError: invalid syntax" {
			t.Fatalf("expected content to be raw query, got %v", resp.UserMessage.Content)
		}
		if resp.UserMessage.ErrorContext == nil {
			t.Fatalf("expected error_context to be present")
		}
		var errCtx domain.ErrorContext
		if err := json.Unmarshal(*resp.UserMessage.ErrorContext, &errCtx); err != nil {
			t.Fatalf("failed to parse error_context: %v", err)
		}
		if errCtx.RawQuery != "[Python] SyntaxError: invalid syntax" {
			t.Fatalf("expected error_context.raw_query to match raw query, got %q", errCtx.RawQuery)
		}

		// Assistant message validation
		if resp.AssistantMessage.SenderRole != domain.SenderRoleAssistant {
			t.Fatalf("expected sender_role 'assistant', got %s", resp.AssistantMessage.SenderRole)
		}
		if resp.AssistantMessage.MessageType != domain.MessageTypeRAGSolution {
			t.Fatalf("expected message_type 'rag_solution', got %s", resp.AssistantMessage.MessageType)
		}
		if resp.AssistantMessage.Content == nil || *resp.AssistantMessage.Content == "" {
			t.Fatalf("expected non-empty synthesis content in assistant message")
		}
		if resp.AssistantMessage.RAGSources == nil {
			t.Fatalf("expected rag_sources to be populated")
		}
		var sources []search.Source
		if err := json.Unmarshal(*resp.AssistantMessage.RAGSources, &sources); err != nil {
			t.Fatalf("failed to unmarshal rag_sources: %v", err)
		}
		if len(sources) == 0 {
			t.Fatalf("expected at least one RAG source")
		}

		// Verify both messages persisted in store
		storedMsgs, err := h.chatStore.GetMessages(context.Background(), chat.ID)
		if err != nil || len(storedMsgs) != 2 {
			t.Fatalf("expected 2 stored messages, got %d (err: %v)", len(storedMsgs), err)
		}
	})

	t.Run("SearchBackendFailure_Returns502_UserMessagePersisted", func(t *testing.T) {
		h.searchClient.failSearch = true
		defer func() { h.searchClient.failSearch = false }()

		countBefore, _ := h.chatStore.GetMessages(context.Background(), chat.ID)

		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "Some new error message",
		})
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 on search failure, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp["error"] != "search backend unavailable" {
			t.Fatalf("expected 'search backend unavailable', got %q", errResp["error"])
		}

		// Verify user message was already persisted
		countAfter, _ := h.chatStore.GetMessages(context.Background(), chat.ID)
		if len(countAfter) != len(countBefore)+1 {
			t.Fatalf("user message should be saved even if search backend fails (before: %d, after: %d)", len(countBefore), len(countAfter))
		}
		lastMsg := countAfter[len(countAfter)-1]
		if lastMsg.SenderRole != domain.SenderRoleUser || *lastMsg.Content != "Some new error message" {
			t.Fatalf("persisted message does not match user message: %v", lastMsg)
		}
	})

	t.Run("RAGBackendFailure_Returns502", func(t *testing.T) {
		h.searchClient.failRAG = true
		defer func() { h.searchClient.failRAG = false }()

		rec := h.request(http.MethodPost, "/api/v1/chats/"+chat.ID.String()+"/messages", h.tokenA, map[string]interface{}{
			"query": "Another error message",
		})
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 on RAG failure, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

// ---------------------------------------------------------------------------
// 5. History & Listing Tests
// ---------------------------------------------------------------------------

func TestChatHandler_HistoryAndListing(t *testing.T) {
	h := newTestHarness(t)

	// Create test chats for User A
	now := time.Now().UTC()
	chat1 := &domain.Chat{
		ID:        uuid.New(),
		UserID:    h.userA.ID,
		Title:     "Alpha Chat",
		IsPinned:  false,
		CreatedAt: now.Add(-30 * time.Minute),
		UpdatedAt: now.Add(-30 * time.Minute),
	}
	chat2 := &domain.Chat{
		ID:        uuid.New(),
		UserID:    h.userA.ID,
		Title:     "Pinned Beta Chat",
		IsPinned:  true,
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
	}
	_ = h.chatStore.Create(context.Background(), chat1)
	_ = h.chatStore.Create(context.Background(), chat2)

	// Add messages with different created_at to chat1
	msg1Time := now.Add(-10 * time.Minute)
	msg2Time := now.Add(-5 * time.Minute)

	rawQ := "[Go] panic: nil pointer dereference"
	msg1 := &domain.ChatMessage{
		ID:          uuid.New(),
		ChatID:      chat1.ID,
		SenderRole:  domain.SenderRoleUser,
		MessageType: domain.MessageTypeErrorReport,
		Content:     &rawQ,
		CreatedAt:   msg1Time,
	}
	synth := "Fix nil pointer by checking if pointer != nil"
	msg2 := &domain.ChatMessage{
		ID:          uuid.New(),
		ChatID:      chat1.ID,
		SenderRole:  domain.SenderRoleAssistant,
		MessageType: domain.MessageTypeRAGSolution,
		Content:     &synth,
		CreatedAt:   msg2Time,
	}
	// Add in reverse chronological order to test sorting
	_ = h.chatStore.AddMessage(context.Background(), msg2)
	_ = h.chatStore.AddMessage(context.Background(), msg1)

	// Attach tags to msg1
	_ = h.tagStore.AttachToMessage(context.Background(), msg1.ID, []string{"go", "pointers"})

	t.Run("GetChat_SortedAscending_UserTagsPopulated", func(t *testing.T) {
		rec := h.request(http.MethodGet, "/api/v1/chats/"+chat1.ID.String(), h.tokenA, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Chat     domain.Chat          `json:"chat"`
			Messages []domain.ChatMessage `json:"messages"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(resp.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
		}
		if resp.Messages[0].ID != msg1.ID || resp.Messages[1].ID != msg2.ID {
			t.Fatalf("messages should be sorted created_at ASC: msg1 then msg2")
		}

		// User message should have tags populated
		userMsg := resp.Messages[0]
		if len(userMsg.Tags) != 2 {
			t.Fatalf("expected 2 tags on user message, got %d: %v", len(userMsg.Tags), userMsg.Tags)
		}
		if userMsg.Tags[0].Name != "go" || userMsg.Tags[1].Name != "pointers" {
			t.Fatalf("expected tags 'go' and 'pointers', got %v", userMsg.Tags)
		}
	})

	t.Run("ListChats_PaginationClamping", func(t *testing.T) {
		// Default limit 20
		rec := h.request(http.MethodGet, "/api/v1/chats", h.tokenA, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var listResp struct {
			Items []domain.Chat `json:"items"`
			Total int           `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &listResp)
		if listResp.Total != 2 {
			t.Fatalf("expected total 2, got %d", listResp.Total)
		}
		// Pinned chat should be first
		if !listResp.Items[0].IsPinned {
			t.Fatalf("expected first chat to be pinned")
		}

		// Limit clamping to 100
		recClamped := h.request(http.MethodGet, "/api/v1/chats?limit=500&offset=-10", h.tokenA, nil)
		if recClamped.Code != http.StatusOK {
			t.Fatalf("expected 200 with clamped pagination, got %d", recClamped.Code)
		}

		// Search filtering
		recSearch := h.request(http.MethodGet, "/api/v1/chats?search=Beta", h.tokenA, nil)
		if recSearch.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recSearch.Code)
		}
		var searchResp struct {
			Items []domain.Chat `json:"items"`
			Total int           `json:"total"`
		}
		_ = json.Unmarshal(recSearch.Body.Bytes(), &searchResp)
		if searchResp.Total != 1 || searchResp.Items[0].Title != "Pinned Beta Chat" {
			t.Fatalf("expected 1 result matching 'Beta', got %d", searchResp.Total)
		}
	})
}

// ---------------------------------------------------------------------------
// 6. CRUD Validation Tests
// ---------------------------------------------------------------------------

func TestChatHandler_CRUDValidation(t *testing.T) {
	h := newTestHarness(t)

	t.Run("CreateChat_DefaultTitle", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats", h.tokenA, map[string]interface{}{})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var chat domain.Chat
		_ = json.Unmarshal(rec.Body.Bytes(), &chat)
		if chat.Title != "Новый чат" {
			t.Fatalf("expected title 'Новый чат', got %q", chat.Title)
		}
	})

	t.Run("CreateChat_TitleTooLong_Returns400", func(t *testing.T) {
		tooLong := strings.Repeat("a", 256)
		rec := h.request(http.MethodPost, "/api/v1/chats", h.tokenA, map[string]interface{}{
			"title": tooLong,
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for title > 255 chars, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("CreateChat_ValidCustomTitle", func(t *testing.T) {
		rec := h.request(http.MethodPost, "/api/v1/chats", h.tokenA, map[string]interface{}{
			"title": "Custom Dialog",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var chat domain.Chat
		_ = json.Unmarshal(rec.Body.Bytes(), &chat)
		if chat.Title != "Custom Dialog" {
			t.Fatalf("expected 'Custom Dialog', got %q", chat.Title)
		}
	})

	t.Run("UpdateChat_Validation", func(t *testing.T) {
		chat := &domain.Chat{
			ID:        uuid.New(),
			UserID:    h.userA.ID,
			Title:     "Old Title",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_ = h.chatStore.Create(context.Background(), chat)

		// Empty title -> 400
		recEmpty := h.request(http.MethodPatch, "/api/v1/chats/"+chat.ID.String(), h.tokenA, map[string]interface{}{
			"title": "   ",
		})
		if recEmpty.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for empty whitespace title, got %d", recEmpty.Code)
		}

		// Title > 255 -> 400
		tooLong := strings.Repeat("x", 256)
		recLong := h.request(http.MethodPatch, "/api/v1/chats/"+chat.ID.String(), h.tokenA, map[string]interface{}{
			"title": tooLong,
		})
		if recLong.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for title > 255, got %d", recLong.Code)
		}

		// Valid title + pin update
		isPinned := true
		recValid := h.request(http.MethodPatch, "/api/v1/chats/"+chat.ID.String(), h.tokenA, map[string]interface{}{
			"title":     "New Good Title",
			"is_pinned": isPinned,
		})
		if recValid.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recValid.Code, recValid.Body.String())
		}
		var updated domain.Chat
		_ = json.Unmarshal(recValid.Body.Bytes(), &updated)
		if updated.Title != "New Good Title" || !updated.IsPinned {
			t.Fatalf("expected updated title and is_pinned=true, got %v", updated)
		}
	})

	t.Run("DeleteChat_Success", func(t *testing.T) {
		chat := &domain.Chat{
			ID:        uuid.New(),
			UserID:    h.userA.ID,
			Title:     "To Delete",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_ = h.chatStore.Create(context.Background(), chat)

		rec := h.request(http.MethodDelete, "/api/v1/chats/"+chat.ID.String(), h.tokenA, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify deletion
		_, err := h.chatStore.GetByID(context.Background(), chat.ID)
		if err != sql.ErrNoRows {
			t.Fatalf("expected chat to be deleted from store")
		}
	})
}
