package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// SearchHandler handles /api/v1/search and /api/v1/graph endpoints.
type SearchHandler struct {
	searchClient search.SearchClient
	graphClient  graph.GraphClient
}

// NewSearchHandler constructs a SearchHandler with search and graph clients.
func NewSearchHandler(searchClient search.SearchClient, graphClient graph.GraphClient) *SearchHandler {
	return &SearchHandler{
		searchClient: searchClient,
		graphClient:  graphClient,
	}
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

	resp, err := h.searchClient.Search(r.Context(), search.SearchRequest{
		Query:    trimmedQuery,
		TopK:     topK,
		Tags:     body.Tags,
		Language: body.Language,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
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

	resp, err := h.searchClient.RAG(r.Context(), search.SearchRequest{
		Query:    trimmedQuery,
		TopK:     topK,
		Tags:     body.Tags,
		Language: body.Language,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
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
		respondError(w, http.StatusInternalServerError, err.Error())
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
