package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// RAGSynthesizer defines the interface for RAG synthesis orchestration.
type RAGSynthesizer interface {
	Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error)
	SynthesizeStream(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error)
}

// LatencyRecorder records latency measurements in milliseconds.
type LatencyRecorder interface {
	Record(durationMs float64)
}

// SearchHandler handles /api/v1/search and /api/v1/graph endpoints.
type SearchHandler struct {
	searchClient search.SearchClient
	graphClient  graph.GraphClient
	ragOrch      RAGSynthesizer
	metrics      LatencyRecorder
}

// NewSearchHandler constructs a SearchHandler with search and graph clients.
func NewSearchHandler(searchClient search.SearchClient, graphClient graph.GraphClient, ragOrch ...RAGSynthesizer) *SearchHandler {
	var orch RAGSynthesizer
	if len(ragOrch) > 0 {
		orch = ragOrch[0]
	}
	return &SearchHandler{
		searchClient: searchClient,
		graphClient:  graphClient,
		ragOrch:      orch,
	}
}

// WithRAGOrchestrator attaches a RAG synthesizer orchestrator to the handler.
func (h *SearchHandler) WithRAGOrchestrator(orch RAGSynthesizer) *SearchHandler {
	h.ragOrch = orch
	return h
}

// WithMetrics attaches a latency recorder for search calls.
func (h *SearchHandler) WithMetrics(recorder LatencyRecorder) *SearchHandler {
	h.metrics = recorder
	return h
}

// RegisterRoutes wires search and graph routes on mux.
func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/search", h.Search)
	mux.HandleFunc("POST /api/v1/search/rag", h.SearchRAG)
	mux.HandleFunc("GET /api/v1/graph/threads", h.GetThreadGraph)
	mux.HandleFunc("GET /api/v1/graph/threads/{doc_id}", h.GetThreadDetails)
}

type searchRequestPayload struct {
	Query    string   `json:"query"`
	TopK     int      `json:"top_k"`
	Tags     []string `json:"tags"`
	Language string   `json:"language"`
}

// Search handles POST /api/v1/search.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	var body searchRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trimmedQuery := strings.TrimSpace(body.Query)
	runeCount := utf8.RuneCountInString(trimmedQuery)
	if runeCount < 1 || runeCount > 8000 {
		respondError(w, http.StatusBadRequest, "query must be between 1 and 8000 characters")
		return
	}

	topK := body.TopK
	if topK <= 0 {
		topK = 10
	} else if topK > 50 {
		topK = 50
	}

	start := time.Now()
	resp, err := h.searchClient.Search(r.Context(), search.SearchRequest{
		Query:    trimmedQuery,
		TopK:     topK,
		Tags:     body.Tags,
		Language: body.Language,
	})
	if h.metrics != nil {
		h.metrics.Record(float64(time.Since(start).Nanoseconds()) / 1e6)
	}
	if err != nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// SearchRAG handles POST /api/v1/search/rag.
func (h *SearchHandler) SearchRAG(w http.ResponseWriter, r *http.Request) {
	var body searchRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trimmedQuery := strings.TrimSpace(body.Query)
	runeCount := utf8.RuneCountInString(trimmedQuery)
	if runeCount < 1 || runeCount > 8000 {
		respondError(w, http.StatusBadRequest, "query must be between 1 and 8000 characters")
		return
	}

	topK := body.TopK
	if topK <= 0 {
		topK = 10
	} else if topK > 50 {
		topK = 50
	}

	searchReq := search.SearchRequest{
		Query:    trimmedQuery,
		TopK:     topK,
		Tags:     body.Tags,
		Language: body.Language,
	}

	if h.searchClient == nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	// 1. Retrieve search hits from core first (errors before streaming -> 502 JSON)
	start := time.Now()
	searchResp, err := h.searchClient.Search(r.Context(), searchReq)
	if h.metrics != nil {
		h.metrics.Record(float64(time.Since(start).Nanoseconds()) / 1e6)
	}
	if err != nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	// 2. Handle non-SSE vs SSE
	if !isSSE(r) {
		if h.ragOrch != nil {
			ragResp, err := h.ragOrch.Synthesize(r.Context(), trimmedQuery, searchResp.Results)
			if err != nil {
				respondError(w, http.StatusBadGateway, "rag synthesis unavailable")
				return
			}
			respondJSON(w, http.StatusOK, ragResp)
			return
		}

		// Fallback to searchClient.RAG if no orchestrator provided
		resp, err := h.searchClient.RAG(r.Context(), searchReq)
		if err != nil {
			respondError(w, http.StatusBadGateway, "search backend unavailable")
			return
		}
		respondJSON(w, http.StatusOK, resp)
		return
	}

	// SSE streaming mode
	flusher := initSSE(w, http.StatusOK)
	if h.ragOrch != nil {
		ragResp, err := h.ragOrch.SynthesizeStream(r.Context(), trimmedQuery, searchResp.Results, func(delta string) {
			_ = writeSSE(w, flusher, "delta", map[string]string{"text": delta})
		})
		if err != nil {
			_ = writeSSE(w, flusher, "error", map[string]string{"error": err.Error()})
			return
		}
		_ = writeSSE(w, flusher, "done", ragResp)
		return
	}

	// Fallback streaming if no orchestrator configured
	resp, err := h.searchClient.RAG(r.Context(), searchReq)
	if err != nil {
		_ = writeSSE(w, flusher, "error", map[string]string{"error": err.Error()})
		return
	}
	_ = writeSSE(w, flusher, "delta", map[string]string{"text": resp.Synthesis})
	_ = writeSSE(w, flusher, "done", resp)
}

// GetThreadGraph handles GET /api/v1/graph/threads?doc_ids=1,2.
func (h *SearchHandler) GetThreadGraph(w http.ResponseWriter, r *http.Request) {
	rawDocIDs := strings.TrimSpace(r.URL.Query().Get("doc_ids"))
	if rawDocIDs == "" {
		respondError(w, http.StatusBadRequest, "doc_ids query parameter is required")
		return
	}

	parts := strings.Split(rawDocIDs, ",")
	if len(parts) < 1 || len(parts) > 20 {
		respondError(w, http.StatusBadRequest, "doc_ids must contain between 1 and 20 ids")
		return
	}

	docIDs := make([]int64, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			respondError(w, http.StatusBadRequest, "invalid doc_id in doc_ids parameter")
			return
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid doc_id in doc_ids parameter")
			return
		}
		docIDs = append(docIDs, id)
	}

	resp, err := h.graphClient.GetThreadGraph(r.Context(), docIDs)
	if err != nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetThreadDetails handles GET /api/v1/graph/threads/{doc_id}.
func (h *SearchHandler) GetThreadDetails(w http.ResponseWriter, r *http.Request) {
	rawDocID := r.PathValue("doc_id")
	docID, err := strconv.ParseInt(rawDocID, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid doc_id")
		return
	}

	resp, err := h.graphClient.GetThreadDetails(r.Context(), docID)
	if err != nil {
		respondError(w, http.StatusNotFound, "thread not found")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}
