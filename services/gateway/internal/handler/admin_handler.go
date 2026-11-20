package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/mcp"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	searchpb "github.com/soskiiit/errorium/services/gateway/internal/pb/search"
)

// CoreAdminClient abstracts index stats and reload operations on search_engine_core.
type CoreAdminClient interface {
	GetStats(ctx context.Context) (*searchpb.GetStatsResponse, error)
	ReloadIndex(ctx context.Context) (*searchpb.ReloadIndexResponse, error)
}

// MetricsProvider provides latency percentiles and total queries count.
type MetricsProvider interface {
	Stats() (p50, p95, p99 float64, total int64)
}

// AdminUserStore abstracts user management operations for the admin API.
type AdminUserStore interface {
	List(ctx context.Context, limit, offset int, search string) ([]*domain.User, int, error)
	UpdateRoleAndStatus(ctx context.Context, id uuid.UUID, role *domain.Role, isActive *bool) (*domain.User, error)
}

// AdminRagConfigStore abstracts RAG/LLM runtime configuration storage.
type AdminRagConfigStore interface {
	Get(ctx context.Context) (*domain.RagConfig, error)
	Update(ctx context.Context, cfg *domain.RagConfig) error
}

// AdminAuditStore abstracts security and admin action logging.
type AdminAuditStore interface {
	Log(ctx context.Context, userID *uuid.UUID, action string, clientIP *netip.Addr, details json.RawMessage) error
}

// AdminHandler handles /api/v1/admin/* operations.
type AdminHandler struct {
	coreClient CoreAdminClient
	metrics    MetricsProvider
	users      AdminUserStore
	ragConfig  AdminRagConfigStore
	audit      AdminAuditStore
}

// NewAdminHandler constructs a new AdminHandler with its dependencies.
func NewAdminHandler(
	coreClient CoreAdminClient,
	metrics MetricsProvider,
	users AdminUserStore,
	ragConfig AdminRagConfigStore,
	audit AdminAuditStore,
) *AdminHandler {
	return &AdminHandler{
		coreClient: coreClient,
		metrics:    metrics,
		users:      users,
		ragConfig:  ragConfig,
		audit:      audit,
	}
}

// RegisterRoutes wires admin routes protected by the supplied admin middleware.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux, adminMW func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/admin/stats", adminMW(http.HandlerFunc(h.Stats)))
	mux.Handle("POST /api/v1/admin/index/reload", adminMW(http.HandlerFunc(h.ReloadIndex)))
	mux.Handle("GET /api/v1/admin/users", adminMW(http.HandlerFunc(h.ListUsers)))
	mux.Handle("PATCH /api/v1/admin/users/{id}", adminMW(http.HandlerFunc(h.PatchUser)))
	mux.Handle("GET /api/v1/admin/rag/config", adminMW(http.HandlerFunc(h.GetRagConfig)))
	mux.Handle("PUT /api/v1/admin/rag/config", adminMW(http.HandlerFunc(h.PutRagConfig)))
}

type adminLatencyStats struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

type adminStatsResponse struct {
	TotalDocuments int64             `json:"total_documents"`
	AvgDocLength   float64           `json:"avg_doc_length"`
	VocabularySize int64             `json:"vocabulary_size"`
	IndexSizeBytes int64             `json:"index_size_bytes"`
	IsReady        bool              `json:"is_ready"`
	LatencyMs      adminLatencyStats `json:"latency_ms"`
	QueriesTotal   int64             `json:"queries_total"`
}

// Stats handles GET /api/v1/admin/stats.
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if h.coreClient == nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	resp, err := h.coreClient.GetStats(r.Context())
	if err != nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	var p50, p95, p99 float64
	var queriesTotal int64
	if h.metrics != nil {
		p50, p95, p99, queriesTotal = h.metrics.Stats()
	}

	respondJSON(w, http.StatusOK, adminStatsResponse{
		TotalDocuments: resp.GetTotalDocuments(),
		AvgDocLength:   resp.GetAvgDocLength(),
		VocabularySize: resp.GetVocabularySize(),
		IndexSizeBytes: resp.GetIndexSizeBytes(),
		IsReady:        resp.GetIsReady(),
		LatencyMs: adminLatencyStats{
			P50: p50,
			P95: p95,
			P99: p99,
		},
		QueriesTotal: queriesTotal,
	})
}

