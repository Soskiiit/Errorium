package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
)

// UserInfo holds normalized user profile information received from an IdP.
type UserInfo struct {
	ProviderUserID string
	Email          string
	Username       string
	RawData        json.RawMessage
}

// ProviderConfig configures an OAuth2/OIDC SSO provider.
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	EmailsURL    string // Specific to providers like GitHub with separate email endpoints
	Scopes       []string
	HTTPClient   *http.Client
}

// Provider represents an SSO provider capable of performing OAuth2 code flows.
type Provider interface {
	Name() domain.SSOProvider
	ClientID() string
	ClientSecret() string
	AuthURL(state, redirectURI string) string
	Exchange(ctx context.Context, code, redirectURI string) (string, error)
	FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// Registry stores registered SSO providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[domain.SSOProvider]Provider
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[domain.SSOProvider]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get returns the provider by name, if found.
func (r *Registry) Get(name domain.SSOProvider) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// ── Base Provider ────────────────────────────────────────────────────────────

type baseProvider struct {
	name         domain.SSOProvider
	clientID     string
	clientSecret string
	authURL      string
	tokenURL     string
	userInfoURL  string
	scopes       []string
	httpClient   *http.Client
}

func (b *baseProvider) Name() domain.SSOProvider {
	return b.name
}

func (b *baseProvider) ClientID() string {
	return b.clientID
}

func (b *baseProvider) ClientSecret() string {
	return b.clientSecret
}

func (b *baseProvider) AuthURL(state, redirectURI string) string {
	u, err := url.Parse(b.authURL)
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("client_id", b.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(b.scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func (b *baseProvider) client() *http.Client {
	if b.httpClient != nil {
		return b.httpClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

type tokenExchangeResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (b *baseProvider) Exchange(ctx context.Context, code, redirectURI string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", b.clientID)
	data.Set("client_secret", b.clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := b.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokResp tokenExchangeResponse
	if err := json.Unmarshal(body, &tokResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tokResp.Error != "" {
		return "", fmt.Errorf("token endpoint error: %s: %s", tokResp.Error, tokResp.ErrorDescription)
	}

	if tokResp.AccessToken == "" {
		return "", errors.New("empty access_token in token response")
	}

	return tokResp.AccessToken, nil
}

// ── Helper to format ID ──────────────────────────────────────────────────────

func stringifyID(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatInt(int64(val), 10)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case json.Number:
		return val.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

// ── GitHub Provider ──────────────────────────────────────────────────────────

type GitHubProvider struct {
	baseProvider
	emailsURL string
}

type githubUserResponse struct {
	ID    any     `json:"id"`
	Login string  `json:"login"`
	Name  string  `json:"name"`
	Email *string `json:"email"`
}

type githubEmailItem struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewGitHubProvider(cfg ProviderConfig) *GitHubProvider {
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = "https://github.com/login/oauth/authorize"
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://github.com/login/oauth/access_token"
	}
	userInfoURL := cfg.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://api.github.com/user"
	}
	emailsURL := cfg.EmailsURL
	if emailsURL == "" {
		emailsURL = "https://api.github.com/user/emails"
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}

	return &GitHubProvider{
		baseProvider: baseProvider{
			name:         domain.SSOProviderGitHub,
			clientID:     cfg.ClientID,
			clientSecret: cfg.ClientSecret,
			authURL:      authURL,
			tokenURL:     tokenURL,
			userInfoURL:  userInfoURL,
			scopes:       scopes,
			httpClient:   cfg.HTTPClient,
		},
		emailsURL: emailsURL,
	}
}

func (p *GitHubProvider) FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read userinfo response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var ghUser githubUserResponse
	if err := json.Unmarshal(body, &ghUser); err != nil {
		return nil, fmt.Errorf("unmarshal userinfo: %w", err)
	}

	providerUserID := stringifyID(ghUser.ID)
	username := strings.TrimSpace(ghUser.Login)
	if username == "" {
		username = strings.TrimSpace(ghUser.Name)
	}

	email := ""
	if ghUser.Email != nil {
		email = strings.TrimSpace(*ghUser.Email)
	}

	// Fallback to /user/emails if primary email is not exposed in /user
	if email == "" && p.emailsURL != "" {
		emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.emailsURL, nil)
		if err == nil {
			emailReq.Header.Set("Authorization", "Bearer "+accessToken)
			emailReq.Header.Set("Accept", "application/json")
			emailResp, err := p.client().Do(emailReq)
			if err == nil {
				defer emailResp.Body.Close()
				if emailResp.StatusCode == http.StatusOK {
					var emails []githubEmailItem
					if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
						for _, item := range emails {
							if item.Primary && item.Verified {
								email = strings.TrimSpace(item.Email)
								break
							}
						}
						// If no primary+verified found, take first verified
						if email == "" {
							for _, item := range emails {
								if item.Verified {
									email = strings.TrimSpace(item.Email)
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return &UserInfo{
		ProviderUserID: providerUserID,
		Email:          email,
		Username:       username,
		RawData:        json.RawMessage(body),
	}, nil
}

// ── Google Provider ──────────────────────────────────────────────────────────

type GoogleProvider struct {
	baseProvider
}

type googleUserResponse struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewGoogleProvider(cfg ProviderConfig) *GoogleProvider {
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	userInfoURL := cfg.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	return &GoogleProvider{
		baseProvider: baseProvider{
			name:         domain.SSOProviderGoogle,
			clientID:     cfg.ClientID,
			clientSecret: cfg.ClientSecret,
			authURL:      authURL,
			tokenURL:     tokenURL,
			userInfoURL:  userInfoURL,
			scopes:       scopes,
			httpClient:   cfg.HTTPClient,
		},
	}
}

func (p *GoogleProvider) FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create google userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute google userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google userinfo response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("google userinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var gUser googleUserResponse
	if err := json.Unmarshal(body, &gUser); err != nil {
		return nil, fmt.Errorf("unmarshal google userinfo: %w", err)
	}

	username := strings.TrimSpace(gUser.Name)
	if username == "" && gUser.Email != "" {
		username = strings.Split(gUser.Email, "@")[0]
	}

	return &UserInfo{
		ProviderUserID: gUser.Sub,
		Email:          strings.TrimSpace(gUser.Email),
		Username:       username,
		RawData:        json.RawMessage(body),
	}, nil
}

// ── Yandex Provider ──────────────────────────────────────────────────────────

type YandexProvider struct {
	baseProvider
}

type yandexUserResponse struct {
	ID           any      `json:"id"`
	Login        string   `json:"login"`
	DefaultLogin string   `json:"default_login"`
	RealName     string   `json:"real_name"`
	DefaultEmail string   `json:"default_email"`
	Emails       []string `json:"emails"`
}

func NewYandexProvider(cfg ProviderConfig) *YandexProvider {
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = "https://oauth.yandex.ru/authorize"
	}
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth.yandex.ru/token"
	}
	userInfoURL := cfg.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://login.yandex.ru/info"
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"login:email", "login:info"}
	}

	return &YandexProvider{
		baseProvider: baseProvider{
			name:         domain.SSOProviderYandex,
			clientID:     cfg.ClientID,
			clientSecret: cfg.ClientSecret,
			authURL:      authURL,
			tokenURL:     tokenURL,
			userInfoURL:  userInfoURL,
			scopes:       scopes,
			httpClient:   cfg.HTTPClient,
		},
	}
}

func (p *YandexProvider) FetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create yandex userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute yandex userinfo request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read yandex userinfo response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yandex userinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var yaUser yandexUserResponse
	if err := json.Unmarshal(body, &yaUser); err != nil {
		return nil, fmt.Errorf("unmarshal yandex userinfo: %w", err)
	}

	providerUserID := stringifyID(yaUser.ID)

	username := strings.TrimSpace(yaUser.DefaultLogin)
	if username == "" {
		username = strings.TrimSpace(yaUser.Login)
	}
	if username == "" {
		username = strings.TrimSpace(yaUser.RealName)
	}

	email := strings.TrimSpace(yaUser.DefaultEmail)
	if email == "" && len(yaUser.Emails) > 0 {
		email = strings.TrimSpace(yaUser.Emails[0])
	}

	return &UserInfo{
		ProviderUserID: providerUserID,
		Email:          email,
		Username:       username,
		RawData:        json.RawMessage(body),
	}, nil
}
