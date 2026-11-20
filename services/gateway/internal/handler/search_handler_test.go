package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

func setupSearchMux() *http.ServeMux {
	searchStub := search.NewStub()
	graphStub := graph.NewStub()
	h := handler.NewSearchHandler(searchStub, graphStub)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestSearchHandler_Search(t *testing.T) {
	mux := setupSearchMux()

	// 1. Valid search query -> 200 OK
	validBody := `{"query": "invalid syntax python", "top_k": 5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(validBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var searchResp search.SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("failed to decode SearchResponse: %v", err)
	}
	if len(searchResp.Results) == 0 {
		t.Error("expected non-empty search results")
	}

	// 2. Empty query -> 400 Bad Request
	emptyBody := `{"query": "   ", "top_k": 5}`
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(emptyBody))
	recEmpty := httptest.NewRecorder()
	mux.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty query, got %d", recEmpty.Code)
	}

	// 3. Query > 8000 runes -> 400 Bad Request
	longQuery := strings.Repeat("a", 8001)
	longBody, _ := json.Marshal(map[string]string{"query": longQuery})
	reqLong := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(longBody))
	recLong := httptest.NewRecorder()
	mux.ServeHTTP(recLong, reqLong)

	if recLong.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for >8000 runes query, got %d", recLong.Code)
	}

	// 4. Malformed JSON -> 400 Bad Request
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString("not-a-json"))
	recBadJSON := httptest.NewRecorder()
	mux.ServeHTTP(recBadJSON, reqBadJSON)

	if recBadJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", recBadJSON.Code)
	}
}

func TestSearchHandler_SearchRAG(t *testing.T) {
	mux := setupSearchMux()

	// 1. Valid RAG query -> 200 OK
	validBody := `{"query": "invalid syntax python"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", bytes.NewBufferString(validBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var ragResp search.RAGResult
	if err := json.Unmarshal(rec.Body.Bytes(), &ragResp); err != nil {
		t.Fatalf("failed to decode RAGResult: %v", err)
	}
	if ragResp.Synthesis == "" {
		t.Error("expected non-empty RAG synthesis")
	}
	if ragResp.Code == nil || *ragResp.Code == "" {
		t.Error("expected non-empty code snippet in RAG response")
	}

	// 2. Empty query -> 400 Bad Request
	emptyBody := `{"query": ""}`
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", bytes.NewBufferString(emptyBody))
	recEmpty := httptest.NewRecorder()
	mux.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty query, got %d", recEmpty.Code)
	}
}

func TestSearchHandler_GetThreadGraph(t *testing.T) {
	mux := setupSearchMux()

	// 1. Valid doc_ids -> 200 OK
	reqValid := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads?doc_ids=27539308,15478423", nil)
	recValid := httptest.NewRecorder()
	mux.ServeHTTP(recValid, reqValid)

	if recValid.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", recValid.Code, recValid.Body.String())
	}

	var tg graph.ThreadGraph
	if err := json.Unmarshal(recValid.Body.Bytes(), &tg); err != nil {
		t.Fatalf("failed to decode ThreadGraph: %v", err)
	}
	if len(tg.Nodes) == 0 {
		t.Error("expected non-empty nodes in graph")
	}
	if len(tg.Edges) == 0 {
		t.Error("expected non-empty edges in graph")
	}

	// 2. Missing/empty doc_ids -> 400 Bad Request
	reqEmpty := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads?doc_ids=", nil)
	recEmpty := httptest.NewRecorder()
	mux.ServeHTTP(recEmpty, reqEmpty)

	if recEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty doc_ids, got %d", recEmpty.Code)
	}

	// 3. More than 20 doc_ids -> 400 Bad Request
	var ids []string
	for i := 0; i < 21; i++ {
		ids = append(ids, "123")
	}
	reqTooMany := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads?doc_ids="+strings.Join(ids, ","), nil)
	recTooMany := httptest.NewRecorder()
	mux.ServeHTTP(recTooMany, reqTooMany)

	if recTooMany.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for >20 doc_ids, got %d", recTooMany.Code)
	}

	// 4. Invalid non-integer doc_id -> 400 Bad Request
	reqInvalid := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads?doc_ids=123,abc", nil)
	recInvalid := httptest.NewRecorder()
	mux.ServeHTTP(recInvalid, reqInvalid)

	if recInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer doc_id, got %d", recInvalid.Code)
	}
}

