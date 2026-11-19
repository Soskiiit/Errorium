package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChatMessage_JSONSerialization_NilPointers(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	msg := &ChatMessage{
		ID:          uuid.MustParse("01934272-3564-77e4-b7e3-77dd7fc7c13b"),
		ChatID:      uuid.MustParse("01934272-3564-77e4-b7e3-77dd7fc7c13c"),
		SenderRole:  SenderRoleUser,
		MessageType: MessageTypeText,
		Content:     nil,
		Code:        nil,
		CreatedAt:   now,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal ChatMessage: %v", err)
	}

	jsonStr := string(data)

	// Verify "content":null and "code":null are present in serialized JSON
	if !strings.Contains(jsonStr, `"content":null`) {
		t.Errorf("expected ChatMessage JSON to contain '\"content\":null', got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"code":null`) {
		t.Errorf("expected ChatMessage JSON to contain '\"code\":null', got: %s", jsonStr)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ChatMessage JSON: %v", err)
	}

	contentVal, ok := parsed["content"]
	if !ok || contentVal != nil {
		t.Errorf("expected parsed content to be nil, got: %v (present: %v)", contentVal, ok)
	}

	codeVal, ok := parsed["code"]
	if !ok || codeVal != nil {
		t.Errorf("expected parsed code to be nil, got: %v (present: %v)", codeVal, ok)
	}
}

func TestChatMessage_JSONSerialization_PopulatedPointers(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	content := "Traceback (most recent call last):\nIndexError: list index out of range"
	code := "items = []\nprint(items[0])"

	msg := &ChatMessage{
		ID:          uuid.MustParse("01934272-3564-77e4-b7e3-77dd7fc7c13b"),
		ChatID:      uuid.MustParse("01934272-3564-77e4-b7e3-77dd7fc7c13c"),
		SenderRole:  SenderRoleUser,
		MessageType: MessageTypeErrorReport,
		Content:     &content,
		Code:        &code,
		CreatedAt:   now,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal ChatMessage: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal ChatMessage JSON: %v", err)
	}

	if val, ok := parsed["content"]; !ok || val != content {
		t.Errorf("expected content %q, got %v", content, val)
	}

	if val, ok := parsed["code"]; !ok || val != code {
		t.Errorf("expected code %q, got %v", code, val)
	}
}
