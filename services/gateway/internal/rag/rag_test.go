package rag_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/llm"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// mockLLMClient implements llm.Client for testing.
type mockLLMClient struct {
	completeFunc func(ctx context.Context, system, user string, p llm.Params) (string, error)
	streamFunc   func(ctx context.Context, system, user string, p llm.Params, onDelta func(string)) error
}

func (m *mockLLMClient) Complete(ctx context.Context, system, user string, p llm.Params) (string, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, system, user, p)
	}
	return "mock complete response", nil
}

func (m *mockLLMClient) Stream(ctx context.Context, system, user string, p llm.Params, onDelta func(string)) error {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, system, user, p, onDelta)
	}
	onDelta("mock ")
	onDelta("stream ")
	onDelta("response")
	return nil
}

// mockRagConfigStore implements repository.RagConfigStore for testing.
type mockRagConfigStore struct {
	cfg *domain.RagConfig
	err error
}

func (m *mockRagConfigStore) Get(ctx context.Context) (*domain.RagConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.cfg != nil {
		return m.cfg, nil
	}
	return &domain.RagConfig{
		Provider:     domain.LLMProviderOpenAI,
		Model:        "gpt-4o-mini",
		SystemPrompt: "Mock system prompt",
		Temperature:  0.2,
		MaxTokens:    1024,
	}, nil
}

func (m *mockRagConfigStore) Update(ctx context.Context, cfg *domain.RagConfig) error {
	m.cfg = cfg
	return nil
}

func TestExtractCodeAndSynthesis(t *testing.T) {
	// 1. Single fenced code block
	raw := `Суть ошибки в синтаксисе вызова print.
Первопричина: скобки не были закрыты.

` + "```python\nprint(\"fixed\")\n```" + `

Для профилактики используйте линтер flake8.`

	synthesis, code := rag.ExtractCodeAndSynthesis(raw)
	if code == nil {
		t.Fatal("expected code block to be extracted, got nil")
	}
	if *code != "print(\"fixed\")" {
		t.Fatalf("expected code 'print(\"fixed\")', got %q", *code)
	}
	if strings.Contains(synthesis, "```") {
		t.Fatalf("synthesis should not contain code fence: %s", synthesis)
	}
	if !strings.Contains(synthesis, "Суть ошибки") || !strings.Contains(synthesis, "Для профилактики") {
		t.Fatalf("synthesis missing surrounding text: %s", synthesis)
	}

	// 2. No code block
	rawNoCode := "Простая ошибка без кода. Исправьте конфигурацию."
	synthesis2, code2 := rag.ExtractCodeAndSynthesis(rawNoCode)
	if code2 != nil {
		t.Fatalf("expected nil code, got %q", *code2)
	}
	if synthesis2 != rawNoCode {
		t.Fatalf("expected synthesis to match raw, got %q", synthesis2)
	}

	// 3. Multiple code blocks: first should be extracted into code, remainder into synthesis
	rawMulti := "Первый блок:\n```go\nfunc Fix() {}\n```\nВторой блок:\n```go\nfunc Other() {}\n```"
	synthesis3, code3 := rag.ExtractCodeAndSynthesis(rawMulti)
	if code3 == nil || *code3 != "func Fix() {}" {
		t.Fatalf("expected first code block, got %v", code3)
	}
	if !strings.Contains(synthesis3, "Второй блок") {
		t.Fatalf("expected remainder in synthesis: %s", synthesis3)
	}
}

func TestOrchestrator_Synthesize_Success(t *testing.T) {
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			if system != "Custom system prompt" {
				t.Errorf("unexpected system prompt: %s", system)
			}
			if !strings.Contains(user, "SyntaxError") {
				t.Errorf("expected user prompt to contain query: %s", user)
			}
			if !strings.Contains(user, "Stack Overflow") {
				t.Errorf("expected user prompt to contain SO context: %s", user)
			}
			return "Причина в незакрытой скобке.\n```python\nprint(1)\n```\nПрофилактика: IDE.", nil
		},
	}

	store := &mockRagConfigStore{
		cfg: &domain.RagConfig{
			Provider:     domain.LLMProviderOpenAI,
			Model:        "gpt-4o-mini",
			SystemPrompt: "Custom system prompt",
			Temperature:  0.2,
			MaxTokens:    1024,
		},
	}

	sem := rag.NewSemaphore(8)
	orch := rag.NewOrchestrator(store, sem, mockClient, mockClient, "sk-test", "https://api.openai.com/v1", "default prompt", nil)

	hits := []search.Source{
		{
			DocID:          27539308,
			Title:          "SyntaxError in Python",
			URL:            "https://stackoverflow.com/questions/27539308",
			Snippet:        "Syntax error details",
			BestAnswerBody: "<p>Best answer with <b>HTML</b></p>",
		},
	}

	res, err := orch.Synthesize(context.Background(), "SyntaxError: invalid syntax", hits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil RAGResult")
	}
	if res.Code == nil || *res.Code != "print(1)" {
		t.Fatalf("expected code 'print(1)', got %v", res.Code)
	}
	if !strings.Contains(res.Synthesis, "Причина в незакрытой скобке") {
		t.Fatalf("unexpected synthesis: %s", res.Synthesis)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(res.Sources))
	}
}

