package handler

import (
	"context"
	"net/http"
	"time"
)

// CoreChecker checks whether search_engine_core is available.
type CoreChecker interface {
	HealthCheck(ctx context.Context) (bool, error)
}

// StatusHandler handles GET /api/v1/status.
type StatusHandler struct {
	searchEngineAddr string
	coreChecker      CoreChecker
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
			Enabled:  false,
			Provider: "stub",
		},
	}
	respondJSON(w, http.StatusOK, resp)
}
