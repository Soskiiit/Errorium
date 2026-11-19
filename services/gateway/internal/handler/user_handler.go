package handler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/pkg/storage"
)

var userEmailRE = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// UserStore abstracts persistence for User entities.
type UserStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateProfile(ctx context.Context, u *domain.User) error
}

// UserHandler handles /api/v1/users/* endpoints.
type UserHandler struct {
	users       UserStore
	avatarStore storage.AvatarStore
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(users UserStore, avatarStore storage.AvatarStore) *UserHandler {
	return &UserHandler{
		users:       users,
		avatarStore: avatarStore,
	}
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
	mux.Handle("GET /api/v1/users/{id}/avatar", http.HandlerFunc(h.GetAvatar))
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

type updateProfileRequest struct {
	Username   *string `json:"username"`
	Email      *string `json:"email"`
	Gender     *string `json:"gender"`
	Language   *string `json:"language"`
	Profession *string `json:"profession"`
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, ok := raw["role"]; ok {
		respondError(w, http.StatusBadRequest, "role is not editable")
		return
	}

	var req updateProfileRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == nil && req.Email == nil && req.Gender == nil && req.Language == nil && req.Profession == nil {
		respondError(w, http.StatusBadRequest, "at least one field required")
		return
	}

	if req.Username != nil {
		u := strings.TrimSpace(*req.Username)
		runeCount := utf8.RuneCountInString(u)
		if runeCount < 1 || runeCount > 100 {
			respondError(w, http.StatusBadRequest, "username must be between 1 and 100 characters")
			return
		}
		user.Username = u
	}

	if req.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*req.Email))
		if !userEmailRE.MatchString(e) || len(e) > 255 {
			respondError(w, http.StatusBadRequest, "invalid email format")
			return
		}
		if e != strings.ToLower(user.Email) {
			existing, err := h.users.GetByEmail(r.Context(), e)
			if err == nil && existing != nil && existing.ID != user.ID {
				respondError(w, http.StatusConflict, "email already in use")
				return
			}
		}
		user.Email = e
	}

	if req.Gender != nil {
		switch *req.Gender {
		case "Male", "Female", "Not specified":
			user.Gender = *req.Gender
		default:
			respondError(w, http.StatusBadRequest, "invalid gender")
			return
		}
	}

	if req.Language != nil {
		switch *req.Language {
		case "English", "Русский":
			user.Language = *req.Language
		default:
			respondError(w, http.StatusBadRequest, "invalid language")
			return
		}
	}

	if req.Profession != nil {
		p := strings.TrimSpace(*req.Profession)
		runeCount := utf8.RuneCountInString(p)
		if runeCount < 1 || runeCount > 100 {
			respondError(w, http.StatusBadRequest, "profession must be between 1 and 100 characters")
			return
		}
		user.Profession = p
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

const maxAvatarSize = 5 << 20 // 5 MiB

type avatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

func (h *UserHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		respondError(w, http.StatusBadRequest, "file size exceeds 5MB limit")
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		respondError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read avatar")
		return
	}
	if len(data) > maxAvatarSize {
		respondError(w, http.StatusBadRequest, "file size exceeds 5MB limit")
		return
	}

	mimeType := http.DetectContentType(data)
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
	default:
		respondError(w, http.StatusBadRequest, "unsupported image format")
		return
	}

	if h.avatarStore == nil {
		respondError(w, http.StatusBadGateway, "failed to upload avatar")
		return
	}

	key := fmt.Sprintf("avatars/%s", userID.String())
	if err := h.avatarStore.Put(r.Context(), key, mimeType, data); err != nil {
		respondError(w, http.StatusBadGateway, "failed to upload avatar")
		return
	}

	avatarURL := fmt.Sprintf("/api/v1/users/%s/avatar", userID.String())
	user, err := h.users.GetByID(r.Context(), userID)
	if err == nil && user != nil {
		user.AvatarURL = &avatarURL
		user.UpdatedAt = time.Now().UTC()
		_ = h.users.UpdateProfile(r.Context(), user)
	}

	respondJSON(w, http.StatusOK, avatarResponse{AvatarURL: avatarURL})
}

// ---------------------------------------------------------------------------
// GET /api/v1/users/{id}/avatar
// ---------------------------------------------------------------------------

func (h *UserHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	uid, err := uuid.Parse(idStr)
	if err != nil || uid == uuid.Nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if h.avatarStore == nil {
		respondError(w, http.StatusBadGateway, "failed to retrieve avatar")
		return
	}

	key := fmt.Sprintf("avatars/%s", uid.String())
	data, contentType, err := h.avatarStore.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			respondError(w, http.StatusNotFound, "avatar not found")
			return
		}
		respondError(w, http.StatusBadGateway, "failed to retrieve avatar")
		return
	}

	etag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
