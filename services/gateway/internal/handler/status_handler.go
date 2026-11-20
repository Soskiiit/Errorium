package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
)

// CoreChecker checks whether search_engine_core is available.
type CoreChecker interface {
	HealthCheck(ctx context.Context) (bool, error)
}

// RagConfigGetter retrieves RAG configuration.
type RagConfigGetter interface {
	Get(ctx context.Context) (*domain.RagConfig, error)
}

// StatusHandler handles GET /api/v1/status.
type StatusHandler struct {
	searchEngineAddr string
	coreChecker      CoreChecker
	ragStore         RagConfigGetter
}

// NewStatusHandler constructs a StatusHandler with search engine address and optional health checker.
func NewStatusHandler(searchEngineAddr string, checker ...CoreChecker) *StatusHandler {
	var c CoreChecker
	if len(checker) > 0 {
		c = checker[0]
	}
	return &StatusHandler{
		searchEngineAddr: searchEngineAddr,
		coreChecker:      c,
	}
}

// WithRagConfigStore attaches a RAG config store to the handler.
func (h *StatusHandler) WithRagConfigStore(store RagConfigGetter) *StatusHandler {
	h.ragStore = store
	return h
}

// RegisterRoutes wires the status route on mux.
func (h *StatusHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/status", h.Status)
}

type gatewayStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type searchCoreStatus struct {
	Connected bool   `json:"connected"`
	Address   string `json:"address"`
}

type ragStatus struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

type statusResponse struct {
	Gateway    gatewayStatus    `json:"gateway"`
	SearchCore searchCoreStatus `json:"search_core"`
	RAG        ragStatus        `json:"rag"`
}

// Status returns current services status.
func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	connected := false
	if h.coreChecker != nil && h.searchEngineAddr != "" {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		ok, err := h.coreChecker.HealthCheck(ctx)
		if err == nil && ok {
			connected = true
		}
	}

	ragEnabled := false
	ragProvider := "stub"
	if h.ragStore != nil {
		if cfg, err := h.ragStore.Get(r.Context()); err == nil && cfg != nil {
			if cfg.Provider != "" {
				ragProvider = string(cfg.Provider)
			}
			ragEnabled = (cfg.Provider != domain.LLMProviderStub && cfg.Provider != "")
		}
	}

	resp := statusResponse{
		Gateway: gatewayStatus{
			Status:  "ok",
			Version: "1.0.0",
		},
		SearchCore: searchCoreStatus{
			Connected: connected,
			Address:   h.searchEngineAddr,
		},
		RAG: ragStatus{
			Enabled:  ragEnabled,
			Provider: ragProvider,
		},
	}
	respondJSON(w, http.StatusOK, resp)
}
