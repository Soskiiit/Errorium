package rag_test

import (
	"context"
	"strings"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/llm"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
)

func TestExtractRemember(t *testing.T) {
	raw := "Ответ по сути.\n```remember\nстек: python\nошибка: IndexError\n```\n```python\nprint(1)\n```"
	memory, rest, ok := rag.ExtractRemember(raw)
	if !ok {
		t.Fatal("expected remember block to be found")
	}
	if memory != "стек: python\nошибка: IndexError" {
		t.Fatalf("unexpected memory: %q", memory)
	}
	if strings.Contains(rest, "remember") {
		t.Fatalf("rest must not contain remember block: %s", rest)
	}
	if !strings.Contains(rest, "```python") {
		t.Fatalf("rest must keep code block: %s", rest)
	}
	if _, code := rag.ExtractCodeAndSynthesis(rest); code == nil {
		t.Fatal("code extraction after remember removal failed")
	}

	memory2, rest2, ok2 := rag.ExtractRemember("просто текст")
	if ok2 || memory2 != "" || rest2 != "просто текст" {
		t.Fatalf("no-block case: %q %q %v", memory2, rest2, ok2)
	}
}

func TestOrchestrator_SynthesizeChat_MemoryAndRemember(t *testing.T) {
	var captured string
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			captured = user
			return "Причина в индексе.\n```remember\nфакт: новый\n```\n```python\nprint(1)\n```", nil
		},
	}
	store := &mockRagConfigStore{
		cfg: &domain.RagConfig{
			Provider:     domain.LLMProviderOpenAI,
			Model:        "gpt-4o-mini",
			SystemPrompt: "sys",
			Temperature:  0.2,
			MaxTokens:    1024,
		},
	}
	orch := rag.NewOrchestrator(store, rag.NewSemaphore(8), mockClient, mockClient, "sk-test", "https://api.openai.com/v1", "default", nil)

	cc := rag.ChatContext{
		Memory: "стек: python",
		History: []rag.ChatTurn{
			{Role: "user", Content: "прошлый вопрос"},
			{Role: "assistant", Content: "прошлый ответ"},
		},
	}
	res, err := orch.SynthesizeChat(context.Background(), "а почему так?", nil, cc)
	if err != nil {
		t.Fatalf("SynthesizeChat failed: %v", err)
	}
	for _, want := range []string{"Память чата", "стек: python", "История чата", "[user]: прошлый вопрос", "[assistant]: прошлый ответ"} {
		if !strings.Contains(captured, want) {
			t.Errorf("user prompt missing %q:\n%s", want, captured)
		}
	}
	if res.Remember == nil || *res.Remember != "факт: новый" {
		t.Fatalf("unexpected remember: %v", res.Remember)
	}
	if res.Code == nil || *res.Code != "print(1)" {
		t.Fatalf("unexpected code: %v", res.Code)
	}
	if strings.Contains(res.Synthesis, "```") {
		t.Fatalf("synthesis must be clean of fences: %s", res.Synthesis)
	}
}

func TestOrchestrator_SynthesizeChatStream_Remember(t *testing.T) {
	mockClient := &mockLLMClient{
		streamFunc: func(ctx context.Context, system, user string, p llm.Params, onDelta func(string)) error {
			onDelta("Причина.\n```remember\nфакт: стрим\n```\n")
			onDelta("```python\nprint(1)\n```")
			return nil
		},
	}
	orch := rag.NewOrchestrator(&mockRagConfigStore{}, rag.NewSemaphore(8), mockClient, mockClient, "sk-test", "https://api.openai.com/v1", "default", nil)

	var deltas strings.Builder
	res, err := orch.SynthesizeChatStream(context.Background(), "q", nil, rag.ChatContext{Memory: "m"}, func(d string) {
		deltas.WriteString(d)
	})
	if err != nil {
		t.Fatalf("SynthesizeChatStream failed: %v", err)
	}
	if res.Remember == nil || *res.Remember != "факт: стрим" {
		t.Fatalf("unexpected remember: %v", res.Remember)
	}
	if res.Code == nil || *res.Code != "print(1)" {
		t.Fatalf("unexpected code: %v", res.Code)
	}
	if strings.Contains(res.Synthesis, "```") {
		t.Fatalf("synthesis must be clean of fences: %s", res.Synthesis)
	}
}

func TestOrchestrator_Synthesize_StatelessNoChatSections(t *testing.T) {
	var captured string
	mockClient := &mockLLMClient{
		completeFunc: func(ctx context.Context, system, user string, p llm.Params) (string, error) {
			captured = user
			return "ответ", nil
		},
	}
	orch := rag.NewOrchestrator(&mockRagConfigStore{}, rag.NewSemaphore(8), mockClient, mockClient, "sk", "https://api.openai.com/v1", "default", nil)

	if _, err := orch.Synthesize(context.Background(), "q", nil); err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if strings.Contains(captured, "История чата") || strings.Contains(captured, "Память чата") {
		t.Fatalf("stateless prompt must not contain chat sections:\n%s", captured)
	}
}