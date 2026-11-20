package llm

import (
	"context"
	"errors"
)

// ErrUnavailable is returned when LLM backend is down, returns HTTP >= 500, or experiences network failure.
var ErrUnavailable = errors.New("llm service unavailable")

// Params specifies parameters for an LLM generation request.
type Params struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

// Client represents a backend LLM provider capable of completion and streaming.
type Client interface {
	Complete(ctx context.Context, system, user string, p Params) (string, error)
	Stream(ctx context.Context, system, user string, p Params, onDelta func(string)) error
}
