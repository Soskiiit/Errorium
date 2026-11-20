package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

type mockRAGSynthesizer struct {
	synthesizeFunc       func(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error)
	synthesizeStreamFunc func(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error)
}

func (m *mockRAGSynthesizer) Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, query, hits)
	}
	code := "print('fixed')"
	return &search.RAGResult{
		Query:     query,
		Synthesis: "Complete synthesis",
		Code:      &code,
		Sources:   hits,
		TookMs:    10,
	}, nil
}

func (m *mockRAGSynthesizer) SynthesizeStream(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
	if m.synthesizeStreamFunc != nil {
		return m.synthesizeStreamFunc(ctx, query, hits, onDelta)
	}
	onDelta("Delta 1 ")
	onDelta("Delta 2")
	code := "print('fixed')"
	return &search.RAGResult{
		Query:     query,
		Synthesis: "Delta 1 Delta 2",
		Code:      &code,
		Sources:   hits,
		TookMs:    15,
	}, nil
}

type fakeChatStoreForSSE struct {
	mu       sync.Mutex
	chats    map[uuid.UUID]*domain.Chat
	messages []*domain.ChatMessage
}

func newFakeChatStoreForSSE() *fakeChatStoreForSSE {
	return &fakeChatStoreForSSE{
		chats:    make(map[uuid.UUID]*domain.Chat),
		messages: make([]*domain.ChatMessage, 0),
	}
}

func (f *fakeChatStoreForSSE) Create(ctx context.Context, c *domain.Chat) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chats[c.ID] = c
	return nil
}

func (f *fakeChatStoreForSSE) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, s string) ([]*domain.Chat, int, error) {
	return nil, 0, nil
}

func (f *fakeChatStoreForSSE) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.chats[id]
	if !ok {
		return nil, errors.New("chat not found")
	}
	return c, nil
}

func (f *fakeChatStoreForSSE) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.chats[id]
	if !ok || c.UserID != userID {
		return nil, errors.New("chat not found")
	}
	return c, nil
}

func (f *fakeChatStoreForSSE) Update(ctx context.Context, c *domain.Chat) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chats[c.ID] = c
	return nil
}

func (f *fakeChatStoreForSSE) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.chats, id)
	return nil
}

func (f *fakeChatStoreForSSE) AddMessage(ctx context.Context, msg *domain.ChatMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeChatStoreForSSE) GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var res []*domain.ChatMessage
	for _, m := range f.messages {
		if m.ChatID == chatID {
			res = append(res, m)
		}
	}
	return res, nil
}

