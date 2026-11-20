package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/internal/sso"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

// SSOHandler handles OAuth2/OIDC SSO login and callback endpoints.
type SSOHandler struct {
	providers       *sso.Registry
	ssoStateStore   repository.SsoStateStore
	ssoAccountStore repository.SSOAccountStore
	userStore       repository.UserStore
	sessionStore    repository.SessionStore
	jwt             *jwtpkg.Manager
	redirectBase    string
	frontendURL     string
	refreshExpDays  int
}

// NewSSOHandler constructs an SSOHandler.
func NewSSOHandler(
	providers *sso.Registry,
	ssoStateStore repository.SsoStateStore,
	ssoAccountStore repository.SSOAccountStore,
	userStore repository.UserStore,
	sessionStore repository.SessionStore,
	jwt *jwtpkg.Manager,
	redirectBase string,
	frontendURL string,
	refreshExpDays int,
) *SSOHandler {
	return &SSOHandler{
		providers:       providers,
		ssoStateStore:   ssoStateStore,
		ssoAccountStore: ssoAccountStore,
		userStore:       userStore,
		sessionStore:    sessionStore,
		jwt:             jwt,
		redirectBase:    redirectBase,
		frontendURL:     frontendURL,
		refreshExpDays:  refreshExpDays,
	}
}

// RegisterRoutes wires SSO endpoints into mux.
func (h *SSOHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/sso/login/{provider}", h.login)
	mux.HandleFunc("GET /api/v1/auth/sso/callback/{provider}", h.callback)
}

func isSupportedProvider(p domain.SSOProvider) bool {
	return p == domain.SSOProviderGitHub || p == domain.SSOProviderGoogle || p == domain.SSOProviderYandex
}

// ── GET /api/v1/auth/sso/login/{provider} ────────────────────────────────────

