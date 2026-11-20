package middleware

import (
	"net/http"
)

// RequireAdmin verifies that the authenticated user has the "admin" role.
// If the role is missing or not "admin", it responds with HTTP 403 Forbidden.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := RoleFromCtx(r.Context())
		if !ok || role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden: admin role required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
