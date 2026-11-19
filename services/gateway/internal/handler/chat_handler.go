package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
)

// ---------------------------------------------------------------------------
// Store interfaces (implemented by @db-layer)
// ---------------------------------------------------------------------------

// ChatStore abstracts persistence for Chat and ChatMessage entities.
type ChatStore interface {
	Create(ctx context.Context, c *domain.Chat) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, search string) ([]*domain.Chat, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error)
	Update(ctx context.Context, c *domain.Chat) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMessage(ctx context.Context, msg *domain.ChatMessage) error
	GetMessages(ctx context.Context, chatID uuid.UUID) ([]*domain.ChatMessage, error)
}

// TagStore abstracts persistence for Tag entities.
type TagStore interface {
	GetOrCreate(ctx context.Context, name string) (*domain.Tag, error)
	AttachToMessage(ctx context.Context, chatMessageID uuid.UUID, tagNames []string) error
	GetByMessage(ctx context.Context, chatMessageID uuid.UUID) ([]*domain.Tag, error)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// ChatHandler handles all /api/v1/chats/* endpoints.
type ChatHandler struct {
	chats ChatStore
	tags  TagStore
}

// NewChatHandler constructs a ChatHandler.
func NewChatHandler(chats ChatStore, tags TagStore) *ChatHandler {
	return &ChatHandler{chats: chats, tags: tags}
}

// RegisterRoutes wires up the chat routes on mux.
// An optional auth middleware may be provided to protect all chat routes.
func (h *ChatHandler) RegisterRoutes(mux *http.ServeMux, authMW ...func(http.Handler) http.Handler) {
	wrap := func(handler http.Handler) http.Handler { return handler }
	if len(authMW) > 0 && authMW[0] != nil {
		wrap = authMW[0]
	}

	mux.Handle("GET /api/v1/chats", wrap(http.HandlerFunc(h.ListChats)))
	mux.Handle("POST /api/v1/chats", wrap(http.HandlerFunc(h.CreateChat)))
	mux.Handle("GET /api/v1/chats/{id}", wrap(http.HandlerFunc(h.GetChat)))
	mux.Handle("PATCH /api/v1/chats/{id}", wrap(http.HandlerFunc(h.UpdateChat)))
	mux.Handle("DELETE /api/v1/chats/{id}", wrap(http.HandlerFunc(h.DeleteChat)))
	mux.Handle("POST /api/v1/chats/{id}/messages", wrap(http.HandlerFunc(h.SendMessage)))
}

// ---------------------------------------------------------------------------
// GET /api/v1/chats
// ---------------------------------------------------------------------------

type listChatsResponse struct {
	Items []*domain.Chat `json:"items"`
	Total int            `json:"total"`
}

func (h *ChatHandler) ListChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	search := r.URL.Query().Get("search")

	chats, total, err := h.chats.ListByUser(r.Context(), userID, limit, offset, search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list chats")
		return
	}

	respondJSON(w, http.StatusOK, listChatsResponse{Items: chats, Total: total})
}

// ---------------------------------------------------------------------------
// POST /api/v1/chats
// ---------------------------------------------------------------------------

type createChatRequest struct {
	Title string `json:"title"`
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	title := req.Title
	if title == "" {
		title = "Новый чат"
	}

	now := time.Now().UTC()
	chat := &domain.Chat{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		IsPinned:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.chats.Create(r.Context(), chat); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create chat")
		return
	}

	respondJSON(w, http.StatusCreated, chat)
}

// ---------------------------------------------------------------------------
// GET /api/v1/chats/{id}
// ---------------------------------------------------------------------------

type getChatResponse struct {
	Chat     *domain.Chat          `json:"chat"`
	Messages []*domain.ChatMessage `json:"messages"`
}

func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	chat, err := h.chats.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	messages, err := h.chats.GetMessages(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get messages")
		return
	}

	respondJSON(w, http.StatusOK, getChatResponse{Chat: chat, Messages: messages})
}

// ---------------------------------------------------------------------------
// PATCH /api/v1/chats/{id}
// ---------------------------------------------------------------------------

type updateChatRequest struct {
	Title    *string `json:"title"`
	IsPinned *bool   `json:"is_pinned"`
}

func (h *ChatHandler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	chat, err := h.chats.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	var req updateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title != nil {
		chat.Title = *req.Title
	}
	if req.IsPinned != nil {
		chat.IsPinned = *req.IsPinned
	}
	chat.UpdatedAt = time.Now().UTC()

	if err := h.chats.Update(r.Context(), chat); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update chat")
		return
	}

	respondJSON(w, http.StatusOK, chat)
}

// ---------------------------------------------------------------------------
// DELETE /api/v1/chats/{id}
// ---------------------------------------------------------------------------

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	if err := h.chats.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete chat")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /api/v1/chats/{id}/messages
// ---------------------------------------------------------------------------

type sendMessageRequest struct {
	Query string   `json:"query"`
	Tags  []string `json:"tags"`
	TopK  int      `json:"top_k"`
}

type sendMessageResponse struct {
	UserMessage      *domain.ChatMessage `json:"user_message"`
	AssistantMessage *domain.ChatMessage `json:"assistant_message"`
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	chatID, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	// Ensure the chat exists.
	if _, err := h.chats.GetByID(r.Context(), chatID); err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		respondError(w, http.StatusBadRequest, "invalid request body: query is required")
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	now := time.Now().UTC()

	// Build error_context for the user message.
	errorCtxJSON, _ := json.Marshal(domain.ErrorContext{
		RawQuery: req.Query,
		Tags:     req.Tags,
	})
	rawErrorCtx := json.RawMessage(errorCtxJSON)

	// Save user message.
	userMsg := &domain.ChatMessage{
		ID:           uuid.New(),
		ChatID:       chatID,
		SenderRole:   domain.SenderRoleUser,
		MessageType:  domain.MessageTypeErrorReport,
		ErrorContext: &rawErrorCtx,
		CreatedAt:    now,
	}
	if err := h.chats.AddMessage(r.Context(), userMsg); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save user message")
		return
	}

	// Attach tags to user message.
	if len(req.Tags) > 0 && h.tags != nil {
		_ = h.tags.AttachToMessage(r.Context(), userMsg.ID, req.Tags)
	}

	// Mock search results (no gRPC yet).
	mockSources := []domain.RAGSource{
		{
			DocID:      1,
			Title:      "Related SO thread",
			Score:      1.5,
			URL:        "https://stackoverflow.com/q/1",
			Snippet:    "Example snippet from Stack Overflow answer.",
			IsAccepted: true,
		},
	}
	ragSourcesJSON, _ := json.Marshal(mockSources)
	rawRAGSources := json.RawMessage(ragSourcesJSON)

	// Save assistant message.
	assistantMsg := &domain.ChatMessage{
		ID:          uuid.New(),
		ChatID:      chatID,
		SenderRole:  domain.SenderRoleAssistant,
		MessageType: domain.MessageTypeRAGSolution,
		RAGSources:  &rawRAGSources,
		CreatedAt:   now,
	}
	if err := h.chats.AddMessage(r.Context(), assistantMsg); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save assistant message")
		return
	}

	respondJSON(w, http.StatusCreated, sendMessageResponse{
		UserMessage:      userMsg,
		AssistantMessage: assistantMsg,
	})
}
