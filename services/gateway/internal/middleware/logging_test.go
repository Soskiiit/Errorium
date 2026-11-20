package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
)

func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	}()
	fn()
	return buf.String()
}

func TestRequestLogger(t *testing.T) {
	t.Run("logs method uri status and client", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"uuid"}`))
		})
		handler := middleware.RequestLogger(next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/chats?limit=5", nil)
		req.RemoteAddr = "172.18.0.1:54321"
		rec := httptest.NewRecorder()

		out := captureLog(t, func() { handler.ServeHTTP(rec, req) })

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rec.Code)
		}
		for _, want := range []string{"POST", "/api/v1/chats?limit=5", "201", "172.18.0.1"} {
			if !strings.Contains(out, want) {
				t.Errorf("log line %q must contain %q", out, want)
			}
		}
	})

	t.Run("implicit 200 when WriteHeader not called", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		handler := middleware.RequestLogger(next)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		out := captureLog(t, func() { handler.ServeHTTP(rec, req) })

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(out, "200") {
			t.Errorf("log line %q must contain status 200", out)
		}
	})

	t.Run("prefers X-Real-IP over RemoteAddr", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		handler := middleware.RequestLogger(next)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
		req.RemoteAddr = "172.18.0.5:1111"
		req.Header.Set("X-Real-IP", "203.0.113.7")
		rec := httptest.NewRecorder()

		out := captureLog(t, func() { handler.ServeHTTP(rec, req) })

		if !strings.Contains(out, "203.0.113.7") {
			t.Errorf("log line %q must contain X-Real-IP client address", out)
		}
	})

	t.Run("preserves Flusher for SSE streaming", func(t *testing.T) {
		flushed := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f, ok := w.(http.Flusher)
			if !ok {
				t.Error("wrapped ResponseWriter must implement http.Flusher")
				return
			}
			_, _ = w.Write([]byte("event: delta\n\n"))
			f.Flush()
		})
		handler := middleware.RequestLogger(next)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		flushed = rec.Flushed
		if !flushed {
			t.Error("Flush must reach the underlying ResponseWriter")
		}
		if body := rec.Body.String(); body != "event: delta\n\n" {
			t.Errorf("unexpected streamed body: %q", body)
		}
	})

	t.Run("first WriteHeader wins", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.WriteHeader(http.StatusInternalServerError)
		})
		handler := middleware.RequestLogger(next)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/chats/missing", nil)
		rec := httptest.NewRecorder()

		out := captureLog(t, func() { handler.ServeHTTP(rec, req) })

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
		if !strings.Contains(out, "404") {
			t.Errorf("log line %q must contain first status 404", out)
		}
	})
}
