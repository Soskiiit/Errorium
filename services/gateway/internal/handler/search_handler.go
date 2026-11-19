package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// SearchHandler handles /api/v1/search endpoints.
// gRPC integration with Search Engine Core is not yet wired; mock data is returned.
type SearchHandler struct{}

// NewSearchHandler constructs a SearchHandler.
func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

// RegisterRoutes wires the search routes on mux.
func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/search", h.Search)
	mux.HandleFunc("POST /api/v1/search/rag", h.SearchRAG)
}

// ---------------------------------------------------------------------------
// Request / response types
// ---------------------------------------------------------------------------

type searchRequest struct {
	Query string   `json:"query"`
	TopK  int      `json:"top_k"`
	Tags  []string `json:"tags"`
}

type searchResult struct {
	DocID   uint32  `json:"doc_id"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
}

type searchResponse struct {
	Query   string         `json:"query"`
	Results []searchResult `json:"results"`
	TookMs  int64          `json:"took_ms"`
}

type ragResponse struct {
	Query     string         `json:"query"`
	Synthesis string         `json:"synthesis"`
	Sources   []searchResult `json:"sources"`
	TookMs    int64          `json:"took_ms"`
}

// ---------------------------------------------------------------------------
// POST /api/v1/search
// ---------------------------------------------------------------------------

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		respondError(w, http.StatusBadRequest, "invalid request body: query is required")
		return
	}

	start := time.Now()

	// Mock BM25 results — gRPC call to search_engine_core will replace this.
	results := []searchResult{
		{
			DocID:   1,
			Title:   "NullPointerException when calling getString() on a null Bundle",
			Score:   4.2,
			URL:     "https://stackoverflow.com/q/218384",
			Snippet: "You are calling getString() on a Bundle that is null. Check if the bundle is non-null before accessing it.",
		},
		{
			DocID:   2,
			Title:   "How to fix IndexOutOfBoundsException in Java ArrayList",
			Score:   3.7,
			URL:     "https://stackoverflow.com/q/19588080",
			Snippet: "Make sure the index is within [0, size()-1] before accessing the list element.",
		},
	}

	respondJSON(w, http.StatusOK, searchResponse{
		Query:   req.Query,
		Results: results,
		TookMs:  time.Since(start).Milliseconds(),
	})
}

// ---------------------------------------------------------------------------
// POST /api/v1/search/rag
// ---------------------------------------------------------------------------

func (h *SearchHandler) SearchRAG(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		respondError(w, http.StatusBadRequest, "invalid request body: query is required")
		return
	}

	start := time.Now()

	// Mock sources — same as /search for now.
	sources := []searchResult{
		{
			DocID:   1,
			Title:   "NullPointerException when calling getString() on a null Bundle",
			Score:   4.2,
			URL:     "https://stackoverflow.com/q/218384",
			Snippet: "You are calling getString() on a Bundle that is null.",
		},
	}

	synthesis := "Based on BM25 search results, the most likely cause of your error is accessing " +
		"a null reference. Ensure all objects are properly initialised before use. " +
		"See the linked Stack Overflow threads for concrete examples and fixes."

	respondJSON(w, http.StatusOK, ragResponse{
		Query:     req.Query,
		Synthesis: synthesis,
		Sources:   sources,
		TookMs:    time.Since(start).Milliseconds(),
	})
}