func TestSearchHandler_GetThreadDetails(t *testing.T) {
	mux := setupSearchMux()

	// 1. Known doc_id -> 200 OK
	reqKnown := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads/27539308", nil)
	recKnown := httptest.NewRecorder()
	mux.ServeHTTP(recKnown, reqKnown)

	if recKnown.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", recKnown.Code, recKnown.Body.String())
	}

	var details graph.ThreadDetails
	if err := json.Unmarshal(recKnown.Body.Bytes(), &details); err != nil {
		t.Fatalf("failed to decode ThreadDetails: %v", err)
	}
	if details.DocID != 27539308 {
		t.Errorf("expected doc_id 27539308, got %d", details.DocID)
	}

	// 2. Unknown doc_id -> 404 Not Found
	reqUnknown := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads/99999999999", nil)
	recUnknown := httptest.NewRecorder()
	mux.ServeHTTP(recUnknown, reqUnknown)

	if recUnknown.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown doc_id, got %d", recUnknown.Code)
	}

	// 3. Bad non-integer doc_id in path -> 400 Bad Request
	reqBadID := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads/invalid-id", nil)
	recBadID := httptest.NewRecorder()
	mux.ServeHTTP(recBadID, reqBadID)

	if recBadID.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer doc_id path, got %d", recBadID.Code)
	}
}

type errSearchClient struct{}

func (e *errSearchClient) Search(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
	return nil, errors.New("backend down")
}

func (e *errSearchClient) RAG(ctx context.Context, req search.SearchRequest) (*search.RAGResult, error) {
	return nil, errors.New("backend down")
}

type errGraphClient struct{}

func (e *errGraphClient) GetThreadGraph(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
	return nil, errors.New("backend down")
}

func (e *errGraphClient) GetThreadDetails(ctx context.Context, docID int64) (*graph.ThreadDetails, error) {
	return nil, graph.ErrNotFound
}

func TestSearchHandler_BackendError_502(t *testing.T) {
	h := handler.NewSearchHandler(&errSearchClient{}, &errGraphClient{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 1. Search returns 502
	reqSearch := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"test"}`))
	recSearch := httptest.NewRecorder()
	mux.ServeHTTP(recSearch, reqSearch)
	if recSearch.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for search backend error, got %d", recSearch.Code)
	}

	// 2. RAG returns 502
	reqRAG := httptest.NewRequest(http.MethodPost, "/api/v1/search/rag", bytes.NewBufferString(`{"query":"test"}`))
	recRAG := httptest.NewRecorder()
	mux.ServeHTTP(recRAG, reqRAG)
	if recRAG.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for rag backend error, got %d", recRAG.Code)
	}

	// 3. Graph returns 502
	reqGraph := httptest.NewRequest(http.MethodGet, "/api/v1/graph/threads?doc_ids=1,2", nil)
	recGraph := httptest.NewRecorder()
	mux.ServeHTTP(recGraph, reqGraph)
	if recGraph.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for graph backend error, got %d", recGraph.Code)
	}
}

type mockLatencyRecorder struct {
	recorded []float64
}

func (m *mockLatencyRecorder) Record(durationMs float64) {
	m.recorded = append(m.recorded, durationMs)
}

func TestSearchHandler_MetricsRecorded(t *testing.T) {
	searchStub := search.NewStub()
	graphStub := graph.NewStub()
	recorder := &mockLatencyRecorder{}

	h := handler.NewSearchHandler(searchStub, graphStub).WithMetrics(recorder)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewBufferString(`{"query":"python error"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if len(recorder.recorded) != 1 {
		t.Fatalf("expected 1 recorded latency, got %d", len(recorder.recorded))
	}
	if recorder.recorded[0] < 0 {
		t.Errorf("expected non-negative duration, got %f", recorder.recorded[0])
	}
}
