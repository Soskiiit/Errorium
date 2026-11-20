package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// ---------------------------------------------------------------------------
// Store interfaces (implemented by @db-layer)
// ---------------------------------------------------------------------------

// ChatStore abstracts persistence for Chat and ChatMessage entities.
type ChatStore interface {
	Create(ctx context.Context, c *domain.Chat) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, search string) ([]*domain.Chat, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error)
	GetByIDForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Chat, error)
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
	chats   ChatStore
	tags    TagStore
	search  search.SearchClient
	ragOrch RAGSynthesizer
}

// NewChatHandler constructs a ChatHandler.
func NewChatHandler(chats ChatStore, tags TagStore, searchClient search.SearchClient, ragOrch ...RAGSynthesizer) *ChatHandler {
	var orch RAGSynthesizer
	if len(ragOrch) > 0 {
		orch = ragOrch[0]
	}
	return &ChatHandler{
		chats:   chats,
		tags:    tags,
		search:  searchClient,
		ragOrch: orch,
	}
}

// WithRAGOrchestrator attaches a RAG synthesizer orchestrator to the handler.
func (h *ChatHandler) WithRAGOrchestrator(orch RAGSynthesizer) *ChatHandler {
	h.ragOrch = orch
	return h
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

	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	chats, total, err := h.chats.ListByUser(r.Context(), userID, limit, offset, search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list chats")
		return
	}
	if chats == nil {
		chats = []*domain.Chat{}
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

	title := strings.TrimSpace(req.Title)
	if len([]rune(title)) > 255 {
		respondError(w, http.StatusBadRequest, "title cannot exceed 255 characters")
		return
	}
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
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	chat, err := h.chats.GetByIDForUser(r.Context(), id, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	messages, err := h.chats.GetMessages(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get messages")
		return
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})

	if h.tags != nil {
		for _, msg := range messages {
			if msg.SenderRole == domain.SenderRoleUser {
				tags, err := h.tags.GetByMessage(r.Context(), msg.ID)
				if err == nil && len(tags) > 0 {
					msg.Tags = make([]domain.Tag, len(tags))
					for i, t := range tags {
						msg.Tags[i] = *t
					}
				}
			}
		}
	}

	if messages == nil {
		messages = []*domain.ChatMessage{}
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
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	chat, err := h.chats.GetByIDForUser(r.Context(), id, userID)
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
		t := strings.TrimSpace(*req.Title)
		if len([]rune(t)) < 1 || len([]rune(t)) > 255 {
			respondError(w, http.StatusBadRequest, "title must be between 1 and 255 characters")
			return
		}
		chat.Title = t
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
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	if _, err := h.chats.GetByIDForUser(r.Context(), id, userID); err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	if err := h.chats.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete chat")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// chatContext builds the per-chat RAG context: memory cell plus the last 5
// messages, each truncated to 600 runes (ADR-014).
func (h *ChatHandler) chatContext(ctx context.Context, chat *domain.Chat) rag.ChatContext {
	cc := rag.ChatContext{Memory: chat.Memory}
	msgs, err := h.chats.GetMessages(ctx, chat.ID)
	if err != nil {
		return cc
	}
	if len(msgs) > 5 {
		msgs = msgs[len(msgs)-5:]
	}
	for _, m := range msgs {
		if m.Content == nil {
			continue
		}
		content := *m.Content
		if runes := []rune(content); len(runes) > 600 {
			content = string(runes[:600])
		}
		cc.History = append(cc.History, rag.ChatTurn{Role: string(m.SenderRole), Content: content})
	}
	return cc
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
	userID, ok := middleware.UserIDFromCtx(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID, err := parseUUIDParam(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	// Ensure the chat exists and belongs to the user.
	chat, err := h.chats.GetByIDForUser(r.Context(), chatID, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "chat not found")
		return
	}

	// Snapshot history/memory before the current user message is saved.
	cc := h.chatContext(r.Context(), chat)

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rawQuery := req.Query
	tagRegex := regexp.MustCompile(`\[([^\[\]]{1,50})\]`)
	matches := tagRegex.FindAllStringSubmatch(rawQuery, -1)

	var mergedTags []string
	seen := make(map[string]bool)
	addTag := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		r := []rune(t)
		if len(r) > 50 {
			t = string(r[:50])
		}
		if t != "" && !seen[t] {
			seen[t] = true
			mergedTags = append(mergedTags, t)
		}
	}
	for _, m := range matches {
		addTag(m[1])
	}
	for _, t := range req.Tags {
		addTag(t)
	}

	clean := tagRegex.ReplaceAllString(rawQuery, " ")
	wsRegex := regexp.MustCompile(`\s+`)
	cleanQuery := strings.TrimSpace(wsRegex.ReplaceAllString(clean, " "))
	if cleanQuery == "" {
		respondError(w, http.StatusBadRequest, "clean query is empty")
		return
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 5
	} else if topK > 20 {
		topK = 20
	}

	now := time.Now().UTC()

	// Build error_context for the user message.
	errorCtxJSON, _ := json.Marshal(domain.ErrorContext{
		RawQuery: rawQuery,
		Tags:     mergedTags,
	})
	rawErrorCtx := json.RawMessage(errorCtxJSON)

	// Save user message.
	userMsg := &domain.ChatMessage{
		ID:           uuid.New(),
		ChatID:       chatID,
		SenderRole:   domain.SenderRoleUser,
		MessageType:  domain.MessageTypeErrorReport,
		Content:      &rawQuery,
		ErrorContext: &rawErrorCtx,
		CreatedAt:    now,
	}
	if err := h.chats.AddMessage(r.Context(), userMsg); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save user message")
		return
	}

	// Attach tags to user message.
	if len(mergedTags) > 0 && h.tags != nil {
		_ = h.tags.AttachToMessage(r.Context(), userMsg.ID, mergedTags)
	}

	// Populate tags on userMsg.
	if h.tags != nil {
		storedTags, err := h.tags.GetByMessage(r.Context(), userMsg.ID)
		if err == nil && len(storedTags) > 0 {
			userMsg.Tags = make([]domain.Tag, len(storedTags))
			for i, t := range storedTags {
				userMsg.Tags[i] = *t
			}
		}
	}
	if len(userMsg.Tags) == 0 && len(mergedTags) > 0 {
		for _, t := range mergedTags {
			userMsg.Tags = append(userMsg.Tags, domain.Tag{Name: t})
		}
	}

	// Call search.
	if h.search == nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}
	searchReq := search.SearchRequest{
		Query: cleanQuery,
		TopK:  topK,
		Tags:  mergedTags,
	}
	searchResp, err := h.search.Search(r.Context(), searchReq)
	if err != nil {
		respondError(w, http.StatusBadGateway, "search backend unavailable")
		return
	}

	// Handle non-SSE vs SSE
	if !isSSE(r) {
		var ragResult *search.RAGResult
		if h.ragOrch != nil {
			var res *search.RAGResult
			var err error
			if chatOrch, ok := h.ragOrch.(ChatRAGSynthesizer); ok {
				res, err = chatOrch.SynthesizeChat(r.Context(), cleanQuery, searchResp.Results, cc)
			} else {
				res, err = h.ragOrch.Synthesize(r.Context(), cleanQuery, searchResp.Results)
			}
			if err != nil {
				respondError(w, http.StatusBadGateway, "search backend unavailable")
				return
			}
			ragResult = res
		} else {
			res, err := h.search.RAG(r.Context(), searchReq)
			if err != nil {
				respondError(w, http.StatusBadGateway, "search backend unavailable")
				return
			}
			ragResult = res
		}

		var rawRAGSources *json.RawMessage
		if ragResult.Sources != nil {
			sourcesJSON, err := json.Marshal(ragResult.Sources)
			if err == nil {
				rawSources := json.RawMessage(sourcesJSON)
				rawRAGSources = &rawSources
			}
		}

		assistantNow := time.Now().UTC()
		assistantMsg := &domain.ChatMessage{
			ID:          uuid.New(),
			ChatID:      chatID,
			SenderRole:  domain.SenderRoleAssistant,
			MessageType: domain.MessageTypeRAGSolution,
			Content:     &ragResult.Synthesis,
			Code:        ragResult.Code,
			RAGSources:  rawRAGSources,
			CreatedAt:   assistantNow,
		}
		if err := h.chats.AddMessage(r.Context(), assistantMsg); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to save assistant message")
			return
		}

		if ragResult.Remember != nil && strings.TrimSpace(*ragResult.Remember) != "" {
			chat.Memory = strings.TrimSpace(*ragResult.Remember)
		}

		// Auto-titling: if chat.Title == "Новый чат", set title to first 80 runes of cleanQuery.
		if chat.Title == "Новый чат" {
			runes := []rune(cleanQuery)
			if len(runes) > 80 {
				runes = runes[:80]
			}
			chat.Title = string(runes)
		}
		chat.UpdatedAt = time.Now().UTC()
		_ = h.chats.Update(r.Context(), chat)

		respondJSON(w, http.StatusCreated, sendMessageResponse{
			UserMessage:      userMsg,
			AssistantMessage: assistantMsg,
		})
		return
	}

	// SSE streaming mode
	flusher := initSSE(w, http.StatusCreated)

	// 1. First event: user_message
	_ = writeSSE(w, flusher, "user_message", map[string]any{"user_message": userMsg})

	// 2. Stream deltas
	var accumulated strings.Builder
	var streamErr error
	var finalRagResult *search.RAGResult

	if h.ragOrch != nil {
		onDelta := func(delta string) {
			accumulated.WriteString(delta)
			_ = writeSSE(w, flusher, "delta", map[string]string{"text": delta})
		}
		if chatOrch, ok := h.ragOrch.(ChatRAGSynthesizer); ok {
			finalRagResult, streamErr = chatOrch.SynthesizeChatStream(r.Context(), cleanQuery, searchResp.Results, cc, onDelta)
		} else {
			finalRagResult, streamErr = h.ragOrch.SynthesizeStream(r.Context(), cleanQuery, searchResp.Results, onDelta)
		}
	} else {
		res, err := h.search.RAG(r.Context(), searchReq)
		if err != nil {
			streamErr = err
		} else {
			finalRagResult = res
			accumulated.WriteString(res.Synthesis)
			_ = writeSSE(w, flusher, "delta", map[string]string{"text": res.Synthesis})
		}
	}

	// 3. Handle client disconnect / cancellation
	if r.Context().Err() != nil {
		if accumulated.Len() > 0 {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			partialContent := accumulated.String()
			partialMsg := &domain.ChatMessage{
				ID:          uuid.New(),
				ChatID:      chatID,
				SenderRole:  domain.SenderRoleAssistant,
				MessageType: domain.MessageTypeRAGSolution,
				Content:     &partialContent,
				CreatedAt:   time.Now().UTC(),
			}
			_ = h.chats.AddMessage(saveCtx, partialMsg)
		}
		return
	}

	if streamErr != nil {
		_ = writeSSE(w, flusher, "error", map[string]string{"error": streamErr.Error()})
		return
	}

	// 4. Save completed assistant message to DB
	var rawRAGSources *json.RawMessage
	if finalRagResult != nil && finalRagResult.Sources != nil {
		sourcesJSON, err := json.Marshal(finalRagResult.Sources)
		if err == nil {
			rawSources := json.RawMessage(sourcesJSON)
			rawRAGSources = &rawSources
		}
	}

	assistantNow := time.Now().UTC()
	assistantMsg := &domain.ChatMessage{
		ID:          uuid.New(),
		ChatID:      chatID,
		SenderRole:  domain.SenderRoleAssistant,
		MessageType: domain.MessageTypeRAGSolution,
		Content:     &finalRagResult.Synthesis,
		Code:        finalRagResult.Code,
		RAGSources:  rawRAGSources,
		CreatedAt:   assistantNow,
	}
	if err := h.chats.AddMessage(r.Context(), assistantMsg); err != nil {
		_ = writeSSE(w, flusher, "error", map[string]string{"error": "failed to save assistant message"})
		return
	}

	if finalRagResult != nil && finalRagResult.Remember != nil && strings.TrimSpace(*finalRagResult.Remember) != "" {
		chat.Memory = strings.TrimSpace(*finalRagResult.Remember)
	}

	if chat.Title == "Новый чат" {
		runes := []rune(cleanQuery)
		if len(runes) > 80 {
			runes = runes[:80]
		}
		chat.Title = string(runes)
	}
	chat.UpdatedAt = time.Now().UTC()
	_ = h.chats.Update(r.Context(), chat)

	// 5. Final event: done
	_ = writeSSE(w, flusher, "done", sendMessageResponse{
		UserMessage:      userMsg,
		AssistantMessage: assistantMsg,
	})
}