func TestSearchRAG_SSE_Streaming(t *testing.T) {
	searchStub := search.NewStub()
	graphStub := &fakeGraphClient{}
	ragMock := &mockRAGSynthesizer{
		synthesizeStreamFunc: func(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
			onDelta("Part 1: cause. ")
			onDelta("Part 2: fix.")
			code := "def fix(): pass"
			return &search.RAGResult{
				Query:     query,
				Synthesis: "Part 1: cause. Part 2: fix.",
				Code:      &code,
				Sources:   hits,
				TookMs:    42,
			}, nil
		},
	}

	h := handler.NewSearchHandler(searchStub, graphStub, ragMock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 1. Request with Accept: text/event-stream
	reqBody := `{"query":"syntax error in python","top_k":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	// Verify headers
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("X-Accel-Buffering") != "no" {
		t.Errorf("expected X-Accel-Buffering no, got %s", rec.Header().Get("X-Accel-Buffering"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: delta\ndata: {\"text\":\"Part 1: cause. \"}\n\n") {
		t.Errorf("expected first delta event in body: %s", body)
	}
	if !strings.Contains(body, "event: delta\ndata: {\"text\":\"Part 2: fix.\"}\n\n") {
		t.Errorf("expected second delta event in body: %s", body)
	}
	if !strings.Contains(body, "event: done\ndata: {") {
		t.Errorf("expected done event in body: %s", body)
	}

	// 2. Error BEFORE stream (e.g. invalid query) -> standard JSON with 400
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", strings.NewReader(`{"query":""}`))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Accept", "text/event-stream")

	recInvalid := httptest.NewRecorder()
	mux.ServeHTTP(recInvalid, reqInvalid)

	if recInvalid.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 before stream, got %d", recInvalid.Code)
	}
	if !strings.Contains(recInvalid.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON error before stream, got: %s", recInvalid.Header().Get("Content-Type"))
	}

	// 3. Error MIDWAY through stream -> event: error
	ragErrMidway := &mockRAGSynthesizer{
		synthesizeStreamFunc: func(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
			onDelta("Some tokens")
			return nil, errors.New("connection reset by peer")
		},
	}
	hErr := handler.NewSearchHandler(searchStub, graphStub, ragErrMidway)
	muxErr := http.NewServeMux()
	hErr.RegisterRoutes(muxErr)

	reqMidway := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", strings.NewReader(reqBody))
	reqMidway.Header.Set("Content-Type", "application/json")
	reqMidway.Header.Set("Accept", "text/event-stream")

	recMidway := httptest.NewRecorder()
	muxErr.ServeHTTP(recMidway, reqMidway)

	if !strings.Contains(recMidway.Body.String(), "event: error\ndata: {\"error\":\"connection reset by peer\"}\n\n") {
		t.Errorf("expected error event midway: %s", recMidway.Body.String())
	}
}

func TestSearchRAG_NonSSE_Regression(t *testing.T) {
	searchStub := search.NewStub()
	graphStub := &fakeGraphClient{}
	ragMock := &mockRAGSynthesizer{}

	h := handler.NewSearchHandler(searchStub, graphStub, ragMock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Without Accept: text/event-stream -> returns standard JSON 200
	reqBody := `{"query":"syntax error","top_k":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected application/json, got %s", rec.Header().Get("Content-Type"))
	}
	if strings.Contains(rec.Body.String(), "event: ") {
		t.Errorf("non-SSE response must not contain SSE events: %s", rec.Body.String())
	}
}

func TestChatMessages_SSE_Streaming_Sequence(t *testing.T) {
	store := newFakeChatStoreForSSE()
	userID := uuid.New()
	chatID := uuid.New()

	chat := &domain.Chat{
		ID:        chatID,
		UserID:    userID,
		Title:     "Новый чат",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = store.Create(context.Background(), chat)

	searchStub := search.NewStub()
	ragMock := &mockRAGSynthesizer{
		synthesizeStreamFunc: func(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
			onDelta("Here is ")
			onDelta("the solution.")
			code := "print('hello')"
			return &search.RAGResult{
				Query:     query,
				Synthesis: "Here is the solution.",
				Code:      &code,
				Sources:   hits,
			}, nil
		},
	}

	chatHandler := handler.NewChatHandler(store, nil, searchStub, ragMock)
	mux := http.NewServeMux()
	chatHandler.RegisterRoutes(mux)

	reqBody := `{"query":"[python] print error","top_k":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID.String()+"/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(middleware.WithUserID(req.Context(), userID))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("expected 201 or 200, got %d, body: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()

	// 1. First event MUST be event: user_message
	if !strings.HasPrefix(strings.TrimSpace(body), "event: user_message") {
		t.Fatalf("stream must start with user_message event, got: %s", body)
	}

	// 2. Next events MUST be delta
	if !strings.Contains(body, "event: delta\ndata: {\"text\":\"Here is \"}\n\n") {
		t.Errorf("missing delta event 1: %s", body)
	}
	if !strings.Contains(body, "event: delta\ndata: {\"text\":\"the solution.\"}\n\n") {
		t.Errorf("missing delta event 2: %s", body)
	}

	// 3. Final event MUST be event: done with user_message and assistant_message
	if !strings.Contains(body, "event: done\ndata: {") {
		t.Errorf("missing done event: %s", body)
	}

	// 4. Verify both messages are in DB
	messages, err := store.GetMessages(context.Background(), chatID)
	if err != nil || len(messages) != 2 {
		t.Fatalf("expected 2 messages in DB (user + assistant), got %d (err: %v)", len(messages), err)
	}
	if messages[0].SenderRole != domain.SenderRoleUser {
		t.Errorf("expected message 0 to be user, got %s", messages[0].SenderRole)
	}
	if messages[1].SenderRole != domain.SenderRoleAssistant {
		t.Errorf("expected message 1 to be assistant, got %s", messages[1].SenderRole)
	}
	if messages[1].Content == nil || *messages[1].Content != "Here is the solution." {
		t.Errorf("unexpected assistant content: %v", messages[1].Content)
	}
	if messages[1].Code == nil || *messages[1].Code != "print('hello')" {
		t.Errorf("unexpected assistant code: %v", messages[1].Code)
	}
}

func TestChatMessages_SSE_ClientDisconnect_SavesAccumulated(t *testing.T) {
	store := newFakeChatStoreForSSE()
	userID := uuid.New()
	chatID := uuid.New()

	chat := &domain.Chat{
		ID:        chatID,
		UserID:    userID,
		Title:     "Чат для проверки обрыва",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = store.Create(context.Background(), chat)

	ctx, cancel := context.WithCancel(context.Background())

	searchStub := search.NewStub()
	ragMock := &mockRAGSynthesizer{
		synthesizeStreamFunc: func(c context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
			onDelta("Accumulated partial response before disconnect")
			// Simulate client disconnect: cancel context!
			cancel()
			return nil, context.Canceled
		},
	}

	chatHandler := handler.NewChatHandler(store, nil, searchStub, ragMock)
	mux := http.NewServeMux()
	chatHandler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID.String()+"/messages", strings.NewReader(`{"query":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(middleware.WithUserID(ctx, userID))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify that accumulated content was saved to DB as assistant message
	messages, err := store.GetMessages(context.Background(), chatID)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	var foundAssistant bool
	for _, m := range messages {
		if m.SenderRole == domain.SenderRoleAssistant {
			foundAssistant = true
			if m.Content == nil || !strings.Contains(*m.Content, "Accumulated partial response") {
				t.Errorf("expected accumulated content in assistant message, got: %v", m.Content)
			}
		}
	}
	if !foundAssistant {
		t.Fatal("expected assistant message with accumulated content to be saved upon disconnect")
	}
}

type fakeGraphClient struct{}

func (f *fakeGraphClient) GetThreadGraph(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
	return &graph.ThreadGraph{}, nil
}

func (f *fakeGraphClient) GetThreadDetails(ctx context.Context, docID int64) (*graph.ThreadDetails, error) {
	return &graph.ThreadDetails{}, nil
}
