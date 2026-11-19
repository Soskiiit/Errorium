package handler

import (
	"net/http"
)

// StandardTags is the curated list of 23 tags matching the frontend tag pills.
var StandardTags = []string{
	"Go",
	"Python",
	"Java",
	"C#",
	"C++",
	"JS",
	"SQL",
	"Rust",
	"TypeScript",
	"Docker",
	"Linux",
	"Kubernetes",
	"PostgreSQL",
	"Redis",
	"GraphQL",
	"React",
	"Vue",
	"Swift",
	"Kotlin",
	"PHP",
	"Git",
	"Nginx",
	"gRPC",
}

// TagsHandler handles GET /api/v1/tags.
type TagsHandler struct{}

// NewTagsHandler constructs a TagsHandler.
func NewTagsHandler() *TagsHandler {
	return &TagsHandler{}
}

// RegisterRoutes wires the tags route on mux.
func (h *TagsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tags", h.ListTags)
}

type tagItem struct {
	Name string `json:"name"`
}

type tagsResponse struct {
	Items []tagItem `json:"items"`
}

// ListTags returns the curated list of 23 standard tags.
func (h *TagsHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	items := make([]tagItem, len(StandardTags))
	for i, name := range StandardTags {
		items[i] = tagItem{Name: name}
	}
	respondJSON(w, http.StatusOK, tagsResponse{Items: items})
}
