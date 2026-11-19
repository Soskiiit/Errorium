package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
	"github.com/soskiiit/errorium/services/gateway/pkg/storage"
)

type fakeChatStore struct {
	lastUserID uuid.UUID
}

func (f *fakeChatStore) Create(ctx context.Context, c *domain.Chat) error {
	f.lastUserID = c.UserID
	return nil
}

func (f *fakeChatStore) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, search string) ([]*domain.Chat, int, error) {
	f.lastUserID = userID
	return []*domain.Chat{
		{
			ID:        uuid.New(),
			UserID:    userID,
			Title:     "Test Chat",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}, 1, nil
}

func (f *fakeChatStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	return &domain.Chat{ID: id, UserID: f.lastUserID, Title: "Test Chat"}, nil
}

func (f *fakeChatStore) GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error) {
	return &domain.Chat{ID: id, UserID: userID, Title: "Test Chat"}, nil
}

func (f *fakeChatStore) Update(ctx context.Context, c *domain.Chat) error {
	return nil
}

func (f *fakeChatStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (f *fakeChatStore) AddMessage(ctx context.Context, msg *domain.ChatMessage) error {
	return nil
}

func (f *fakeChatStore) GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error) {
	return nil, nil
}

type fakeTagStore struct{}

func (f *fakeTagStore) GetOrCreate(ctx context.Context, name string) (*domain.Tag, error) {
	return &domain.Tag{ID: uuid.New(), Name: name}, nil
}

func (f *fakeTagStore) AttachToMessage(ctx context.Context, chatMessageID uuid.UUID, tagNames []string) error {
	return nil
}

func (f *fakeTagStore) GetByMessage(ctx context.Context, chatMessageID uuid.UUID) ([]*domain.Tag, error) {
	return nil, nil
}

type fakeUserStore struct {
	lastUserID uuid.UUID
	user       *domain.User
}

func (f *fakeUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	f.lastUserID = id
	if f.user != nil {
		return f.user, nil
	}
	return &domain.User{
		ID:        id,
		Username:  "alice",
		Email:     "alice@example.com",
		Role:      domain.RoleUser,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.user != nil && f.user.Email == email {
		return f.user, nil
	}
	return nil, nil
}

func (f *fakeUserStore) UpdateProfile(ctx context.Context, u *domain.User) error {
	f.lastUserID = u.ID
	f.user = u
	return nil
}

func setupTestAuthManager(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	return jwtpkg.NewManager(config.JWTConfig{
		SecretKey:              "super-secret-test-key-32-bytes-long!",
		AccessTokenExpireMin:   15,
		RefreshTokenExpireDays: 7,
	})
}

func TestAuthContext_ChatHandler(t *testing.T) {
	jwtMgr := setupTestAuthManager(t)
	authMW := middleware.Auth(jwtMgr)

	fakeChats := &fakeChatStore{}
	fakeTags := &fakeTagStore{}
	chatHandler := handler.NewChatHandler(fakeChats, fakeTags, search.NewStub())

	mux := http.NewServeMux()
	chatHandler.RegisterRoutes(mux, authMW)

	expectedUserID := uuid.New()
	user := &domain.User{
		ID:       expectedUserID,
		Email:    "chatuser@example.com",
		Username: "chatuser",
		Role:     domain.RoleUser,
	}

	validToken, err := jwtMgr.GenerateAccess(user)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectUserID   bool
	}{
		{
			name:           "valid JWT sets user_id in context and chat handler reads it",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectUserID:   true,
		},
		{
			name:           "missing Authorization header returns 401",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid JWT returns 401",
			authHeader:     "Bearer invalid.token.value",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.expectUserID {
				if fakeChats.lastUserID != expectedUserID {
					t.Fatalf("expected handler to receive user ID %s, got %s", expectedUserID, fakeChats.lastUserID)
				}
			}
		})
	}
}

func TestAuthContext_UserHandler(t *testing.T) {
	jwtMgr := setupTestAuthManager(t)
	authMW := middleware.Auth(jwtMgr)

	expectedUserID := uuid.New()
	expectedUser := &domain.User{
		ID:        expectedUserID,
		Username:  "bob",
		Email:     "bob@example.com",
		Role:      domain.RoleUser,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	fakeUsers := &fakeUserStore{user: expectedUser}
	userHandler := handler.NewUserHandler(fakeUsers, storage.NewMemoryStore())

	mux := http.NewServeMux()
	userHandler.RegisterRoutes(mux, authMW)

	validToken, err := jwtMgr.GenerateAccess(expectedUser)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		verifyUser     bool
	}{
		{
			name:           "valid JWT sets user_id in context and user handler reads it",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			verifyUser:     true,
		},
		{
			name:           "missing Authorization header returns 401",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			verifyUser:     false,
		},
		{
			name:           "invalid JWT returns 401",
			authHeader:     "Bearer malformed-token-string",
			expectedStatus: http.StatusUnauthorized,
			verifyUser:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}

			if tc.verifyUser {
				if fakeUsers.lastUserID != expectedUserID {
					t.Fatalf("expected handler to lookup user ID %s, got %s", expectedUserID, fakeUsers.lastUserID)
				}
				var respUser domain.User
				if err := json.Unmarshal(rec.Body.Bytes(), &respUser); err != nil {
					t.Fatalf("failed to decode user response: %v", err)
				}
				if respUser.ID != expectedUserID {
					t.Fatalf("expected response user ID %s, got %s", expectedUserID, respUser.ID)
				}
				if respUser.Username != "bob" {
					t.Fatalf("expected username bob, got %s", respUser.Username)
				}
			}
		})
	}
}
