package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/pkg/mailer"
)

// PasswordResetHandler handles password forgot, reset, and change requests.
type PasswordResetHandler struct {
	users       repository.UserStore
	resetTokens repository.PasswordResetStore
	sessions    repository.SessionStore
	mailer      mailer.Mailer
	frontendURL string
}

// NewPasswordResetHandler constructs a new PasswordResetHandler.
func NewPasswordResetHandler(
	users repository.UserStore,
	resetTokens repository.PasswordResetStore,
	sessions repository.SessionStore,
	mailer mailer.Mailer,
	frontendURL string,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		users:       users,
		resetTokens: resetTokens,
		sessions:    sessions,
		mailer:      mailer,
		frontendURL: frontendURL,
	}
}

// RegisterRoutes registers password reset and change endpoints on the mux.
func (h *PasswordResetHandler) RegisterRoutes(mux *http.ServeMux, authMW ...func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/v1/auth/password/forgot", h.forgot)
	mux.HandleFunc("POST /api/v1/auth/password/reset", h.reset)

	wrap := func(handler http.Handler) http.Handler { return handler }
	if len(authMW) > 0 && authMW[0] != nil {
		wrap = authMW[0]
	}
	mux.Handle("POST /api/v1/auth/password/change", wrap(http.HandlerFunc(h.change)))
}

// ── POST /api/v1/auth/password/forgot ─────────────────────────────────────────

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *PasswordResetHandler) forgot(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || !emailRE.MatchString(email) {
		writeError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), strings.ToLower(email))
	if err != nil || user == nil {
		// Always return 202 Accepted to prevent user enumeration
		w.WriteHeader(http.StatusAccepted)
		return
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate reset token")
		return
	}

	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	_ = h.resetTokens.DeleteExpired(r.Context())

	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := h.resetTokens.Create(r.Context(), resetToken); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save reset token")
		return
	}

	frontendURL := strings.TrimRight(h.frontendURL, "/")
	link := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, rawToken)

	if err := h.mailer.SendReset(r.Context(), user.Email, link); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send reset email")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// ── POST /api/v1/auth/password/reset ──────────────────────────────────────────

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *PasswordResetHandler) reset(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		writeError(w, http.StatusUnauthorized, "invalid or expired reset token")
		return
	}

	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	tok, err := h.resetTokens.GetByTokenHash(r.Context(), tokenHash)
	if err != nil || tok == nil || tok.UsedAt != nil || time.Now().After(tok.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "invalid or expired reset token")
		return
	}

	n := utf8.RuneCountInString(req.NewPassword)
	if n < 8 || n > 128 || len(req.NewPassword) > 128 {
		writeError(w, http.StatusBadRequest, "password must be between 8 and 128 characters")
		return
	}

	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.users.UpdatePasswordHash(r.Context(), tok.UserID, newHash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	if err := h.resetTokens.MarkUsed(r.Context(), tok.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark reset token used")
		return
	}

	if err := h.sessions.RevokeByUser(r.Context(), tok.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke sessions")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── POST /api/v1/auth/password/change ─────────────────────────────────────────

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *PasswordResetHandler) change(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	if user.PasswordHash == nil || !checkPassword(*user.PasswordHash, req.CurrentPassword) {
		writeError(w, http.StatusUnauthorized, "invalid current password")
		return
	}

	n := utf8.RuneCountInString(req.NewPassword)
	if n < 8 || n > 128 || len(req.NewPassword) > 128 {
		writeError(w, http.StatusBadRequest, "password must be between 8 and 128 characters")
		return
	}

	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.users.UpdatePasswordHash(r.Context(), user.ID, newHash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	if err := h.sessions.RevokeByUser(r.Context(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke sessions")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
