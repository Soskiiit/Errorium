package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// mockChatRAG implements handler.ChatRAGSynthesizer and captures the chat context.
type mockChatRAG struct {
	mu       sync.Mutex
	lastCC   rag.ChatContext
	remember string
}

func (m *mockChatRAG) Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
	return m.SynthesizeChat(ctx, query, hits, rag.ChatContext{})
}

func (m *mockChatRAG) SynthesizeStream(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
	return m.SynthesizeChatStream(ctx, query, hits, rag.ChatContext{}, onDelta)
}

func (m *mockChatRAG) SynthesizeChat(ctx context.Context, query string, hits []search.Source, cc rag.ChatContext) (*search.RAGResult, error) {
	m.mu.Lock()
	m.lastCC = cc
	m.mu.Unlock()
	res := &search.RAGResult{Query: query, Synthesis: "ответ", Sources: hits}
	if m.remember != "" {
		r := m.remember
		res.Remember = &r
	}
	return res, nil
}

func (m *mockChatRAG) SynthesizeChatStream(ctx context.Context, query string, hits []search.Source, cc rag.ChatContext, onDelta func(string)) (*search.RAGResult, error) {
	if onDelta != nil {
		onDelta("ответ")
	}
	return m.SynthesizeChat(ctx, query, hits, cc)
}

func newChatMemoryHarness(t *testing.T, remember string) (*testChatStore, *mockChatRAG, *http.ServeMux, string, *domain.Chat) {
	t.Helper()

	jwtMgr := jwtpkg.NewManager(config.JWTConfig{
		SecretKey:            "super-secret-gateway-test-key-32-bytes!!",
		AccessTokenExpireMin: 15,
	})
	store := newTestChatStore()
	orch := &mockChatRAG{remember: remember}
	ch := handler.NewChatHandler(store, newTestTagStore(), newTestSearchClient(), orch)
	mux := http.NewServeMux()
	ch.RegisterRoutes(mux, middleware.Auth(jwtMgr))

	user := &domain.User{ID: uuid.New(), Email: "mem@example.com", Username: "mem", Role: domain.RoleUser}
	token, err := jwtMgr.GenerateAccess(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	now := time.Now().UTC()
	chat := &domain.Chat{ID: uuid.New(), UserID: user.ID, Title: "Новый чат", Memory: "стек: python", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), chat); err != nil {
		t.Fatalf("failed to create chat: %v", err)
	}
	return store, orch, mux, token, chat
}

func sendChatMessage(t *testing.T, mux *http.ServeMux, token string, chatID uuid.UUID, query string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"query": query})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chats/"+chatID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestChatHandler_SendMessage_ChatMemory(t *testing.T) {
	store, orch, mux, token, chat := newChatMemoryHarness(t, "факт: python 3.12")

	base := time.Now().UTC()
	for i := 0; i < 7; i++ {
		role := domain.SenderRoleUser
		if i%2 == 1 {
			role = domain.SenderRoleAssistant
		}
		content := fmt.Sprintf("msg %d", i)
		if err := store.AddMessage(context.Background(), &domain.ChatMessage{
			ID:          uuid.New(),
			ChatID:      chat.ID,
			SenderRole:  role,
			MessageType: domain.MessageTypeText,
			Content:     &content,
			CreatedAt:   base.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	rec := sendChatMessage(t, mux, token, chat.ID, "а почему так?")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	orch.mu.Lock()
	cc := orch.lastCC
	orch.mu.Unlock()

	if cc.Memory != "стек: python" {
		t.Errorf("expected chat memory injected, got %q", cc.Memory)
	}
	if len(cc.History) != 5 {
		t.Fatalf("expected history window of 5, got %d", len(cc.History))
	}
	if cc.History[0].Content != "msg 2" || cc.History[4].Content != "msg 6" {
		t.Errorf("expected last 5 messages (msg 2..msg 6), got %v", cc.History)
	}

	got, err := store.GetByID(context.Background(), chat.ID)
	if err != nil {
		t.Fatalf("failed to get chat: %v", err)
	}
	if got.Memory != "факт: python 3.12" {
		t.Errorf("expected memory updated from remember block, got %q", got.Memory)
	}
}

func TestChatHandler_SendMessage_EmptyRememberKeepsMemory(t *testing.T) {
	store, _, mux, token, chat := newChatMemoryHarness(t, "")

	rec := sendChatMessage(t, mux, token, chat.ID, "привет")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := store.GetByID(context.Background(), chat.ID)
	if err != nil {
		t.Fatalf("failed to get chat: %v", err)
	}
	if got.Memory != "стек: python" {
		t.Errorf("empty remember must keep previous memory, got %q", got.Memory)
	}
}