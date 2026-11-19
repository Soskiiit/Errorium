package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/handler"
)

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
