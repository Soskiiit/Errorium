package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/internal/app"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

func TestApp_NoDB_Health200_Chats404(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	// 1. GET /health should succeed (200)
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)

	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200, got %d, body: %s", recHealth.Code, recHealth.Body.String())
	}

	var healthResp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recHealth.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("failed to parse /health response JSON: %v", err)
	}
	if healthResp.Status != "ok" {
		t.Fatalf("expected health status 'ok', got %q", healthResp.Status)
	}

	// 2. Unregistered routes (when DB is unconfigured) must return 404
	unregisteredPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/chats"},
		{http.MethodPost, "/api/v1/chats"},
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodPatch, "/api/v1/users/me"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/register"},
	}

	for _, up := range unregisteredPaths {
		req := httptest.NewRequest(up.method, up.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %s %s to return 404 when DB is unconfigured, got %d", up.method, up.path, rec.Code)
		}
	}

	// 3. Public routes must be accessible even when DB is unconfigured
	publicRoutes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/tags", ""},
		{http.MethodGet, "/api/v1/status", ""},
		{http.MethodPost, "/api/v1/search", `{"query":"syntax"}`},
		{http.MethodPost, "/api/v1/search/rag", `{"query":"syntax"}`},
		{http.MethodGet, "/api/v1/graph/threads?doc_ids=27539308", ""},
		{http.MethodGet, "/api/v1/graph/threads/27539308", ""},
	}

	for _, pr := range publicRoutes {
		var req *http.Request
		if pr.body != "" {
			req = httptest.NewRequest(pr.method, pr.path, strings.NewReader(pr.body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(pr.method, pr.path, nil)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected public route %s %s to return 200 without DB, got %d, body: %s", pr.method, pr.path, rec.Code, rec.Body.String())
		}
	}
}

func TestApp_CORS_Headers(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}

	methods := rec.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST, PATCH, DELETE, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods 'GET, POST, PATCH, DELETE, OPTIONS', got %q", methods)
	}

	headers := rec.Header().Get("Access-Control-Allow-Headers")
	if headers != "Authorization, Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers 'Authorization, Content-Type', got %q", headers)
	}
}

func TestApp_CORS_OPTIONS_204(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	handler := application.Handler()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/chats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected OPTIONS to return 204 No Content, got %d", rec.Code)
	}

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}
}

func TestApp_Shutdown(t *testing.T) {
	cfg := &config.Config{
		Port: "8000",
		DB: config.DBConfig{
			Host: "",
		},
	}

	application := app.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		t.Fatalf("expected graceful shutdown without error, got: %v", err)
	}
}
