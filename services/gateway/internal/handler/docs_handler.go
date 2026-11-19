package handler

import (
	"net/http"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Errorium API</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui-bundle.js" charset="UTF-8"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/openapi.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                persistAuthorization: true
            });
        };
    </script>
</body>
</html>`

// DocsHandler serves raw OpenAPI specification and Swagger UI interactive documentation.
type DocsHandler struct {
	spec []byte
}

// NewDocsHandler creates a new DocsHandler with the given raw OpenAPI YAML specification.
func NewDocsHandler(spec []byte) *DocsHandler {
	return &DocsHandler{
		spec: spec,
	}
}

// RegisterRoutes registers OpenAPI specification and Swagger UI routes on the provided mux.
func (h *DocsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", h.ServeOpenAPI)
	mux.HandleFunc("GET /docs", h.ServeSwaggerUI)
}

// ServeOpenAPI serves the raw OpenAPI YAML document.
func (h *DocsHandler) ServeOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.spec)
}

// ServeSwaggerUI serves the standalone Swagger UI HTML document.
func (h *DocsHandler) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerUIHTML))
}
