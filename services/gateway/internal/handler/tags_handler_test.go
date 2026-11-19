package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/handler"
)

func TestTagsHandler_Exact23TagsInOrder(t *testing.T) {
	h := handler.NewTagsHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	expectedTags := []string{
		"Go", "Python", "Java", "C#", "C++", "JS", "SQL", "Rust", "TypeScript",
		"Docker", "Linux", "Kubernetes", "PostgreSQL", "Redis", "GraphQL",
		"React", "Vue", "Swift", "Kotlin", "PHP", "Git", "Nginx", "gRPC",
	}

	if len(resp.Items) != len(expectedTags) {
		t.Fatalf("expected exactly %d tags, got %d", len(expectedTags), len(resp.Items))
	}

	for i, expected := range expectedTags {
		if resp.Items[i].Name != expected {
			t.Errorf("tag[%d]: expected %q, got %q", i, expected, resp.Items[i].Name)
		}
	}
}
