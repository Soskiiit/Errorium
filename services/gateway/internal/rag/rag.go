package rag

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/llm"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// Semaphore implements a thread-safe token bucket / concurrency limiter for LLM calls.
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore constructs a Semaphore with the given concurrency limit.
func NewSemaphore(limit int) *Semaphore {
	if limit <= 0 {
		limit = 8
	}
	return &Semaphore{ch: make(chan struct{}, limit)}
}

// Acquire attempts to acquire a slot within the specified timeout.
// Returns true and a release function if acquired; false and nil otherwise.
func (s *Semaphore) Acquire(ctx context.Context, timeout time.Duration) (bool, func()) {
	// Fast path
	select {
	case s.ch <- struct{}{}:
		var once sync.Once
		release := func() {
			once.Do(func() {
				<-s.ch
			})
		}
		return true, release
	default:
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case s.ch <- struct{}{}:
		var once sync.Once
		release := func() {
			once.Do(func() {
				<-s.ch
			})
		}
		return true, release
	case <-timer.C:
		return false, nil
	case <-ctx.Done():
		return false, nil
	}
}

var codeBlockRegex = regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_+-]+)?\r?\n?(.*?)\r?\n?```")
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
var multiNewlineRegex = regexp.MustCompile(`\n{3,}`)

// ExtractCodeAndSynthesis separates the first fenced code block from synthesis text.
func ExtractCodeAndSynthesis(raw string) (string, *string) {
	matches := codeBlockRegex.FindStringSubmatchIndex(raw)
	if matches == nil {
		return strings.TrimSpace(raw), nil
	}

	code := strings.TrimSpace(raw[matches[2]:matches[3]])
	before := raw[:matches[0]]
	after := raw[matches[1]:]

	synthesis := strings.TrimSpace(before + "\n" + after)
	synthesis = strings.TrimSpace(multiNewlineRegex.ReplaceAllString(synthesis, "\n\n"))

	var codePtr *string
	if code != "" {
		codePtr = &code
	}
	return synthesis, codePtr
}

func stripHTML(s string) string {
	clean := htmlTagRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(clean)
}

func buildUserPrompt(query string, hits []search.Source) string {
	var sb strings.Builder
	sb.WriteString("Запрос пользователя:\n")
	sb.WriteString(query)
	sb.WriteString("\n\nКонтекст из Stack Overflow:\n")

	if len(hits) == 0 {
		sb.WriteString("Релевантные решения не найдены.\n")
	} else {
		for i, hit := range hits {
			sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, hit.Title))
			if hit.URL != "" {
				sb.WriteString(fmt.Sprintf("URL: %s\n", hit.URL))
			}
			snippet := stripHTML(hit.Snippet)
			if snippet != "" {
				sb.WriteString(fmt.Sprintf("Фрагмент: %s\n", snippet))
			}
			body := stripHTML(hit.BestAnswerBody)
			if body == "" {
				if fix, ok := search.GetFixture(hit.DocID); ok && fix.AnswerBody != "" {
					body = stripHTML(fix.AnswerBody)
				}
			}
			if body != "" {
				sb.WriteString(fmt.Sprintf("Ответ: %s\n", body))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("Инструкция:\n")
	sb.WriteString("Ответь на русском языке. Сначала объясни суть и первопричину проблемы, затем приведи ОДИН fenced блок исправленного кода (в формате ```язык ... ```), затем дай рекомендации по профилактике.")

	return sb.String()
}

// Orchestrator manages RAG synthesis with LLM provider selection and template fallback.
type Orchestrator struct {
	store                repository.RagConfigStore
	semaphore            *Semaphore
	openAIClient         llm.Client
	gigaChatClient       llm.Client
	openAIKey            string
	defaultOpenAIBaseURL string
	defaultSystemPrompt  string
	httpClient           *http.Client
}

// NewOrchestrator constructs a new RAG orchestrator.
func NewOrchestrator(
	store repository.RagConfigStore,
	sem *Semaphore,
	openAI llm.Client,
	gigaChat llm.Client,
	openAIKey string,
	defaultOpenAIBaseURL string,
	defaultSystemPrompt string,
	httpClient *http.Client,
) *Orchestrator {
	if sem == nil {
		sem = NewSemaphore(8)
	}
	if defaultSystemPrompt == "" {
		defaultSystemPrompt = "Ты — Errorium AI, ассистент-эксперт по разработке и отладке ПО. Анализируй контекст из базы знаний Stack Overflow и отвечай чётко, структурированно и только по существу ошибки."
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	return &Orchestrator{
		store:                store,
		semaphore:            sem,
		openAIClient:         openAI,
		gigaChatClient:       gigaChat,
		openAIKey:            openAIKey,
		defaultOpenAIBaseURL: defaultOpenAIBaseURL,
		defaultSystemPrompt:  defaultSystemPrompt,
		httpClient:           httpClient,
	}
}

func (o *Orchestrator) fallbackTemplate(query string, hits []search.Source, tookMs int64) *search.RAGResult {
	synthesis, code := search.SynthesizeTemplate(query, hits)
	return &search.RAGResult{
		Query:     query,
		Synthesis: synthesis,
		Code:      code,
		Sources:   hits,
		TookMs:    tookMs,
	}
}

func (o *Orchestrator) resolveClient(cfg *domain.RagConfig) (llm.Client, llm.Params) {
	params := llm.Params{
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
	}

	switch cfg.Provider {
	case domain.LLMProviderOpenAI:
		baseURL := o.defaultOpenAIBaseURL
		if cfg.BaseURL != nil && *cfg.BaseURL != "" {
			baseURL = *cfg.BaseURL
		}
		if o.openAIClient != nil && (cfg.BaseURL == nil || *cfg.BaseURL == "" || *cfg.BaseURL == o.defaultOpenAIBaseURL) {
			return o.openAIClient, params
		}
		return llm.NewOpenAICompat(baseURL, o.openAIKey, o.httpClient), params

	case domain.LLMProviderGigaChat:
		return o.gigaChatClient, params

	default:
		return nil, params
	}
}

// Synthesize generates a complete RAG synthesis, degrading to template on failure.
func (o *Orchestrator) Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
	start := time.Now()

	var cfg *domain.RagConfig
	if o.store != nil {
		c, err := o.store.Get(ctx)
		if err == nil && c != nil {
			cfg = c
		}
	}

	// If no config or provider is stub -> deterministic template synthesis
	if cfg == nil || cfg.Provider == domain.LLMProviderStub || cfg.Provider == "" {
		tookMs := time.Since(start).Milliseconds()
		return o.fallbackTemplate(query, hits, tookMs), nil
	}

	client, params := o.resolveClient(cfg)
	if client == nil {
		tookMs := time.Since(start).Milliseconds()
		return o.fallbackTemplate(query, hits, tookMs), nil
	}

	// Global LLM semaphore: wait up to 2 seconds
	acquired, release := o.semaphore.Acquire(ctx, 2*time.Second)
	if !acquired {
		log.Printf("RAG: LLM semaphore acquisition timed out (limit reached), degrading to template")
		tookMs := time.Since(start).Milliseconds()
		return o.fallbackTemplate(query, hits, tookMs), nil
	}
	defer release()

	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = o.defaultSystemPrompt
	}
	userPrompt := buildUserPrompt(query, hits)

	rawResp, err := client.Complete(ctx, systemPrompt, userPrompt, params)
	if err != nil {
		log.Printf("RAG: LLM Complete failed (%v), degrading to template", err)
		tookMs := time.Since(start).Milliseconds()
		return o.fallbackTemplate(query, hits, tookMs), nil
	}

	synthesis, code := ExtractCodeAndSynthesis(rawResp)
	tookMs := time.Since(start).Milliseconds()

	return &search.RAGResult{
		Query:     query,
		Synthesis: synthesis,
		Code:      code,
		Sources:   hits,
		TookMs:    tookMs,
	}, nil
}

// SynthesizeStream generates a streaming RAG synthesis, invoking onDelta for text chunks.
func (o *Orchestrator) SynthesizeStream(ctx context.Context, query string, hits []search.Source, onDelta func(string)) (*search.RAGResult, error) {
	start := time.Now()

	var cfg *domain.RagConfig
	if o.store != nil {
		c, err := o.store.Get(ctx)
		if err == nil && c != nil {
			cfg = c
		}
	}

	// Stub mode -> emit template delta and return
	if cfg == nil || cfg.Provider == domain.LLMProviderStub || cfg.Provider == "" {
		tookMs := time.Since(start).Milliseconds()
		res := o.fallbackTemplate(query, hits, tookMs)
		if onDelta != nil {
			onDelta(res.Synthesis)
		}
		return res, nil
	}

	client, params := o.resolveClient(cfg)
	if client == nil {
		tookMs := time.Since(start).Milliseconds()
		res := o.fallbackTemplate(query, hits, tookMs)
		if onDelta != nil {
			onDelta(res.Synthesis)
		}
		return res, nil
	}

	// Semaphore acquisition: wait up to 2 seconds
	acquired, release := o.semaphore.Acquire(ctx, 2*time.Second)
	if !acquired {
		log.Printf("RAG: LLM semaphore acquisition timed out for stream, degrading to template")
		tookMs := time.Since(start).Milliseconds()
		res := o.fallbackTemplate(query, hits, tookMs)
		if onDelta != nil {
			onDelta(res.Synthesis)
		}
		return res, nil
	}
	defer release()

	systemPrompt := cfg.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = o.defaultSystemPrompt
	}
	userPrompt := buildUserPrompt(query, hits)

	var fullText strings.Builder
	streamErr := client.Stream(ctx, systemPrompt, userPrompt, params, func(delta string) {
		fullText.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	})

	if streamErr != nil {
		// If error occurred before any deltas were streamed, degrade to template
		if fullText.Len() == 0 {
			log.Printf("RAG: LLM Stream failed before deltas (%v), degrading to template", streamErr)
			tookMs := time.Since(start).Milliseconds()
			res := o.fallbackTemplate(query, hits, tookMs)
			if onDelta != nil {
				onDelta(res.Synthesis)
			}
			return res, nil
		}
		// Error midway during stream: propagate error
		return nil, streamErr
	}

	synthesis, code := ExtractCodeAndSynthesis(fullText.String())
	tookMs := time.Since(start).Milliseconds()

	return &search.RAGResult{
		Query:     query,
		Synthesis: synthesis,
		Code:      code,
		Sources:   hits,
		TookMs:    tookMs,
	}, nil
}
