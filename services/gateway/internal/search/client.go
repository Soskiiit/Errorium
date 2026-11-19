package search

import "context"

// SearchClient abstracts search and RAG operations.
type SearchClient interface {
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	RAG(ctx context.Context, req SearchRequest) (*RAGResult, error)
}

// SearchRequest represents a search or RAG synthesis query.
type SearchRequest struct {
	Query    string   `json:"query"`
	TopK     int      `json:"top_k"`
	Tags     []string `json:"tags"`
	Language string   `json:"language"`
}

// Source represents a Stack Overflow error thread source.
type Source struct {
	DocID         int64    `json:"doc_id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	Snippet       string   `json:"snippet"`
	Tags          []string `json:"tags"`
	QuestionScore int      `json:"question_score"`
	BM25Score     float64  `json:"bm25_score"`
	IsAccepted    bool     `json:"is_accepted"`
}

// SearchResponse contains ranked search hits.
type SearchResponse struct {
	Query     string   `json:"query"`
	Results   []Source `json:"results"`
	TotalHits int64    `json:"total_hits"`
	TookMs    int64    `json:"took_ms"`
}

// RAGResult contains synthesized solution with code and references.
type RAGResult struct {
	Query     string   `json:"query"`
	Synthesis string   `json:"synthesis"`
	Code      *string  `json:"code"`
	Sources   []Source `json:"sources"`
	TookMs    int64    `json:"took_ms"`
}
