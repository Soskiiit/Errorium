package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/handler"
)

func TestTagsHandler_Exact23TagsInOrder(t *testing.T) {
	// No JWT → returns default order.
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

func TestTagsHandler_RankTagsForProfession_AllTagsPresent(t *testing.T) {
	for _, profession := range []string{
		"Backend developer",
		"Frontend developer",
		"Fullstack developer",
		"DevOps engineer",
		"QA engineer",
		"Data engineer / ML",
		"System architect",
		"Mobile developer",
		"Security engineer",
		"Tech lead",
		"Unknown profession",
	} {
		got := handler.RankTagsForProfession(profession)
		if len(got) != len(handler.StandardTags) {
			t.Errorf("profession %q: expected %d tags, got %d", profession, len(handler.StandardTags), len(got))
			continue
		}
		// All standard tags must be present exactly once.
		seen := make(map[string]int)
		for _, tag := range got {
			seen[tag]++
		}
		for _, tag := range handler.StandardTags {
			if seen[tag] != 1 {
				t.Errorf("profession %q: tag %q appears %d times in result", profession, tag, seen[tag])
			}
		}
	}
}

func TestTagsHandler_RankTagsForProfession_BackendFirstTag(t *testing.T) {
	got := handler.RankTagsForProfession("Backend developer")
	if len(got) == 0 {
		t.Fatal("empty result for backend developer")
	}
	if got[0] != "Go" {
		t.Errorf("expected first tag 'Go' for Backend developer, got %q", got[0])
	}
}

func TestTagsHandler_RankTagsForProfession_FrontendFirstTag(t *testing.T) {
	got := handler.RankTagsForProfession("Frontend developer")
	if len(got) == 0 {
		t.Fatal("empty result for frontend developer")
	}
	if got[0] != "JS" {
		t.Errorf("expected first tag 'JS' for Frontend developer, got %q", got[0])
	}
}

