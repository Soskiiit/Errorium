// Package jwt provides a minimal HS256 JWT implementation using only the
// Go standard library (crypto/hmac, crypto/sha256, encoding/base64).
// It intentionally avoids external dependencies so the service can compile
// without network access.
package jwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

// ── Token claims ──────────────────────────────────────────────────────────────

// RegisteredClaims mirrors the standard JWT registered claim names.
type RegisteredClaims struct {
	Subject   string `json:"sub,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	JWTID     string `json:"jti,omitempty"`
}

// Claims is the payload stored in an access token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	RegisteredClaims
}

// ── Manager ───────────────────────────────────────────────────────────────────

// Manager creates and validates HS256 JWTs.
type Manager struct {
	secretKey  []byte
	accessExp  time.Duration
	refreshExp time.Duration
}

// NewManager constructs a Manager from JWTConfig.
func NewManager(cfg config.JWTConfig) *Manager {
	return &Manager{
		secretKey:  []byte(cfg.SecretKey),
		accessExp:  time.Duration(cfg.AccessTokenExpireMin) * time.Minute,
		refreshExp: time.Duration(cfg.RefreshTokenExpireDays) * 24 * time.Hour,
	}
}

// ── Token generation ──────────────────────────────────────────────────────────

// GenerateAccess creates a signed HS256 access token for the given user.
func (m *Manager) GenerateAccess(user *domain.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(m.accessExp).Unix(),
			JWTID:     uuid.New().String(),
		},
	}
	return m.sign(claims)
}

// GenerateRefresh returns a random UUID string to be used as a refresh token.
// The actual secret is stored server-side as a SHA-256 hash.
func (m *Manager) GenerateRefresh() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("jwt: generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ParseAccess validates an HS256 access token and returns its claims.
func (m *Manager) ParseAccess(tokenStr string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: malformed token")
	}

	// Verify signature.
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("jwt: invalid signature encoding")
	}
	expected := m.hmacSHA256(parts[0] + "." + parts[1])
	if !hmac.Equal(sig, expected) {
		return nil, errors.New("jwt: invalid signature")
	}

	// Decode payload.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("jwt: invalid payload encoding")
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("jwt: decode claims: %w", err)
	}

	// Check expiry.
	if claims.ExpiresAt != 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("jwt: token expired")
	}

	return &claims, nil
}

// RefreshTokenHash returns the hex-encoded SHA-256 hash of a refresh token.
// This hash is stored in the database instead of the raw token.
func (m *Manager) RefreshTokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (m *Manager) sign(claims Claims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	h := base64.RawURLEncoding.EncodeToString(headerJSON)
	p := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sig := m.hmacSHA256(h + "." + p)

	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (m *Manager) hmacSHA256(data string) []byte {
	mac := hmac.New(sha256.New, m.secretKey)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
