package handler

import (
	"net/http"
)

// StatusHandler handles GET /api/v1/status.
type StatusHandler struct {
	searchEngineAddr string
}

// NewStatusHandler constructs a StatusHandler with search engine address.
func NewStatusHandler(searchEngineAddr string) *StatusHandler {
	return &StatusHandler{
		searchEngineAddr: searchEngineAddr,
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
	resp := statusResponse{
		Gateway: gatewayStatus{
			Status:  "ok",
			Version: "1.0.0",
		},
		SearchCore: searchCoreStatus{
			Connected: false,
			Address:   h.searchEngineAddr,
		},
		RAG: ragStatus{
			Enabled:  false,
			Provider: "stub",
		},
	}
	respondJSON(w, http.StatusOK, resp)
}