func TestOrchestrator_Synthesize_DegradationOnLLMError(t *testing.T) {
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			return "", llm.ErrUnavailable
		},
	}

	store := &mockRagConfigStore{
		cfg: &domain.RagConfig{
			Provider: domain.LLMProviderOpenAI,
		},
	}

	sem := rag.NewSemaphore(8)
	orch := rag.NewOrchestrator(store, sem, mockClient, mockClient, "sk-test", "https://api.openai.com/v1", "default prompt", nil)

	hits := []search.Source{
		{
			DocID:   27539308,
			Title:   "SyntaxError in Python",
			URL:     "https://stackoverflow.com/questions/27539308",
			Snippet: "Syntax error snippet",
		},
	}

	res, err := orch.Synthesize(context.Background(), "SyntaxError", hits)
	// IMPORTANT: Search must NOT fail! Error should be nil, fallback template returned.
	if err != nil {
		t.Fatalf("expected no error on degradation, got: %v", err)
	}
	if res == nil {
		t.Fatal("expected fallback RAGResult, got nil")
	}
	if !strings.Contains(res.Synthesis, "SyntaxError") {
		t.Fatalf("expected fallback template to contain query/title, got: %s", res.Synthesis)
	}
	if len(res.Sources) != 1 {
		t.Fatalf("expected sources to remain intact on degradation, got %d", len(res.Sources))
	}
}

func TestOrchestrator_Synthesize_StubProvider(t *testing.T) {
	var llmCalled bool
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			llmCalled = true
			return "should not be called", nil
		},
	}

	store := &mockRagConfigStore{
		cfg: &domain.RagConfig{
			Provider: domain.LLMProviderStub,
		},
	}

	sem := rag.NewSemaphore(8)
	orch := rag.NewOrchestrator(store, sem, mockClient, mockClient, "sk-test", "https://api.openai.com/v1", "default prompt", nil)

	hits := []search.Source{
		{
			DocID:   15478423,
			Title:   "SyntaxError print",
			Snippet: "Missing closing parenthesis",
		},
	}

	res, err := orch.Synthesize(context.Background(), "print error", hits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalled {
		t.Fatal("LLM client must not be called when provider is stub")
	}
	if res == nil || res.Synthesis == "" {
		t.Fatal("expected deterministic fallback synthesis")
	}
}

func TestOrchestrator_Semaphore_9thCallDegradesAfter2Seconds(t *testing.T) {
	// Test the exact requirement:
	// "Глобальный семафор LLM: не более LLM_MAX_CONCURRENT (8) одновременных вызовов...
	//  9-й параллельный вызов при лимите 8 ждёт и деградирует после 2 с, слот освобождается при ошибке (тест под -race)"

	sem := rag.NewSemaphore(8)

	// Block 8 slots
	var acquiredCount int32
	var releaseCh = make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquired, release := sem.Acquire(context.Background(), 500*time.Millisecond)
			if !acquired {
				t.Errorf("failed to acquire initial slot")
				return
			}
			atomic.AddInt32(&acquiredCount, 1)
			<-releaseCh
			release()
		}()
	}

	// Wait until all 8 slots are acquired
	for atomic.LoadInt32(&acquiredCount) < 8 {
		time.Sleep(5 * time.Millisecond)
	}

	// Now 9th call using orchestrator with mock LLM
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			return "LLM answer", nil
		},
	}
	store := &mockRagConfigStore{
		cfg: &domain.RagConfig{Provider: domain.LLMProviderOpenAI},
	}
	orch := rag.NewOrchestrator(store, sem, mockClient, mockClient, "key", "https://api.openai.com/v1", "prompt", nil)

	hits := []search.Source{
		{DocID: 100, Title: "Test hit", Snippet: "Test snippet"},
	}

	start := time.Now()
	// Override timeout in acquire test or wait for the 2-second timeout
	res, err := orch.Synthesize(context.Background(), "my query", hits)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("expected degradation without error, got: %v", err)
	}
	if res == nil {
		t.Fatal("expected fallback result")
	}
	// Verify it waited at least ~2 seconds (allowing a slight leeway)
	if duration < 1900*time.Millisecond {
		t.Fatalf("expected to wait ~2s before degrading, took %v", duration)
	}
	// Verify it degraded to template
	if !strings.Contains(res.Synthesis, "Based on Stack Overflow solution") {
		t.Fatalf("expected fallback template synthesis, got: %s", res.Synthesis)
	}

	// Release the 8 held slots
	close(releaseCh)
	wg.Wait()

	// Verify semaphore is completely freed and a new call succeeds
	resSuccess, errSuccess := orch.Synthesize(context.Background(), "my query", hits)
	if errSuccess != nil {
		t.Fatalf("unexpected error after release: %v", errSuccess)
	}
	if resSuccess.Synthesis != "LLM answer" {
		t.Fatalf("expected LLM answer after slots freed, got: %s", resSuccess.Synthesis)
	}
}

func TestOrchestrator_Semaphore_SlotReleasedOnError(t *testing.T) {
	sem := rag.NewSemaphore(1)

	// Acquire and simulate an error / panic in defer
	func() {
		acquired, release := sem.Acquire(context.Background(), 100*time.Millisecond)
		if !acquired {
			t.Fatal("failed to acquire slot")
		}
		defer release()
		// simulate error exit
		_ = errors.New("something went wrong")
	}()

	// Verify slot is immediately available
	acquired, release := sem.Acquire(context.Background(), 100*time.Millisecond)
	if !acquired {
		t.Fatal("slot was not released after error exit")
	}
	release()
}
