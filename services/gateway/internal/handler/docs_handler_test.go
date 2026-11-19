package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/api"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
)

func TestOpenAPIYAML_Embedded(t *testing.T) {
	if len(api.OpenAPIYAML) == 0 {
		t.Fatal("expected api.OpenAPIYAML to be non-empty, but got empty byte slice")
	}

	content := string(api.OpenAPIYAML)
	if !strings.Contains(content, "openapi: 3.0.3") {
		t.Errorf("expected embedded spec to contain 'openapi: 3.0.3', got:\n%s", content[:min(len(content), 200)])
	}
	if !strings.Contains(content, "Errorium Gateway API") {
		t.Errorf("expected embedded spec to contain 'Errorium Gateway API'")
	}
}

func TestDocsHandler_ServeOpenAPI(t *testing.T) {
	dummySpec := []byte("openapi: 3.0.3\ninfo:\n  title: Test API\n  version: 1.0.0\n")
	h := handler.NewDocsHandler(dummySpec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	h.ServeOpenAPI(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	ct := res.Header.Get("Content-Type")
	expectedCT := "application/yaml; charset=utf-8"
	if ct != expectedCT {
		t.Errorf("expected Content-Type %q, got %q", expectedCT, ct)
	}

	body := rec.Body.Bytes()
	if !bytes.Equal(body, dummySpec) {
		t.Errorf("expected body %q, got %q", string(dummySpec), string(body))
	}
}

func TestDocsHandler_ServeOpenAPI_WithRealSpec(t *testing.T) {
	h := handler.NewDocsHandler(api.OpenAPIYAML)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	h.ServeOpenAPI(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	ct := res.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/yaml") {
		t.Errorf("expected Content-Type to contain 'application/yaml', got %q", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "openapi: 3.0.3") {
		t.Errorf("expected body to contain 'openapi: 3.0.3'")
	}
	if !bytes.Equal(rec.Body.Bytes(), api.OpenAPIYAML) {
		t.Errorf("expected body to match api.OpenAPIYAML exactly")
	}
}

func TestDocsHandler_ServeOpenAPI_MethodNotAllowed(t *testing.T) {
	h := handler.NewDocsHandler(api.OpenAPIYAML)

	req := httptest.NewRequest(http.MethodPost, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	h.ServeOpenAPI(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 Method Not Allowed, got %d", res.StatusCode)
	}
}

func TestDocsHandler_ServeSwaggerUI(t *testing.T) {
	h := handler.NewDocsHandler(api.OpenAPIYAML)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	h.ServeSwaggerUI(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}

	ct := res.Header.Get("Content-Type")
	expectedCT := "text/html; charset=utf-8"
	if ct != expectedCT {
		t.Errorf("expected Content-Type %q, got %q", expectedCT, ct)
	}

	html := rec.Body.String()

	requiredSubstrings := []string{
		"<title>Errorium API</title>",
		"https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui.css",
		"https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui-bundle.js",
		"SwaggerUIBundle",
		"/openapi.yaml",
		"dom_id: '#swagger-ui'",
		"deepLinking: true",
		"persistAuthorization: true",
		`<div id="swagger-ui"></div>`,
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(html, sub) {
			t.Errorf("expected Swagger UI HTML to contain %q, but was not found in:\n%s", sub, html)
		}
	}
}

func TestDocsHandler_ServeSwaggerUI_MethodNotAllowed(t *testing.T) {
	h := handler.NewDocsHandler(api.OpenAPIYAML)

	req := httptest.NewRequest(http.MethodPost, "/docs", nil)
	rec := httptest.NewRecorder()

	h.ServeSwaggerUI(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 Method Not Allowed, got %d", res.StatusCode)
	}
}

func TestDocsHandler_RegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	h := handler.NewDocsHandler(api.OpenAPIYAML)
	h.RegisterRoutes(mux)

	// Test GET /openapi.yaml via mux
	reqYAML := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	recYAML := httptest.NewRecorder()
	mux.ServeHTTP(recYAML, reqYAML)

	if recYAML.Code != http.StatusOK {
		t.Errorf("GET /openapi.yaml via mux returned status %d, expected 200", recYAML.Code)
	}
	if recYAML.Header().Get("Content-Type") != "application/yaml; charset=utf-8" {
		t.Errorf("GET /openapi.yaml via mux unexpected Content-Type: %q", recYAML.Header().Get("Content-Type"))
	}
	if !strings.Contains(recYAML.Body.String(), "openapi: 3.0.3") {
		t.Errorf("GET /openapi.yaml via mux expected to contain 'openapi: 3.0.3'")
	}

	// Test GET /docs via mux
	reqDocs := httptest.NewRequest(http.MethodGet, "/docs", nil)
	recDocs := httptest.NewRecorder()
	mux.ServeHTTP(recDocs, reqDocs)

	if recDocs.Code != http.StatusOK {
		t.Errorf("GET /docs via mux returned status %d, expected 200", recDocs.Code)
	}
	if recDocs.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("GET /docs via mux unexpected Content-Type: %q", recDocs.Header().Get("Content-Type"))
	}
	if !strings.Contains(recDocs.Body.String(), "<title>Errorium API</title>") {
		t.Errorf("GET /docs via mux expected to contain title")
	}
}
