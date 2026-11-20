package llm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GigaChatConfig holds configuration for GigaChat OAuth and completion.
type GigaChatConfig struct {
	ClientID     string
	ClientSecret string
	Scope        string
	OAuthURL     string
	BaseURL      string
	Model        string
	CACert       string
}

// TokenManager manages OAuth tokens for GigaChat with thread-safe caching.
type TokenManager struct {
	clientID     string
	clientSecret string
	scope        string
	oauthURL     string
	httpClient   *http.Client

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(cfg GigaChatConfig, client *http.Client) *TokenManager {
	oauthURL := cfg.OAuthURL
	if oauthURL == "" {
		oauthURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	}
	scope := cfg.Scope
	if scope == "" {
		scope = "GIGACHAT_API_PERS"
	}

	return &TokenManager{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		scope:        scope,
		oauthURL:     oauthURL,
		httpClient:   client,
	}
}

// GetToken returns a valid cached access token or requests a new one.
func (m *TokenManager) GetToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cache valid until expires_at - 60s
	if m.cachedToken != "" && time.Now().Before(m.expiresAt.Add(-60*time.Second)) {
		return m.cachedToken, nil
	}

	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(m.clientID+":"+m.clientSecret))
	rquid := uuid.New().String()

	data := url.Values{}
	data.Set("scope", m.scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.oauthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create oauth request: %w", err)
	}

	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("RqUID", rquid)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("%w: oauth request failed: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("%w: oauth server error %d", ErrUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gigachat oauth error: status %d", resp.StatusCode)
	}

	var oauthResp struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"` // Epoch ms
	}
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return "", fmt.Errorf("failed to decode oauth response: %w", err)
	}

	m.cachedToken = oauthResp.AccessToken
	m.expiresAt = time.UnixMilli(oauthResp.ExpiresAt)

	return m.cachedToken, nil
}

// InvalidateCache invalidates the cached OAuth token (useful for tests).
func (m *TokenManager) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cachedToken = ""
	m.expiresAt = time.Time{}
}

// GigaChat implements Client for GigaChat using OAuth and OpenAI-compatible completions.
type GigaChat struct {
	tokenMgr   *TokenManager
	baseURL    string
	model      string
	httpClient *http.Client
}

var _ Client = (*GigaChat)(nil)

// NewGigaChat constructs a GigaChat client with Russian trusted root CA certificate.
func NewGigaChat(cfg GigaChatConfig) (*GigaChat, error) {
	if cfg.CACert == "" {
		return nil, errors.New("gigachat ca certificate file path (GIGACHAT_CA_CERT) is required")
	}

	certData, err := os.ReadFile(cfg.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to read gigachat ca certificate file: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certData) {
		return nil, errors.New("failed to parse gigachat ca certificate: invalid or corrupt PEM data")
	}

	// InsecureSkipVerify is STRICTLY PROHIBITED
	tlsConfig := &tls.Config{
		RootCAs:            certPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
	}

	return NewGigaChatWithClient(cfg, client)
}

// NewGigaChatWithClient constructs a GigaChat client with a custom http.Client.
func NewGigaChatWithClient(cfg GigaChatConfig, client *http.Client) (*GigaChat, error) {
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.giga.chat/v1"
	}
	model := cfg.Model
	if model == "" {
		model = "GigaChat"
	}

	tokenMgr := NewTokenManager(cfg, client)

	return &GigaChat{
		tokenMgr:   tokenMgr,
		baseURL:    baseURL,
		model:      model,
		httpClient: client,
	}, nil
}

// GetToken returns the current OAuth token.
func (g *GigaChat) GetToken(ctx context.Context) (string, error) {
	return g.tokenMgr.GetToken(ctx)
}

// ExpireTokenNow forces token expiration for tests.
func (g *GigaChat) ExpireTokenNow() {
	g.tokenMgr.InvalidateCache()
}

// Complete delegates to an OpenAI-compatible completion using the current OAuth token.
func (g *GigaChat) Complete(ctx context.Context, system, user string, p Params) (string, error) {
	token, err := g.tokenMgr.GetToken(ctx)
	if err != nil {
		return "", err
	}

	if p.Model == "" {
		p.Model = g.model
	}

	compat := NewOpenAICompat(g.baseURL, token, g.httpClient)
	return compat.Complete(ctx, system, user, p)
}

// Stream delegates to an OpenAI-compatible stream using the current OAuth token.
func (g *GigaChat) Stream(ctx context.Context, system, user string, p Params, onDelta func(string)) error {
	token, err := g.tokenMgr.GetToken(ctx)
	if err != nil {
		return err
	}

	if p.Model == "" {
		p.Model = g.model
	}

	compat := NewOpenAICompat(g.baseURL, token, g.httpClient)
	return compat.Stream(ctx, system, user, p, onDelta)
}
