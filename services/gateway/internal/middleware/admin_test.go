package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
)

func TestRequireAdmin(t *testing.T) {
	dummyNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.RequireAdmin(dummyNext)

	t.Run("no role in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != `{"error":"forbidden: admin role required"}` {
			t.Errorf("unexpected body: %s", body)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
	})

	t.Run("user role in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		ctx := context.WithValue(req.Context(), middleware.ContextKeyRole, "user")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != `{"error":"forbidden: admin role required"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("admin role in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
		ctx := context.WithValue(req.Context(), middleware.ContextKeyRole, "admin")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != `{"status":"ok"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
