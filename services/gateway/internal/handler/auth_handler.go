package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

var emailRE = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// AuthHandler implements the Auth API endpoints.
type AuthHandler struct {
	users    repository.UserStore
	sessions repository.SessionStore
	jwt      *jwtpkg.Manager
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(
	users repository.UserStore,
	sessions repository.SessionStore,
	jwt *jwtpkg.Manager,
) *AuthHandler {
	return &AuthHandler{users: users, sessions: sessions, jwt: jwt}
}

// RegisterRoutes wires all /api/v1/auth/... routes into mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	authMW := middleware.Auth(h.jwt)

	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.refresh)
	mux.Handle("POST /api/v1/auth/logout", authMW(http.HandlerFunc(h.logout)))
	mux.Handle("GET /api/v1/auth/me", authMW(http.HandlerFunc(h.me)))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type tokenPair struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *domain.User `json:"user"`
}

func (h *AuthHandler) issueTokenPair(w http.ResponseWriter, r *http.Request, user *domain.User, status int) {
	access, err := h.jwt.GenerateAccess(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}
	refresh, err := h.jwt.GenerateRefresh()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	ua := r.UserAgent()
	sess := &domain.UserSession{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: h.jwt.RefreshTokenHash(refresh),
		IsRevoked:        false,
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
		CreatedAt:        time.Now(),
		LastUsedAt:       time.Now(),
	}
	if ua != "" {
		sess.UserAgent = &ua
	}
	if err := h.sessions.Create(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, status, tokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         user,
	})
}

// ── POST /api/v1/auth/register ────────────────────────────────────────────────

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if !emailRE.MatchString(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: &hash,
		Role:         domain.RoleUser,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.users.Create(r.Context(), user); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	h.issueTokenPair(w, r, user, http.StatusCreated)
}

// ── POST /api/v1/auth/login ───────────────────────────────────────────────────

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "account is disabled")
		return
	}
	if user.PasswordHash == nil || !checkPassword(*user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	now := time.Now()
	_ = h.users.UpdateLastLogin(r.Context(), user.ID, now)
	user.LastLoginAt = &now

	h.issueTokenPair(w, r, user, http.StatusOK)
}

// ── POST /api/v1/auth/refresh ─────────────────────────────────────────────────

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	hash := h.jwt.RefreshTokenHash(req.RefreshToken)
	sess, err := h.sessions.GetByTokenHash(r.Context(), hash)
	if err != nil || sess == nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if sess.IsRevoked {
		writeError(w, http.StatusUnauthorized, "refresh token has been revoked")
		return
	}
	if time.Now().After(sess.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	// Revoke old session.
	if err := h.sessions.RevokeByTokenHash(r.Context(), hash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	user, err := h.users.GetByID(r.Context(), sess.UserID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	h.issueTokenPair(w, r, user, http.StatusOK)
}

// ── POST /api/v1/auth/logout ──────────────────────────────────────────────────

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// Best-effort decode; it's fine if the body is empty or malformed.
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.RefreshToken != "" {
		hash := h.jwt.RefreshTokenHash(req.RefreshToken)
		_ = h.sessions.RevokeByTokenHash(r.Context(), hash)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── GET /api/v1/auth/me ───────────────────────────────────────────────────────

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}
