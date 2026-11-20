// Package middleware provides HTTP middleware for the Gateway service.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
)

// Auth returns middleware that validates Bearer JWTs.
// On success it stores UserID and Role in the request context.
// On failure it responds with 401 Unauthorized.
func Auth(jwtManager *jwtpkg.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"invalid Authorization header format"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.ParseAccess(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				http.Error(w, `{"error":"invalid user_id in token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithUserID adds a user UUID to a request context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, id)
}

// UserIDFromCtx extracts the authenticated user's UUID from a request context.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(uuid.UUID)
	return v, ok
}

// RoleFromCtx extracts the authenticated user's role from a request context.
func RoleFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyRole).(string)
	return v, ok
}
