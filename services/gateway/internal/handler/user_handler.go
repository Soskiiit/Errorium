package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
)

// UserStore abstracts persistence for User entities.
// The full implementation lives in internal/repository/user_repo.go (@db-layer).
type UserStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateProfile(ctx context.Context, u *domain.User) error
}

// UserHandler handles /api/v1/users/* endpoints.
type UserHandler struct {
	users UserStore
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(users UserStore) *UserHandler {
	return &UserHandler{users: users}
}

// RegisterRoutes wires user routes on mux.
// An optional auth middleware may be provided to protect authenticated routes.
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux, authMW ...func(http.Handler) http.Handler) {
	wrap := func(handler http.Handler) http.Handler { return handler }
	if len(authMW) > 0 && authMW[0] != nil {
		wrap = authMW[0]
	}

	mux.Handle("GET /api/v1/users/me", wrap(http.HandlerFunc(h.GetMe)))
	mux.Handle("PATCH /api/v1/users/me", wrap(http.HandlerFunc(h.UpdateMe)))
	mux.Handle("POST /api/v1/users/avatar", wrap(http.HandlerFunc(h.UploadAvatar)))
}

// ---------------------------------------------------------------------------
// GET /api/v1/users/me
// ---------------------------------------------------------------------------

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/users/me
// ---------------------------------------------------------------------------

type updateMeRequest struct {
	Username *string `json:"username"`
	Role     *string `json:"role"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	var req updateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Role != nil {
		user.Role = domain.Role(*req.Role)
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.UpdateProfile(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// ---------------------------------------------------------------------------
// POST /api/v1/users/avatar
// ---------------------------------------------------------------------------

const (
	maxAvatarSize = 5 << 20 // 5 MiB
)

type avatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondError(w, http.StatusBadRequest, "avatar field is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read avatar")
		return
	}

	// Determine MIME type from Content-Type header or fallback to detection.
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	// Store as data URL (mock — no S3 upload in this iteration).
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))

	// Persist in DB.
	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}
	user.AvatarURL = &dataURL
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.UpdateProfile(r.Context(), user); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}

	respondJSON(w, http.StatusOK, avatarResponse{AvatarURL: dataURL})
}
