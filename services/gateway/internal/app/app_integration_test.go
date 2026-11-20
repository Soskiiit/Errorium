package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/app"
	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/mcp"
	"github.com/soskiiit/errorium/services/gateway/internal/migrator"
	graphpb "github.com/soskiiit/errorium/services/gateway/internal/pb/graph"
	searchpb "github.com/soskiiit/errorium/services/gateway/internal/pb/search"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
	"github.com/soskiiit/errorium/services/gateway/internal/sso"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

type mockSearchEngineServer struct {
	searchpb.UnimplementedSearchServiceServer
}

func (s *mockSearchEngineServer) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	return &searchpb.SearchResponse{
		TotalHits: 1,
		Hits: []*searchpb.SearchHit{
			{
				DocId:         27539308,
				Title:         "SyntaxError: invalid syntax in Python",
				Snippet:       "Syntax error snippet",
				Score:         4.5,
				QuestionScore: 4180,
				Tags:          []string{"python"},
			},
		},
	}, nil
}

func (s *mockSearchEngineServer) GetStats(ctx context.Context, req *searchpb.GetStatsRequest) (*searchpb.GetStatsResponse, error) {
	return &searchpb.GetStatsResponse{
		TotalDocuments: 100,
		AvgDocLength:   150.5,
		VocabularySize: 2000,
		IndexSizeBytes: 1048576,
		IsReady:        true,
	}, nil
}

func (s *mockSearchEngineServer) ReloadIndex(ctx context.Context, req *searchpb.ReloadIndexRequest) (*searchpb.ReloadIndexResponse, error) {
	return &searchpb.ReloadIndexResponse{
		Ok:             true,
		Message:        "Index successfully reloaded",
		BuildTimestamp: 1750000000,
	}, nil
}

func (s *mockSearchEngineServer) HealthCheck(ctx context.Context, req *searchpb.HealthCheckRequest) (*searchpb.HealthCheckResponse, error) {
	return &searchpb.HealthCheckResponse{
		Status:  searchpb.HealthCheckResponse_SERVING,
		Version: "1.0.0",
	}, nil
}

type mockGraphServiceServer struct {
	graphpb.UnimplementedGraphServiceServer
}

func (s *mockGraphServiceServer) GetThreadGraph(ctx context.Context, req *graphpb.ThreadGraphRequest) (*graphpb.ThreadGraphResponse, error) {
	return &graphpb.ThreadGraphResponse{
		Nodes: []*graphpb.ThreadNode{
			{
				DocId:    27539308,
				Title:    "SyntaxError in Python",
				Category: graphpb.NodeCategory_NODE_CATEGORY_PRIMARY_RESULT,
			},
		},
	}, nil
}

func (s *mockGraphServiceServer) GetThreadDetails(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error) {
	return &graphpb.ThreadDetailsResponse{
		DocId:          req.DocId,
		Title:          "Thread details title",
		BestAnswerBody: "<p>Answer body</p>",
	}, nil
}

