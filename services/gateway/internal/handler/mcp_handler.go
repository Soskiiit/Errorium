package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/internal/mcp"
)

// MCPHandler handles Model Context Protocol (MCP) endpoints: /sse and /mcp/messages.
type MCPHandler struct {
	server       *mcp.Server
	limiter      *mcp.Limiter
	mu           sync.RWMutex
	sessions     map[string]chan []byte
	pingInterval time.Duration
}

// NewMCPHandler constructs an MCPHandler with server and limiter dependencies.
func NewMCPHandler(server *mcp.Server, limiter *mcp.Limiter) *MCPHandler {
	return &MCPHandler{
		server:       server,
		limiter:      limiter,
		sessions:     make(map[string]chan []byte),
		pingInterval: 15 * time.Second,
	}
}

// SetPingInterval overrides the SSE keepalive ping interval (used for tests).
func (h *MCPHandler) SetPingInterval(d time.Duration) {
	if d > 0 {
		h.pingInterval = d
	}
}

// registerSessionForTest registers a session channel for testing without full SSE connection.
func (h *MCPHandler) registerSessionForTest(sessionID string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan []byte, 32)
	h.sessions[sessionID] = ch
	return ch
}

func (h *MCPHandler) removeSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, sessionID)
}

// RegisterRoutes registers /sse and /mcp/messages endpoints.
func (h *MCPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sse", h.ServeSSE)
	mux.HandleFunc("POST /mcp/messages", h.HandleMessages)
}

// ServeSSE handles GET /sse: opens SSE connection, sends endpoint event, and streams JSON-RPC responses.
func (h *MCPHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
	ip := mcp.ExtractIP(r)

	// Check per-IP and global session limits
	allowed, isIPLimit := h.limiter.AllowSession(ip)
	if !allowed {
		w.Header().Set("Retry-After", "5")
		w.Header().Set("Content-Type", "application/json")
		if isIPLimit {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many sessions"})
		return
	}

	sessionID := uuid.New().String()
	if !h.limiter.RegisterSession(sessionID, ip) {
		w.Header().Set("Retry-After", "5")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many sessions"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.limiter.RemoveSession(sessionID)
		respondError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	// Register session event channel
	msgChan := make(chan []byte, 64)
	h.mu.Lock()
	h.sessions[sessionID] = msgChan
	h.mu.Unlock()

	defer func() {
		h.removeSession(sessionID)
		h.limiter.RemoveSession(sessionID)
	}()

	// SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Send initial endpoint event
	if _, err := fmt.Fprintf(w, "event: endpoint\ndata: /mcp/messages?sessionId=%s\n\n", sessionID); err != nil {
		return
	}
	flusher.Flush()

	// Keepalive ticker
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case msg := <-msgChan:
			if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// HandleMessages handles POST /mcp/messages?sessionId=...: receives JSON-RPC 2.0 requests.
func (h *MCPHandler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	h.mu.RLock()
	msgChan, exists := h.sessions[sessionID]
	h.mu.RUnlock()
	if !exists {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	var rpcReq mcp.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json-rpc request")
		return
	}

	// Check sliding window rate limit for tools/call
	if rpcReq.Method == "tools/call" {
		if !h.limiter.AllowCall(sessionID) {
			w.Header().Set("Retry-After", "5")
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
	}

	// notifications/initialized: return 202 Accepted without sending event to SSE
	if rpcReq.Method == "notifications/initialized" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Execute request via MCP Server
	resp := h.server.HandleRequest(r.Context(), rpcReq)
	if resp != nil {
		data, err := json.Marshal(resp)
		if err == nil {
			select {
			case msgChan <- data:
			default:
				// Channel full or consumer lag
			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