// ReloadIndex handles POST /api/v1/admin/index/reload.
func (h *AdminHandler) ReloadIndex(w http.ResponseWriter, r *http.Request) {
	if h.coreClient == nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	resp, err := h.coreClient.ReloadIndex(r.Context())
	if err != nil || !resp.GetOk() {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	h.recordAudit(r, "admin_index_reload", map[string]interface{}{
		"build_timestamp": resp.GetBuildTimestamp(),
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":              true,
		"build_timestamp": resp.GetBuildTimestamp(),
	})
}

// ListUsers handles GET /api/v1/admin/users.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		respondError(w, http.StatusInternalServerError, "user repository unavailable")
		return
	}

	limit := queryInt(r, "limit", 20)
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}

	offset := queryInt(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	search := r.URL.Query().Get("search")
	items, total, err := h.users.List(r.Context(), limit, offset, search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	if items == nil {
		items = []*domain.User{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": total,
	})
}

type patchUserPayload struct {
	Role     *string `json:"role"`
	IsActive *bool   `json:"is_active"`
}

// PatchUser handles PATCH /api/v1/admin/users/{id}.
func (h *AdminHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		respondError(w, http.StatusInternalServerError, "user repository unavailable")
		return
	}

	targetID, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	adminID, ok := middleware.UserIDFromCtx(r.Context())
	if ok && adminID == targetID {
		respondError(w, http.StatusBadRequest, "cannot modify own admin account")
		return
	}

	var body patchUserPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var rolePtr *domain.Role
	if body.Role != nil {
		rVal := strings.TrimSpace(*body.Role)
		if rVal != string(domain.RoleUser) && rVal != string(domain.RoleAdmin) {
			respondError(w, http.StatusBadRequest, "invalid role: must be 'user' or 'admin'")
			return
		}
		dr := domain.Role(rVal)
		rolePtr = &dr
	}

	updatedUser, err := h.users.UpdateRoleAndStatus(r.Context(), targetID, rolePtr, body.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	changes := make(map[string]interface{})
	if body.Role != nil {
		changes["role"] = *body.Role
	}
	if body.IsActive != nil {
		changes["is_active"] = *body.IsActive
	}

	h.recordAudit(r, "admin_user_patch", map[string]interface{}{
		"target_user_id": targetID.String(),
		"changes":        changes,
	})

	respondJSON(w, http.StatusOK, updatedUser)
}

// GetRagConfig handles GET /api/v1/admin/rag/config.
func (h *AdminHandler) GetRagConfig(w http.ResponseWriter, r *http.Request) {
	if h.ragConfig == nil {
		respondError(w, http.StatusInternalServerError, "rag config repository unavailable")
		return
	}

	cfg, err := h.ragConfig.Get(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get rag config")
		return
	}

	respondJSON(w, http.StatusOK, cfg)
}

type putRagConfigPayload struct {
	Provider     string   `json:"provider"`
	BaseURL      *string  `json:"base_url"`
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
}

// PutRagConfig handles PUT /api/v1/admin/rag/config.
func (h *AdminHandler) PutRagConfig(w http.ResponseWriter, r *http.Request) {
	if h.ragConfig == nil {
		respondError(w, http.StatusInternalServerError, "rag config repository unavailable")
		return
	}

	var body putRagConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 1. Validate provider
	prov := domain.LLMProvider(strings.TrimSpace(body.Provider))
	if prov != domain.LLMProviderOpenAI && prov != domain.LLMProviderGigaChat && prov != domain.LLMProviderStub {
		respondError(w, http.StatusBadRequest, "invalid provider: must be 'openai', 'gigachat' or 'stub'")
		return
	}

	// 2. Validate temperature
	if body.Temperature == nil || *body.Temperature < 0 || *body.Temperature > 2 {
		respondError(w, http.StatusBadRequest, "temperature must be between 0 and 2")
		return
	}

	// 3. Validate max_tokens
	if body.MaxTokens == nil || *body.MaxTokens < 1 {
		respondError(w, http.StatusBadRequest, "max_tokens must be >= 1")
		return
	}

	// 4. Validate model
	trimmedModel := strings.TrimSpace(body.Model)
	if prov != domain.LLMProviderStub && trimmedModel == "" {
		respondError(w, http.StatusBadRequest, "model cannot be empty when provider is not 'stub'")
		return
	}

	// 5. Validate base_url for openai provider
	if prov == domain.LLMProviderOpenAI {
		if body.BaseURL == nil || strings.TrimSpace(*body.BaseURL) == "" {
			respondError(w, http.StatusBadRequest, "base_url is required when provider is 'openai'")
			return
		}
		u, err := url.ParseRequestURI(*body.BaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			respondError(w, http.StatusBadRequest, "base_url must be a valid URL")
			return
		}
	}

	cfg := &domain.RagConfig{
		ID:           1,
		Provider:     prov,
		BaseURL:      body.BaseURL,
		Model:        trimmedModel,
		SystemPrompt: body.SystemPrompt,
		Temperature:  *body.Temperature,
		MaxTokens:    *body.MaxTokens,
	}

	if err := h.ragConfig.Update(r.Context(), cfg); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update rag config")
		return
	}

	h.recordAudit(r, "admin_rag_config_update", map[string]interface{}{
		"provider":      string(cfg.Provider),
		"base_url":      cfg.BaseURL,
		"model":         cfg.Model,
		"temperature":   cfg.Temperature,
		"max_tokens":    cfg.MaxTokens,
		"system_prompt": cfg.SystemPrompt,
	})

	respondJSON(w, http.StatusOK, cfg)
}

func (h *AdminHandler) recordAudit(r *http.Request, action string, details map[string]interface{}) {
	if h.audit == nil {
		return
	}
	var userIDPtr *uuid.UUID
	if id, ok := middleware.UserIDFromCtx(r.Context()); ok {
		userIDPtr = &id
	}

	ipStr := mcp.ExtractIP(r)
	var clientIP *netip.Addr
	if addr, err := netip.ParseAddr(ipStr); err == nil {
		clientIP = &addr
	}

	detailsBytes, _ := json.Marshal(details)
	_ = h.audit.Log(r.Context(), userIDPtr, action, clientIP, detailsBytes)
}
