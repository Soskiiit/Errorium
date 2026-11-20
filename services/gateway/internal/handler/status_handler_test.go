package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/handler"
)

type mockCoreChecker struct {
	serving bool
	err     error
}

func (m *mockCoreChecker) HealthCheck(ctx context.Context) (bool, error) {
	return m.serving, m.err
}

func TestStatusHandler_ValidResponse(t *testing.T) {
	expectedAddr := "search_engine_core:50051"
	h := handler.NewStatusHandler(expectedAddr)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Gateway struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		} `json:"gateway"`
		SearchCore struct {
			Connected bool   `json:"connected"`
			Address   string `json:"address"`
		} `json:"search_core"`
		RAG struct {
			Enabled  bool   `json:"enabled"`
			Provider string `json:"provider"`
		} `json:"rag"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	if resp.Gateway.Status != "ok" || resp.Gateway.Version != "1.0.0" {
		t.Errorf("unexpected gateway status: %+v", resp.Gateway)
	}
	if resp.SearchCore.Connected != false || resp.SearchCore.Address != expectedAddr {
		t.Errorf("unexpected search_core status: %+v", resp.SearchCore)
	}
	if resp.RAG.Enabled != false || resp.RAG.Provider != "stub" {
		t.Errorf("unexpected rag status: %+v", resp.RAG)
	}
}

func TestStatusHandler_ConnectedServing(t *testing.T) {
	checker := &mockCoreChecker{serving: true, err: nil}
	h := handler.NewStatusHandler("search_engine_core:50051", checker)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp struct {
		SearchCore struct {
			Connected bool   `json:"connected"`
			Address   string `json:"address"`
		} `json:"search_core"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if !resp.SearchCore.Connected {
		t.Errorf("expected search_core.connected == true when SERVING")
	}
}

func TestStatusHandler_NotServingOrError(t *testing.T) {
	// Not serving
	checkerNotServing := &mockCoreChecker{serving: false, err: nil}
	h1 := handler.NewStatusHandler("search_engine_core:50051", checkerNotServing)
	mux1 := http.NewServeMux()
	h1.RegisterRoutes(mux1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec1 := httptest.NewRecorder()
	mux1.ServeHTTP(rec1, req)

	var resp1 struct {
		SearchCore struct {
			Connected bool `json:"connected"`
		} `json:"search_core"`
	}
	_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)
	if resp1.SearchCore.Connected {
		t.Errorf("expected search_core.connected == false when NOT_SERVING")
	}

	// Error
	checkerErr := &mockCoreChecker{serving: false, err: errors.New("timeout")}
	h2 := handler.NewStatusHandler("search_engine_core:50051", checkerErr)
	mux2 := http.NewServeMux()
	h2.RegisterRoutes(mux2)

	rec2 := httptest.NewRecorder()
	mux2.ServeHTTP(rec2, req)

	var resp2 struct {
		SearchCore struct {
			Connected bool `json:"connected"`
		} `json:"search_core"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2.SearchCore.Connected {
		t.Errorf("expected search_core.connected == false on error")
	}

	// Empty address
	h3 := handler.NewStatusHandler("", checkerNotServing)
	mux3 := http.NewServeMux()
	h3.RegisterRoutes(mux3)

	rec3 := httptest.NewRecorder()
	mux3.ServeHTTP(rec3, req)

	var resp3 struct {
		SearchCore struct {
			Connected bool `json:"connected"`
		} `json:"search_core"`
	}
	_ = json.Unmarshal(rec3.Body.Bytes(), &resp3)
	if resp3.SearchCore.Connected {
		t.Errorf("expected search_core.connected == false when address is empty")
	}
}