func TestApp_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not configured, skipping integration test")
	}

	// 1. Run database migrations
	if err := migrator.Up(dsn); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Start in-process mock gRPC server for core engine
	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on tcp for mock grpc: %v", err)
	}
	defer grpcLis.Close()

	grpcSrv := grpc.NewServer()
	searchpb.RegisterSearchServiceServer(grpcSrv, &mockSearchEngineServer{})
	graphpb.RegisterGraphServiceServer(grpcSrv, &mockGraphServiceServer{})
	go func() {
		_ = grpcSrv.Serve(grpcLis)
	}()
	defer grpcSrv.Stop()

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
		SearchEngineAddr: grpcLis.Addr().String(),
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

	appHandler := application.Handler()

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
		appHandler.ServeHTTP(rec, req)
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
	var reg2Resp struct {
		AccessToken  string       `json:"access_token"`
		RefreshToken string       `json:"refresh_token"`
		User         *domain.User `json:"user"`
	}
	if err := json.Unmarshal(recReg2.Body.Bytes(), &reg2Resp); err != nil {
		t.Fatalf("Step 10 failed: unmarshal User 2 register response: %v", err)
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

	// -------------------------------------------------------------------------
	// Step 14: Wave B - Password forgot (LogMailer) -> reset -> login with new password -> old sessions 401; change -> revoke
	// -------------------------------------------------------------------------
	var logBuf bytes.Buffer
	prevLogWriter := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(prevLogWriter)

	forgotBody := map[string]string{
		"email": user1Email,
	}
	recForgot := doRequest(http.MethodPost, "/api/v1/auth/password/forgot", forgotBody, "")
	if recForgot.Code != http.StatusAccepted {
		t.Fatalf("Step 14 failed: expected 202 Accepted on forgot password, got %d: %s", recForgot.Code, recForgot.Body.String())
	}

	logStr := logBuf.String()
	tokenKey := "?token="
	idx := strings.Index(logStr, tokenKey)
	if idx == -1 {
		t.Fatalf("Step 14 failed: reset token not found in LogMailer output: %q", logStr)
	}
	resetToken := strings.TrimSpace(logStr[idx+len(tokenKey):])
	if nl := strings.IndexAny(resetToken, " \r\n"); nl != -1 {
		resetToken = resetToken[:nl]
	}
	if resetToken == "" {
		t.Fatalf("Step 14 failed: parsed empty reset token")
	}

	newPassword := "NewSecret123!"
	resetBody := map[string]string{
		"token":        resetToken,
		"new_password": newPassword,
	}
	recReset := doRequest(http.MethodPost, "/api/v1/auth/password/reset", resetBody, "")
	if recReset.Code != http.StatusNoContent {
		t.Fatalf("Step 14 failed: expected 204 No Content on reset password, got %d: %s", recReset.Code, recReset.Body.String())
	}

	// Old refresh token must be revoked (401)
	recOldRefreshAfterReset := doRequest(http.MethodPost, "/api/v1/auth/refresh", map[string]string{"refresh_token": user1Refresh}, "")
	if recOldRefreshAfterReset.Code != http.StatusUnauthorized {
		t.Fatalf("Step 14 failed: expected 401 on old refresh token after reset, got %d", recOldRefreshAfterReset.Code)
	}

	// Login with old password must fail (401)
	recOldLogin := doRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user1Email, "password": user1Password}, "")
	if recOldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("Step 14 failed: expected 401 on login with old password, got %d", recOldLogin.Code)
	}

	// Login with new password must succeed (200)
	recNewLogin := doRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user1Email, "password": newPassword}, "")
	if recNewLogin.Code != http.StatusOK {
		t.Fatalf("Step 14 failed: expected 200 on login with new password, got %d: %s", recNewLogin.Code, recNewLogin.Body.String())
	}
	var newLoginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(recNewLogin.Body.Bytes(), &newLoginResp); err != nil {
		t.Fatalf("Step 14 failed: unmarshal login response: %v", err)
	}
	activeAccessToken := newLoginResp.AccessToken
	activeRefreshToken := newLoginResp.RefreshToken

	// Change password via /auth/password/change
	changedPassword := "ChangedSecret123!"
	changeBody := map[string]string{
		"current_password": newPassword,
		"new_password":     changedPassword,
	}
	recChange := doRequest(http.MethodPost, "/api/v1/auth/password/change", changeBody, activeAccessToken)
	if recChange.Code != http.StatusNoContent {
		t.Fatalf("Step 14 failed: expected 204 on change password, got %d: %s", recChange.Code, recChange.Body.String())
	}

	// Refresh token issued before change must be revoked (401)
	recRevokedAfterChange := doRequest(http.MethodPost, "/api/v1/auth/refresh", map[string]string{"refresh_token": activeRefreshToken}, "")
	if recRevokedAfterChange.Code != http.StatusUnauthorized {
		t.Fatalf("Step 14 failed: expected 401 on refresh token after change password, got %d", recRevokedAfterChange.Code)
	}

	// Login with changed password must succeed
	recChangedLogin := doRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user1Email, "password": changedPassword}, "")
	if recChangedLogin.Code != http.StatusOK {
		t.Fatalf("Step 14 failed: expected 200 on login with changed password, got %d", recChangedLogin.Code)
	}
	var changedLoginResp struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(recChangedLogin.Body.Bytes(), &changedLoginResp)
	user1Access = changedLoginResp.AccessToken

	// -------------------------------------------------------------------------
	// Step 15: Wave B - SSO mock-IdP (httptest) -> login -> callback -> user created in DB -> repeat login with same ID -> same user
	// -------------------------------------------------------------------------
	mockIdP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/authorize":
			state := r.URL.Query().Get("state")
			redirectURI := r.URL.Query().Get("redirect_uri")
			http.Redirect(w, r, redirectURI+"?code=mock_oauth_code&state="+state, http.StatusFound)
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock_idp_token_xyz",
				"token_type":   "bearer",
			})
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    999888,
				"login": "octocat_sso",
				"email": "octocat_sso@example.com",
			})
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "octocat_sso@example.com", "primary": true, "verified": true},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockIdP.Close()

	dbPool, err := db.New(cfg.DB)
	if err != nil {
		t.Fatalf("Step 15 failed: open db pool: %v", err)
	}
	defer dbPool.Close()

	ssoStateRepo := repository.NewSsoStateRepository(dbPool)
	ssoAccountRepo := repository.NewSSOAccountRepository(dbPool)
	userRepo := repository.NewUserRepository(dbPool)
	sessionRepo := repository.NewSessionRepository(dbPool)
	jwtMgr := jwtpkg.NewManager(cfg.JWT)

	ssoRegistry := sso.NewRegistry()
	ssoRegistry.Register(sso.NewGitHubProvider(sso.ProviderConfig{
		ClientID:     "mock_client_id",
		ClientSecret: "mock_client_secret",
		AuthURL:      mockIdP.URL + "/login/oauth/authorize",
		TokenURL:     mockIdP.URL + "/login/oauth/access_token",
		UserInfoURL:  mockIdP.URL + "/user",
		EmailsURL:    mockIdP.URL + "/user/emails",
		HTTPClient:   mockIdP.Client(),
	}))

	ssoH := handler.NewSSOHandler(
		ssoRegistry,
		ssoStateRepo,
		ssoAccountRepo,
		userRepo,
		sessionRepo,
		jwtMgr,
		"http://localhost:8000",
		"http://localhost:3000",
		30,
	)
	ssoMux := http.NewServeMux()
	ssoH.RegisterRoutes(ssoMux)

	// 1. Initiate login
	reqSSOLogin := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/login/github", nil)
	recSSOLogin := httptest.NewRecorder()
	ssoMux.ServeHTTP(recSSOLogin, reqSSOLogin)
	if recSSOLogin.Code != http.StatusFound {
		t.Fatalf("Step 15 failed: expected 302 on SSO login, got %d: %s", recSSOLogin.Code, recSSOLogin.Body.String())
	}
	locLogin := recSSOLogin.Header().Get("Location")
	uLogin, err := url.Parse(locLogin)
	if err != nil {
		t.Fatalf("Step 15 failed: parse SSO login Location %q: %v", locLogin, err)
	}
	stateVal := uLogin.Query().Get("state")
	if stateVal == "" {
		t.Fatalf("Step 15 failed: expected state param in SSO redirect URL, got %q", locLogin)
	}

	// Verify state is stored in real DB
	stateInDB, err := ssoStateRepo.Get(context.Background(), stateVal)
	if err != nil || stateInDB == nil {
		t.Fatalf("Step 15 failed: state not found in DB: %v", err)
	}

	// 2. Callback
	cbPath := fmt.Sprintf("/api/v1/auth/sso/callback/github?code=mock_oauth_code&state=%s", stateVal)
	reqSSOCallback := httptest.NewRequest(http.MethodGet, cbPath, nil)
	recSSOCallback := httptest.NewRecorder()
	ssoMux.ServeHTTP(recSSOCallback, reqSSOCallback)
	if recSSOCallback.Code != http.StatusFound {
		t.Fatalf("Step 15 failed: expected 302 on SSO callback, got %d: %s", recSSOCallback.Code, recSSOCallback.Body.String())
	}
	locCallback := recSSOCallback.Header().Get("Location")
	if !strings.Contains(locCallback, "access_token=") || !strings.Contains(locCallback, "refresh_token=") {
		t.Fatalf("Step 15 failed: expected tokens in callback redirect hash, got %q", locCallback)
	}

	// Verify user is in real DB
	ssoUser, err := userRepo.GetByEmail(context.Background(), "octocat_sso@example.com")
	if err != nil || ssoUser == nil {
		t.Fatalf("Step 15 failed: user not found in DB after SSO: %v", err)
	}
	ssoAccount, err := ssoAccountRepo.GetByProviderAndUserID(context.Background(), domain.SSOProviderGitHub, "999888")
	if err != nil || ssoAccount == nil {
		t.Fatalf("Step 15 failed: SSO account link not found in DB: %v", err)
	}
	if ssoAccount.UserID != ssoUser.ID {
		t.Fatalf("Step 15 failed: expected SSO account user ID %s, got %s", ssoUser.ID, ssoAccount.UserID)
	}

	// 3. Repeat SSO login with the same IdP user -> must return same user
	reqSSOLogin2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/login/github", nil)
	recSSOLogin2 := httptest.NewRecorder()
	ssoMux.ServeHTTP(recSSOLogin2, reqSSOLogin2)
	uLogin2, _ := url.Parse(recSSOLogin2.Header().Get("Location"))
	stateVal2 := uLogin2.Query().Get("state")

	cbPath2 := fmt.Sprintf("/api/v1/auth/sso/callback/github?code=mock_oauth_code&state=%s", stateVal2)
	reqSSOCallback2 := httptest.NewRequest(http.MethodGet, cbPath2, nil)
	recSSOCallback2 := httptest.NewRecorder()
	ssoMux.ServeHTTP(recSSOCallback2, reqSSOCallback2)
	if recSSOCallback2.Code != http.StatusFound {
		t.Fatalf("Step 15 failed: expected 302 on second SSO callback, got %d", recSSOCallback2.Code)
	}

	ssoUser2, err := userRepo.GetByEmail(context.Background(), "octocat_sso@example.com")
	if err != nil || ssoUser2 == nil {
		t.Fatalf("Step 15 failed: get user on repeat SSO: %v", err)
	}
	if ssoUser2.ID != ssoUser.ID {
		t.Fatalf("Step 15 failed: repeat SSO created a different user! expected %s, got %s", ssoUser.ID, ssoUser2.ID)
	}

	// -------------------------------------------------------------------------
	// Step 16: Wave B - MCP open /sse -> initialize -> tools/list -> tools/call -> limits 429/503 -> session close releases counters
	// -------------------------------------------------------------------------
	mcpServer := mcp.NewServer(search.NewStub(), graph.NewStub(), rag.NewOrchestrator(nil, rag.NewSemaphore(8), nil, nil, "", "", "", nil))
	mcpLimiter := mcp.NewLimiter(mcp.LimiterConfig{
		MaxSessionsPerIP: 2,
		MaxSessions:      3,
		CallsPerMin:      2,
	})
	mcpHandler := handler.NewMCPHandler(mcpServer, mcpLimiter)
	mcpHandler.SetPingInterval(10 * time.Minute)
	mcpMux := http.NewServeMux()
	mcpHandler.RegisterRoutes(mcpMux)

	// 1. Establish SSE session 1
	ctxSSE1, cancelSSE1 := context.WithCancel(context.Background())
	defer cancelSSE1()
	reqSSE1 := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctxSSE1)
	reqSSE1.RemoteAddr = "10.10.10.1:1111"
	recSSE1 := httptest.NewRecorder()

	go mcpMux.ServeHTTP(recSSE1, reqSSE1)

	// Wait for endpoint event
	var sessionID1 string
	for i := 0; i < 50; i++ {
		body := recSSE1.Body.String()
		if strings.Contains(body, "event: endpoint\n") {
			idx := strings.Index(body, "sessionId=")
			if idx != -1 {
				sessionID1 = strings.TrimSpace(body[idx+len("sessionId="):])
				if nl := strings.IndexAny(sessionID1, " \r\n"); nl != -1 {
					sessionID1 = sessionID1[:nl]
				}
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if sessionID1 == "" {
		t.Fatalf("Step 16 failed: did not receive session ID from /sse")
	}

	callMCP := func(sessID string, body any) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/mcp/messages?sessionId=%s", sessID), bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mcpMux.ServeHTTP(rec, req)
		return rec
	}

	// 2. Initialize
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	}
	recInit := callMCP(sessionID1, initReq)
	if recInit.Code != http.StatusOK {
		t.Fatalf("Step 16 failed: expected 200 on initialize, got %d: %s", recInit.Code, recInit.Body.String())
	}

	// 3. tools/list
	toolsListReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}
	recToolsList := callMCP(sessionID1, toolsListReq)
	if recToolsList.Code != http.StatusOK {
		t.Fatalf("Step 16 failed: expected 200 on tools/list, got %d: %s", recToolsList.Code, recToolsList.Body.String())
	}
	if !strings.Contains(recToolsList.Body.String(), "errorium_search_solution") {
		t.Fatalf("Step 16 failed: tools/list missing errorium_search_solution: %s", recToolsList.Body.String())
	}

	// 4. tools/call - call 1 (search)
	call1Req := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "errorium_search_solution",
			"arguments": map[string]any{"query": "TypeError"},
		},
	}
	recCall1 := callMCP(sessionID1, call1Req)
	if recCall1.Code != http.StatusOK {
		t.Fatalf("Step 16 failed: expected 200 on tool call 1, got %d: %s", recCall1.Code, recCall1.Body.String())
	}

	// tools/call - call 2 (explain)
	call2Req := map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "errorium_explain_error",
			"arguments": map[string]any{"error_message": "NullPointerException"},
		},
	}
	recCall2 := callMCP(sessionID1, call2Req)
	if recCall2.Code != http.StatusOK {
		t.Fatalf("Step 16 failed: expected 200 on tool call 2, got %d: %s", recCall2.Code, recCall2.Body.String())
	}

	// tools/call - call 3 (exceeds CallsPerMin = 2 -> 429)
	call3Req := map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "errorium_get_related_errors",
			"arguments": map[string]any{"error_name": "SyntaxError"},
		},
	}
	recCall3 := callMCP(sessionID1, call3Req)
	if recCall3.Code != http.StatusTooManyRequests {
		t.Fatalf("Step 16 failed: expected 429 on tool call 3 exceeding limit, got %d: %s", recCall3.Code, recCall3.Body.String())
	}
	if recCall3.Header().Get("Retry-After") != "5" {
		t.Errorf("Step 16 failed: expected Retry-After 5, got %q", recCall3.Header().Get("Retry-After"))
	}

	// 5. IP session limit: MaxSessionsPerIP = 2
	ctxSSE2, cancelSSE2 := context.WithCancel(context.Background())
	defer cancelSSE2()
	reqSSE2 := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctxSSE2)
	reqSSE2.RemoteAddr = "10.10.10.1:2222"
	recSSE2 := httptest.NewRecorder()
	go mcpMux.ServeHTTP(recSSE2, reqSSE2)
	time.Sleep(20 * time.Millisecond)

	// Session 3 from same IP (10.10.10.1) -> must be 429
	reqSSE3 := httptest.NewRequest(http.MethodGet, "/sse", nil)
	reqSSE3.RemoteAddr = "10.10.10.1:3333"
	recSSE3 := httptest.NewRecorder()
	mcpMux.ServeHTTP(recSSE3, reqSSE3)
	if recSSE3.Code != http.StatusTooManyRequests {
		t.Fatalf("Step 16 failed: expected 429 on 3rd session from same IP, got %d", recSSE3.Code)
	}

	// 6. Global session limit: MaxSessions = 3
	ctxSSE4, cancelSSE4 := context.WithCancel(context.Background())
	defer cancelSSE4()
	reqSSE4 := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctxSSE4)
	reqSSE4.RemoteAddr = "10.10.10.2:4444"
	recSSE4 := httptest.NewRecorder()
	go mcpMux.ServeHTTP(recSSE4, reqSSE4)
	time.Sleep(20 * time.Millisecond)

	// Session 4 from 10.10.10.3 -> exceeds global limit (3) -> 503
	reqSSE5 := httptest.NewRequest(http.MethodGet, "/sse", nil)
	reqSSE5.RemoteAddr = "10.10.10.3:5555"
	recSSE5 := httptest.NewRecorder()
	mcpMux.ServeHTTP(recSSE5, reqSSE5)
	if recSSE5.Code != http.StatusServiceUnavailable {
		t.Fatalf("Step 16 failed: expected 503 on exceeding global sessions limit, got %d", recSSE5.Code)
	}

	// 7. Session close releases counters
	cancelSSE1()
	cancelSSE2()
	time.Sleep(50 * time.Millisecond)

	// Opening a session from 10.10.10.1 now succeeds
	ctxSSE6, cancelSSE6 := context.WithCancel(context.Background())
	defer cancelSSE6()
	reqSSE6 := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctxSSE6)
	reqSSE6.RemoteAddr = "10.10.10.1:6666"
	recSSE6 := httptest.NewRecorder()
	go mcpMux.ServeHTTP(recSSE6, reqSSE6)
	time.Sleep(20 * time.Millisecond)

	if recSSE6.Code != http.StatusOK && recSSE6.Code != 0 {
		t.Fatalf("Step 16 failed: expected success after closing sessions, got %d", recSSE6.Code)
	}

	// -------------------------------------------------------------------------
	// Step 17: Wave B - Admin bootstrap -> stats (mock gRPC) -> users patch -> rag config PUT -> search uses updated config
	// -------------------------------------------------------------------------
	adminEmail := fmt.Sprintf("admin_%d@test.com", runID)
	adminPassword := "AdminSecret123!"

	// Bootstrap administrator (idempotency check)
	if err := app.BootstrapFirstAdmin(context.Background(), userRepo, adminEmail, adminPassword); err != nil {
		t.Fatalf("Step 17 failed: BootstrapFirstAdmin: %v", err)
	}
	if err := app.BootstrapFirstAdmin(context.Background(), userRepo, adminEmail, adminPassword); err != nil {
		t.Fatalf("Step 17 failed: BootstrapFirstAdmin repeat failed: %v", err)
	}

	// Login as admin
	recAdminLogin := doRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    adminEmail,
		"password": adminPassword,
	}, "")
	if recAdminLogin.Code != http.StatusOK {
		t.Fatalf("Step 17 failed: expected 200 on admin login, got %d: %s", recAdminLogin.Code, recAdminLogin.Body.String())
	}
	var adminLoginResp struct {
		AccessToken string       `json:"access_token"`
		User        *domain.User `json:"user"`
	}
	_ = json.Unmarshal(recAdminLogin.Body.Bytes(), &adminLoginResp)
	adminToken := adminLoginResp.AccessToken
	if adminLoginResp.User.Role != domain.RoleAdmin {
		t.Fatalf("Step 17 failed: expected role 'admin', got %q", adminLoginResp.User.Role)
	}

	// GET /api/v1/admin/stats
	recAdminStats := doRequest(http.MethodGet, "/api/v1/admin/stats", nil, adminToken)
	if recAdminStats.Code != http.StatusOK {
		t.Fatalf("Step 17 failed: expected 200 on /admin/stats, got %d: %s", recAdminStats.Code, recAdminStats.Body.String())
	}
	var statsResp struct {
		TotalDocuments int64   `json:"total_documents"`
		AvgDocLength   float64 `json:"avg_doc_length"`
		IsReady        bool    `json:"is_ready"`
	}
	_ = json.Unmarshal(recAdminStats.Body.Bytes(), &statsResp)
	if statsResp.TotalDocuments != 100 || !statsResp.IsReady {
		t.Errorf("Step 17 failed: unexpected stats: %+v", statsResp)
	}

	// PATCH /api/v1/admin/users/{id} -> deactivate User 2
	deactivate := false
	recPatchUser := doRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%s", reg2Resp.User.ID), map[string]any{
		"is_active": deactivate,
	}, adminToken)
	if recPatchUser.Code != http.StatusOK {
		t.Fatalf("Step 17 failed: expected 200 on /admin/users/{id}, got %d: %s", recPatchUser.Code, recPatchUser.Body.String())
	}
	// Verify deactivated user 2 cannot access /auth/me (403 Forbidden)
	recUser2Me := doRequest(http.MethodGet, "/api/v1/auth/me", nil, user2Access)
	if recUser2Me.Code != http.StatusForbidden {
		t.Fatalf("Step 17 failed: expected 403 Forbidden for deactivated user, got %d", recUser2Me.Code)
	}

	// PUT /api/v1/admin/rag/config
	newPrompt := "You are Errorium AI integration test prompt"
	putRagBody := map[string]any{
		"provider":      "stub",
		"model":         "stub-model",
		"system_prompt": newPrompt,
		"temperature":   0.7,
		"max_tokens":    1200,
	}
	recPutRag := doRequest(http.MethodPut, "/api/v1/admin/rag/config", putRagBody, adminToken)
	if recPutRag.Code != http.StatusOK {
		t.Fatalf("Step 17 failed: expected 200 on PUT /admin/rag/config, got %d: %s", recPutRag.Code, recPutRag.Body.String())
	}

	recGetRag := doRequest(http.MethodGet, "/api/v1/admin/rag/config", nil, adminToken)
	if recGetRag.Code != http.StatusOK {
		t.Fatalf("Step 17 failed: expected 200 on GET /admin/rag/config, got %d: %s", recGetRag.Code, recGetRag.Body.String())
	}
	var gotRag domain.RagConfig
	_ = json.Unmarshal(recGetRag.Body.Bytes(), &gotRag)
	if gotRag.SystemPrompt != newPrompt {
		t.Errorf("Step 17 failed: expected system_prompt %q, got %q", newPrompt, gotRag.SystemPrompt)
	}

	// -------------------------------------------------------------------------
	// Step 18: Wave B - SSE: /search/rag with Accept -> delta/done; chat messages -> user_message/delta/done; JSON mode preserved
	// -------------------------------------------------------------------------
	// 1. /search/rag with Accept: text/event-stream
	reqSSESearch := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", strings.NewReader(`{"query":"SyntaxError in Python"}`))
	reqSSESearch.Header.Set("Content-Type", "application/json")
	reqSSESearch.Header.Set("Accept", "text/event-stream")
	recSSESearch := httptest.NewRecorder()
	appHandler.ServeHTTP(recSSESearch, reqSSESearch)
	if recSSESearch.Code != http.StatusOK {
		t.Fatalf("Step 18 failed: expected 200 on SSE /search/rag, got %d: %s", recSSESearch.Code, recSSESearch.Body.String())
	}
	if ct := recSSESearch.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Step 18 failed: expected text/event-stream, got %q", ct)
	}
	searchSSEBody := recSSESearch.Body.String()
	if !strings.Contains(searchSSEBody, "event: delta\n") || !strings.Contains(searchSSEBody, "event: done\n") {
		t.Fatalf("Step 18 failed: expected delta and done events in search/rag SSE, got: %s", searchSSEBody)
	}

	// 2. /search/rag without Accept -> standard JSON mode
	recJSONSearch := doRequest(http.MethodPost, "/api/v1/search/rag", map[string]string{"query": "SyntaxError in Python"}, "")
	if recJSONSearch.Code != http.StatusOK {
		t.Fatalf("Step 18 failed: expected 200 on JSON /search/rag, got %d", recJSONSearch.Code)
	}
	if ct := recJSONSearch.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Step 18 failed: expected application/json, got %q", ct)
	}
	var searchRAGResp struct {
		Synthesis string `json:"synthesis"`
	}
	if err := json.Unmarshal(recJSONSearch.Body.Bytes(), &searchRAGResp); err != nil || searchRAGResp.Synthesis == "" {
		t.Fatalf("Step 18 failed: failed to parse JSON /search/rag response: %v", err)
	}

	// 3. /chats/{id}/messages with Accept: text/event-stream
	createChatBody2 := map[string]string{"title": "SSE Chat"}
	recCreateChat2 := doRequest(http.MethodPost, "/api/v1/chats", createChatBody2, user1Access)
	var chat2 domain.Chat
	_ = json.Unmarshal(recCreateChat2.Body.Bytes(), &chat2)

	reqSSEMsg := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/chats/%s/messages", chat2.ID), strings.NewReader(`{"query":"[Python] TypeError in code"}`))
	reqSSEMsg.Header.Set("Content-Type", "application/json")
	reqSSEMsg.Header.Set("Accept", "text/event-stream")
	reqSSEMsg.Header.Set("Authorization", "Bearer "+user1Access)
	recSSEMsg := httptest.NewRecorder()
	appHandler.ServeHTTP(recSSEMsg, reqSSEMsg)
	if recSSEMsg.Code != http.StatusCreated {
		t.Fatalf("Step 18 failed: expected 201 on SSE chat messages, got %d: %s", recSSEMsg.Code, recSSEMsg.Body.String())
	}
	chatSSEBody := recSSEMsg.Body.String()
	if !strings.Contains(chatSSEBody, "event: user_message\n") ||
		!strings.Contains(chatSSEBody, "event: delta\n") ||
		!strings.Contains(chatSSEBody, "event: done\n") {
		t.Fatalf("Step 18 failed: expected user_message, delta, done events in chat SSE, got: %s", chatSSEBody)
	}
}
