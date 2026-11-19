package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/app"
	"github.com/soskiiit/errorium/services/gateway/internal/migrator"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

func TestApp_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not configured, skipping integration test")
	}

	// 1. Run database migrations
	if err := migrator.Up(dsn); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Parse DSN to populate config.DBConfig
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("failed to parse TEST_DB_DSN: %v", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	sslMode := u.Query().Get("sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	password, _ := u.User.Password()

	cfg := &config.Config{
		Port:             "8000",
		SearchEngineAddr: "search_engine_core:50051",
		AutoMigrate:      false,
		JWT: config.JWTConfig{
			SecretKey:              "integration-test-secret-key-at-least-32-chars",
			AccessTokenExpireMin:   60,
			RefreshTokenExpireDays: 30,
		},
		DB: config.DBConfig{
			Host:     host,
			Port:     port,
			Name:     dbName,
			User:     u.User.Username(),
			Password: password,
			SSLMode:  sslMode,
		},
	}

	application := app.New(cfg)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = application.Shutdown(ctx)
	})

	handler := application.Handler()

	doRequest := func(method, path string, body any, token string) *httptest.ResponseRecorder {
		var bodyReader io.Reader
		if body != nil {
			switch b := body.(type) {
			case string:
				bodyReader = strings.NewReader(b)
			default:
				data, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}
				bodyReader = bytes.NewReader(data)
			}
		}

		req := httptest.NewRequest(method, path, bodyReader)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	runID := time.Now().UnixNano()
	user1Email := fmt.Sprintf("user1_%d@test.com", runID)
	user1Password := "Password123!"

	// -------------------------------------------------------------------------
	// Step 1: User 1 registration (POST /api/v1/auth/register)
	// -------------------------------------------------------------------------
	reg1Body := map[string]string{
		"username": "user1_integration",
		"email":    user1Email,
		"password": user1Password,
	}
	recReg1 := doRequest(http.MethodPost, "/api/v1/auth/register", reg1Body, "")
	if recReg1.Code != http.StatusCreated {
		t.Fatalf("Step 1 failed: expected 201 Created on register, got %d: %s", recReg1.Code, recReg1.Body.String())
	}

	var reg1Resp struct {
		AccessToken  string       `json:"access_token"`
		RefreshToken string       `json:"refresh_token"`
		User         *domain.User `json:"user"`
	}
	if err := json.Unmarshal(recReg1.Body.Bytes(), &reg1Resp); err != nil {
		t.Fatalf("Step 1 failed: unmarshal register response: %v", err)
	}
	if reg1Resp.AccessToken == "" || reg1Resp.RefreshToken == "" {
		t.Fatalf("Step 1 failed: empty tokens received on register")
	}
	if reg1Resp.User.Profession != "Backend developer" {
		t.Errorf("Step 1 failed: expected default profession 'Backend developer', got %q", reg1Resp.User.Profession)
	}
	if reg1Resp.User.Gender != "Not specified" {
		t.Errorf("Step 1 failed: expected default gender 'Not specified', got %q", reg1Resp.User.Gender)
	}
	if reg1Resp.User.Language != "English" {
		t.Errorf("Step 1 failed: expected default language 'English', got %q", reg1Resp.User.Language)
	}

	// -------------------------------------------------------------------------
	// Step 2: User 1 login (POST /api/v1/auth/login) -> access & refresh tokens
	// -------------------------------------------------------------------------
	login1Body := map[string]any{
		"email":       user1Email,
		"password":    user1Password,
		"remember_me": true,
	}
	recLogin1 := doRequest(http.MethodPost, "/api/v1/auth/login", login1Body, "")
	if recLogin1.Code != http.StatusOK {
		t.Fatalf("Step 2 failed: expected 200 OK on login, got %d: %s", recLogin1.Code, recLogin1.Body.String())
	}

	var login1Resp struct {
		AccessToken  string       `json:"access_token"`
		RefreshToken string       `json:"refresh_token"`
		User         *domain.User `json:"user"`
	}
	if err := json.Unmarshal(recLogin1.Body.Bytes(), &login1Resp); err != nil {
		t.Fatalf("Step 2 failed: unmarshal login response: %v", err)
	}
	user1Access := login1Resp.AccessToken
	user1Refresh := login1Resp.RefreshToken
	if user1Access == "" || user1Refresh == "" {
		t.Fatalf("Step 2 failed: empty tokens in login response")
	}

	// -------------------------------------------------------------------------
	// Step 3: User 1 get me (GET /api/v1/auth/me) -> verify user data
	// -------------------------------------------------------------------------
	recMe1 := doRequest(http.MethodGet, "/api/v1/auth/me", nil, user1Access)
	if recMe1.Code != http.StatusOK {
		t.Fatalf("Step 3 failed: expected 200 OK on /auth/me, got %d: %s", recMe1.Code, recMe1.Body.String())
	}
	var me1 domain.User
	if err := json.Unmarshal(recMe1.Body.Bytes(), &me1); err != nil {
		t.Fatalf("Step 3 failed: unmarshal me response: %v", err)
	}
	if me1.Email != strings.ToLower(user1Email) {
		t.Errorf("Step 3 failed: expected email %q, got %q", strings.ToLower(user1Email), me1.Email)
	}

	// -------------------------------------------------------------------------
	// Step 4: User 1 patch profile (PATCH /api/v1/users/me) -> update profession, gender, language
	// -------------------------------------------------------------------------
	patchProfBody := map[string]string{
		"profession": "DevOps Engineer",
		"gender":     "Male",
		"language":   "Русский",
	}
	recPatchProf := doRequest(http.MethodPatch, "/api/v1/users/me", patchProfBody, user1Access)
	if recPatchProf.Code != http.StatusOK {
		t.Fatalf("Step 4 failed: expected 200 OK on patch profile, got %d: %s", recPatchProf.Code, recPatchProf.Body.String())
	}
	var patchedProf domain.User
	if err := json.Unmarshal(recPatchProf.Body.Bytes(), &patchedProf); err != nil {
		t.Fatalf("Step 4 failed: unmarshal patch profile response: %v", err)
	}
	if patchedProf.Profession != "DevOps Engineer" {
		t.Errorf("Step 4 failed: expected profession 'DevOps Engineer', got %q", patchedProf.Profession)
	}
	if patchedProf.Gender != "Male" {
		t.Errorf("Step 4 failed: expected gender 'Male', got %q", patchedProf.Gender)
	}
	if patchedProf.Language != "Русский" {
		t.Errorf("Step 4 failed: expected language 'Русский', got %q", patchedProf.Language)
	}

	// -------------------------------------------------------------------------
	// Step 5: User 1 create chat (POST /api/v1/chats)
	// -------------------------------------------------------------------------
	createChatBody := map[string]string{
		"title": "Новый чат",
	}
	recCreateChat := doRequest(http.MethodPost, "/api/v1/chats", createChatBody, user1Access)
	if recCreateChat.Code != http.StatusCreated {
		t.Fatalf("Step 5 failed: expected 201 Created on create chat, got %d: %s", recCreateChat.Code, recCreateChat.Body.String())
	}
	var createdChat domain.Chat
	if err := json.Unmarshal(recCreateChat.Body.Bytes(), &createdChat); err != nil {
		t.Fatalf("Step 5 failed: unmarshal created chat: %v", err)
	}
	if createdChat.ID == uuid.Nil {
		t.Fatalf("Step 5 failed: created chat has nil UUID")
	}
	chatID := createdChat.ID

	// -------------------------------------------------------------------------
	// Step 6: User 1 send message (POST /api/v1/chats/{id}/messages) with query
	// "[Python] TypeError: unsupported operand"
	// -------------------------------------------------------------------------
	msgBody := map[string]string{
		"query": "[Python] TypeError: unsupported operand",
	}
	recSendMsg := doRequest(http.MethodPost, fmt.Sprintf("/api/v1/chats/%s/messages", chatID), msgBody, user1Access)
	if recSendMsg.Code != http.StatusCreated {
		t.Fatalf("Step 6 failed: expected 201 Created on send message, got %d: %s", recSendMsg.Code, recSendMsg.Body.String())
	}
	var sendMsgResp struct {
		UserMessage      *domain.ChatMessage `json:"user_message"`
		AssistantMessage *domain.ChatMessage `json:"assistant_message"`
	}
	if err := json.Unmarshal(recSendMsg.Body.Bytes(), &sendMsgResp); err != nil {
		t.Fatalf("Step 6 failed: unmarshal send message response: %v", err)
	}
	if sendMsgResp.UserMessage == nil || sendMsgResp.AssistantMessage == nil {
		t.Fatalf("Step 6 failed: user_message or assistant_message is nil")
	}

	// -------------------------------------------------------------------------
	// Step 7: User 1 get chat (GET /api/v1/chats/{id}) -> verify message content,
	// code, rag_sources, extracted bracketed tags (["python"]), auto-generated
	// title (first 80 runes of clean query)
	// -------------------------------------------------------------------------
	recGetChat := doRequest(http.MethodGet, fmt.Sprintf("/api/v1/chats/%s", chatID), nil, user1Access)
	if recGetChat.Code != http.StatusOK {
		t.Fatalf("Step 7 failed: expected 200 OK on get chat, got %d: %s", recGetChat.Code, recGetChat.Body.String())
	}
	var getChatResp struct {
		Chat     *domain.Chat          `json:"chat"`
		Messages []*domain.ChatMessage `json:"messages"`
	}
	if err := json.Unmarshal(recGetChat.Body.Bytes(), &getChatResp); err != nil {
		t.Fatalf("Step 7 failed: unmarshal get chat response: %v", err)
	}

	// Verify auto-generated title (clean query = "TypeError: unsupported operand")
	expectedTitle := "TypeError: unsupported operand"
	if getChatResp.Chat.Title != expectedTitle {
		t.Errorf("Step 7 failed: expected auto-generated title %q, got %q", expectedTitle, getChatResp.Chat.Title)
	}

	// Verify messages: user and assistant
	if len(getChatResp.Messages) < 2 {
		t.Fatalf("Step 7 failed: expected at least 2 messages, got %d", len(getChatResp.Messages))
	}
	userMsg := getChatResp.Messages[0]
	asstMsg := getChatResp.Messages[1]

	if userMsg.SenderRole != domain.SenderRoleUser {
		t.Errorf("Step 7 failed: expected first message sender_role 'user', got %q", userMsg.SenderRole)
	}
	if userMsg.Content == nil || *userMsg.Content != "[Python] TypeError: unsupported operand" {
		t.Errorf("Step 7 failed: expected raw query in user message content, got %v", userMsg.Content)
	}
	if len(userMsg.Tags) != 1 || strings.ToLower(userMsg.Tags[0].Name) != "python" {
		t.Errorf("Step 7 failed: expected extracted tag ['python'], got %+v", userMsg.Tags)
	}

	if asstMsg.SenderRole != domain.SenderRoleAssistant {
		t.Errorf("Step 7 failed: expected second message sender_role 'assistant', got %q", asstMsg.SenderRole)
	}
	if asstMsg.Content == nil || *asstMsg.Content == "" {
		t.Errorf("Step 7 failed: assistant message content is empty")
	}
	if asstMsg.Code == nil || *asstMsg.Code == "" {
		t.Errorf("Step 7 failed: assistant message code snippet is empty")
	}
	if asstMsg.RAGSources == nil {
		t.Errorf("Step 7 failed: assistant message rag_sources is nil")
	}

	// -------------------------------------------------------------------------
	// Step 8: User 1 patch chat (PATCH /api/v1/chats/{id}) -> pin + rename
	// -------------------------------------------------------------------------
	renamedTitle := "Renamed Python Chat"
	pinned := true
	patchChatBody := map[string]any{
		"title":     renamedTitle,
		"is_pinned": pinned,
	}
	recPatchChat := doRequest(http.MethodPatch, fmt.Sprintf("/api/v1/chats/%s", chatID), patchChatBody, user1Access)
	if recPatchChat.Code != http.StatusOK {
		t.Fatalf("Step 8 failed: expected 200 OK on patch chat, got %d: %s", recPatchChat.Code, recPatchChat.Body.String())
	}
	var patchedChat domain.Chat
	if err := json.Unmarshal(recPatchChat.Body.Bytes(), &patchedChat); err != nil {
		t.Fatalf("Step 8 failed: unmarshal patch chat response: %v", err)
	}
	if patchedChat.Title != renamedTitle || !patchedChat.IsPinned {
		t.Errorf("Step 8 failed: expected title %q and is_pinned true, got %q and %v", renamedTitle, patchedChat.Title, patchedChat.IsPinned)
	}

	// -------------------------------------------------------------------------
	// Step 9: User 1 list chats (GET /api/v1/chats) -> search finds updated chat
	// -------------------------------------------------------------------------
	recListChats := doRequest(http.MethodGet, "/api/v1/chats?search=Renamed", nil, user1Access)
	if recListChats.Code != http.StatusOK {
		t.Fatalf("Step 9 failed: expected 200 OK on list chats, got %d: %s", recListChats.Code, recListChats.Body.String())
	}
	var listResp struct {
		Items []*domain.Chat `json:"items"`
		Total int            `json:"total"`
	}
	if err := json.Unmarshal(recListChats.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("Step 9 failed: unmarshal list chats response: %v", err)
	}
	if listResp.Total < 1 {
		t.Errorf("Step 9 failed: search did not find updated chat, total = %d", listResp.Total)
	}
	found := false
	for _, c := range listResp.Items {
		if c.ID == chatID && c.Title == renamedTitle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Step 9 failed: updated chat %s not found in search results", chatID)
	}

	// -------------------------------------------------------------------------
	// Step 10: Register & login User 2
	// -------------------------------------------------------------------------
	user2Email := fmt.Sprintf("user2_%d@test.com", runID)
	user2Password := "Password456!"
	reg2Body := map[string]string{
		"username": "user2_integration",
		"email":    user2Email,
		"password": user2Password,
	}
	recReg2 := doRequest(http.MethodPost, "/api/v1/auth/register", reg2Body, "")
	if recReg2.Code != http.StatusCreated {
		t.Fatalf("Step 10 failed: expected 201 Created on User 2 register, got %d: %s", recReg2.Code, recReg2.Body.String())
	}

	login2Body := map[string]string{
		"email":    user2Email,
		"password": user2Password,
	}
	recLogin2 := doRequest(http.MethodPost, "/api/v1/auth/login", login2Body, "")
	if recLogin2.Code != http.StatusOK {
		t.Fatalf("Step 10 failed: expected 200 OK on User 2 login, got %d: %s", recLogin2.Code, recLogin2.Body.String())
	}
	var login2Resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(recLogin2.Body.Bytes(), &login2Resp); err != nil {
		t.Fatalf("Step 10 failed: unmarshal User 2 login response: %v", err)
	}
	user2Access := login2Resp.AccessToken

	// -------------------------------------------------------------------------
	// Step 11: User 2 attempts to get User 1's chat (GET /api/v1/chats/{id}) -> 404
	// -------------------------------------------------------------------------
	recOtherChat := doRequest(http.MethodGet, fmt.Sprintf("/api/v1/chats/%s", chatID), nil, user2Access)
	if recOtherChat.Code != http.StatusNotFound {
		t.Fatalf("Step 11 failed: expected 404 Not Found on cross-user chat access, got %d: %s", recOtherChat.Code, recOtherChat.Body.String())
	}

	// -------------------------------------------------------------------------
	// Step 12: User 1 deletes chat (DELETE /api/v1/chats/{id}) -> 204, subsequent GET -> 404
	// -------------------------------------------------------------------------
	recDelChat := doRequest(http.MethodDelete, fmt.Sprintf("/api/v1/chats/%s", chatID), nil, user1Access)
	if recDelChat.Code != http.StatusNoContent {
		t.Fatalf("Step 12 failed: expected 204 No Content on delete chat, got %d: %s", recDelChat.Code, recDelChat.Body.String())
	}
	recGetDeleted := doRequest(http.MethodGet, fmt.Sprintf("/api/v1/chats/%s", chatID), nil, user1Access)
	if recGetDeleted.Code != http.StatusNotFound {
		t.Fatalf("Step 12 failed: expected 404 Not Found after deleting chat, got %d: %s", recGetDeleted.Code, recGetDeleted.Body.String())
	}

	// -------------------------------------------------------------------------
	// Step 13: Refresh token rotation (POST /api/v1/auth/refresh) -> old refresh
	// token revoked (subsequent use -> 401)
	// -------------------------------------------------------------------------
	refreshBody := map[string]string{
		"refresh_token": user1Refresh,
	}
	recRefresh1 := doRequest(http.MethodPost, "/api/v1/auth/refresh", refreshBody, "")
	if recRefresh1.Code != http.StatusOK {
		t.Fatalf("Step 13 failed: expected 200 OK on refresh token, got %d: %s", recRefresh1.Code, recRefresh1.Body.String())
	}
	var refreshResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(recRefresh1.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("Step 13 failed: unmarshal refresh response: %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("Step 13 failed: received empty tokens after refresh")
	}

	// Subsequent use of old refresh token must yield 401
	recOldRefresh := doRequest(http.MethodPost, "/api/v1/auth/refresh", refreshBody, "")
	if recOldRefresh.Code != http.StatusUnauthorized {
		t.Fatalf("Step 13 failed: expected 401 Unauthorized when re-using old refresh token, got %d: %s", recOldRefresh.Code, recOldRefresh.Body.String())
	}
}