func (h *SSOHandler) login(w http.ResponseWriter, r *http.Request) {
	providerName := domain.SSOProvider(r.PathValue("provider"))
	if !isSupportedProvider(providerName) {
		respondError(w, http.StatusBadRequest, "unknown provider")
		return
	}

	p, ok := h.providers.Get(providerName)
	if !ok || p.ClientID() == "" {
		respondError(w, http.StatusServiceUnavailable, "provider not configured")
		return
	}

	state := uuid.New().String()
	ssoState := &domain.SSOAuthState{
		State:     state,
		Provider:  providerName,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := h.ssoStateStore.Create(r.Context(), ssoState); err != nil {
		log.Printf("sso_handler.login: failed to create state: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to initiate sso login")
		return
	}

	redirectURI := fmt.Sprintf("%s/api/v1/auth/sso/callback/%s", strings.TrimRight(h.redirectBase, "/"), providerName)
	authURL := p.AuthURL(state, redirectURI)

	http.Redirect(w, r, authURL, http.StatusFound)
}

// ── GET /api/v1/auth/sso/callback/{provider} ─────────────────────────────────

func (h *SSOHandler) callback(w http.ResponseWriter, r *http.Request) {
	providerName := domain.SSOProvider(r.PathValue("provider"))
	if !isSupportedProvider(providerName) {
		respondError(w, http.StatusBadRequest, "unknown provider")
		return
	}

	p, ok := h.providers.Get(providerName)
	if !ok || p.ClientID() == "" {
		respondError(w, http.StatusServiceUnavailable, "provider not configured")
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		respondError(w, http.StatusUnauthorized, "invalid or expired sso state")
		return
	}

	ssoState, err := h.ssoStateStore.Get(r.Context(), state)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired sso state")
		return
	}

	if time.Now().After(ssoState.ExpiresAt) || ssoState.Provider != providerName {
		_ = h.ssoStateStore.Delete(r.Context(), state)
		respondError(w, http.StatusUnauthorized, "invalid or expired sso state")
		return
	}

	// Delete state immediately for single-use guarantee
	_ = h.ssoStateStore.Delete(r.Context(), state)

	redirectURI := fmt.Sprintf("%s/api/v1/auth/sso/callback/%s", strings.TrimRight(h.redirectBase, "/"), providerName)
	accessToken, err := p.Exchange(r.Context(), code, redirectURI)
	if err != nil {
		log.Printf("sso_handler.callback: exchange failed for %s: %v", providerName, err)
		respondError(w, http.StatusUnauthorized, "sso exchange failed")
		return
	}

	userInfo, err := p.FetchUserInfo(r.Context(), accessToken)
	if err != nil || userInfo.ProviderUserID == "" {
		log.Printf("sso_handler.callback: fetch userinfo failed for %s: %v", providerName, err)
		respondError(w, http.StatusUnauthorized, "sso exchange failed")
		return
	}

	user, err := h.findOrCreateUserAndLink(r.Context(), providerName, userInfo)
	if err != nil {
		log.Printf("sso_handler.callback: findOrCreateUserAndLink error: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to process user account")
		return
	}

	if !user.IsActive {
		respondError(w, http.StatusForbidden, "account is disabled")
		return
	}

	_ = h.userStore.UpdateLastLogin(r.Context(), user.ID, time.Now())

	jwtAccess, err := h.jwt.GenerateAccess(user)
	if err != nil {
		log.Printf("sso_handler.callback: failed to generate access token: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	jwtRefresh, err := h.jwt.GenerateRefresh()
	if err != nil {
		log.Printf("sso_handler.callback: failed to generate refresh token: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to generate refresh token")
		return
	}

	days := h.refreshExpDays
	if days <= 0 {
		days = 30
	}
	ttl := time.Duration(days) * 24 * time.Hour

	sess := &domain.UserSession{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: h.jwt.RefreshTokenHash(jwtRefresh),
		IsRevoked:        false,
		ExpiresAt:        time.Now().Add(ttl),
		CreatedAt:        time.Now(),
		LastUsedAt:       time.Now(),
	}
	if ua := r.UserAgent(); ua != "" {
		sess.UserAgent = &ua
	}
	if err := h.sessionStore.Create(r.Context(), sess); err != nil {
		log.Printf("sso_handler.callback: failed to create session: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	redirectTarget := fmt.Sprintf("%s/auth/callback#access_token=%s&refresh_token=%s",
		strings.TrimRight(h.frontendURL, "/"),
		jwtAccess,
		jwtRefresh,
	)

	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func (h *SSOHandler) findOrCreateUserAndLink(ctx context.Context, provider domain.SSOProvider, info *sso.UserInfo) (*domain.User, error) {
	// 1. Check if linked account exists by (provider, provider_user_id)
	ssoAcc, err := h.ssoAccountStore.GetByProviderAndUserID(ctx, provider, info.ProviderUserID)
	if err == nil && ssoAcc != nil {
		return h.userStore.GetByID(ctx, ssoAcc.UserID)
	}

	var user *domain.User

	// 2. If not linked, check if user exists by email
	if info.Email != "" {
		existing, err := h.userStore.GetByEmail(ctx, strings.ToLower(info.Email))
		if err == nil && existing != nil {
			user = existing
		}
	}

	// 3. If still no user, create a new user
	if user == nil {
		email := strings.ToLower(info.Email)
		if email == "" {
			email = fmt.Sprintf("%s_%s@sso.errorium.local", provider, info.ProviderUserID)
		}
		username := strings.TrimSpace(info.Username)
		if username == "" {
			username = fmt.Sprintf("%s_user_%s", provider, info.ProviderUserID)
		}
		if len(username) > 100 {
			username = username[:100]
		}

		newUser := &domain.User{
			ID:           uuid.New(),
			Email:        email,
			Username:     username,
			PasswordHash: nil,
			Role:         domain.RoleUser,
			Profession:   "Backend developer",
			Gender:       "Not specified",
			Language:     "English",
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := h.userStore.Create(ctx, newUser); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		user = newUser
	}

	// 4. Link SSO account to user
	var profileData json.RawMessage
	if len(info.RawData) > 0 {
		profileData = info.RawData
	} else {
		profileData = json.RawMessage("{}")
	}

	newAcc := &domain.UserSSOAccount{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       provider,
		ProviderUserID: info.ProviderUserID,
		ProfileData:    profileData,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if info.Email != "" {
		emailCopy := strings.ToLower(info.Email)
		newAcc.Email = &emailCopy
	}

	if err := h.ssoAccountStore.Create(ctx, newAcc); err != nil {
		return nil, fmt.Errorf("link sso account: %w", err)
	}

	return user, nil
}
