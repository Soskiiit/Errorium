package handler

import (
	"net/http"

	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
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

// professionTagWeights maps profession → list of preferred tags (highest priority first).
// Tags not listed for a profession are appended at the end in default order.
var professionTagWeights = map[string][]string{
	"Backend developer": {
		"Go", "Python", "Java", "SQL", "PostgreSQL", "Redis", "Docker", "gRPC", "Rust", "C#", "Linux", "Nginx", "Kubernetes", "Git",
	},
	"Frontend developer": {
		"JS", "TypeScript", "React", "Vue", "CSS", "HTML", "Git", "Docker", "GraphQL", "Python",
	},
	"Fullstack developer": {
		"JS", "TypeScript", "React", "Vue", "Python", "Go", "SQL", "PostgreSQL", "Docker", "Git", "Redis",
	},
	"DevOps engineer": {
		"Docker", "Kubernetes", "Linux", "Nginx", "Git", "Python", "Go", "PostgreSQL", "Redis", "Rust",
	},
	"QA engineer": {
		"Python", "JS", "SQL", "Git", "Docker", "Java", "PostgreSQL",
	},
	"Data engineer / ML": {
		"Python", "SQL", "PostgreSQL", "Redis", "Docker", "Kubernetes", "Rust", "Go", "Git",
	},
	"System architect": {
		"Go", "Rust", "Java", "C++", "gRPC", "Docker", "Kubernetes", "PostgreSQL", "Redis", "Linux", "Git",
	},
	"Mobile developer": {
		"Swift", "Kotlin", "Java", "C#", "JS", "TypeScript", "React", "Git", "Docker",
	},
	"Security engineer": {
		"Linux", "Python", "Rust", "Go", "Docker", "Git", "C++",
	},
	"Tech lead": {
		"Go", "Python", "Java", "Docker", "Kubernetes", "PostgreSQL", "Redis", "Git", "Linux",
	},
}

// RankTagsForProfession returns StandardTags reordered so that tags preferred
// for the given profession come first (in preference order), followed by
// remaining tags in default order. It is exported for use in tests.
func RankTagsForProfession(profession string) []string {
	return rankTagsForProfession(profession)
}

// rankTagsForProfession returns StandardTags reordered so that tags preferred
// for the given profession come first (in preference order), followed by
// remaining tags in default order.
func rankTagsForProfession(profession string) []string {
	preferred, ok := professionTagWeights[profession]
	if !ok {
		return StandardTags
	}

	// Build an ordered set from preferred list (keep only tags that exist in StandardTags).
	standardSet := make(map[string]bool, len(StandardTags))
	for _, t := range StandardTags {
		standardSet[t] = true
	}

	seen := make(map[string]bool, len(StandardTags))
	result := make([]string, 0, len(StandardTags))

	for _, tag := range preferred {
		if standardSet[tag] && !seen[tag] {
			result = append(result, tag)
			seen[tag] = true
		}
	}
	// Append remaining tags in default order.
	for _, tag := range StandardTags {
		if !seen[tag] {
			result = append(result, tag)
		}
	}
	return result
}

// TagsHandler handles GET /api/v1/tags.
type TagsHandler struct {
	jwt *jwtpkg.Manager
}

// NewTagsHandler constructs a TagsHandler.
// jwt is optional — when nil, the endpoint ignores auth and returns default tag order.
func NewTagsHandler(jwtMgr ...*jwtpkg.Manager) *TagsHandler {
	h := &TagsHandler{}
	if len(jwtMgr) > 0 && jwtMgr[0] != nil {
		h.jwt = jwtMgr[0]
	}
	return h
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

// ListTags returns the curated list of 23 standard tags, ordered by profession
// relevance when the caller is authenticated (Bearer token in Authorization header).
func (h *TagsHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags := StandardTags

	// Try to personalize order if JWT manager is available.
	if h.jwt != nil {
		if authHeader := r.Header.Get("Authorization"); len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			if claims, err := h.jwt.ParseAccess(token); err == nil && claims != nil {
				if profession := claims.Profession; profession != "" {
					tags = rankTagsForProfession(profession)
				}
			}
		}
	}

	items := make([]tagItem, len(tags))
	for i, name := range tags {
		items[i] = tagItem{Name: name}
	}
	respondJSON(w, http.StatusOK, tagsResponse{Items: items})
}
