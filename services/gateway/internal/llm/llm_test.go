package llm_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/soskiiit/errorium/services/gateway/internal/llm"
)

func TestOpenAICompat_Complete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", auth)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode req: %v", err)
		}
		if req["model"] != "gpt-4o-mini" {
			t.Errorf("expected model gpt-4o-mini, got %v", req["model"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Fixed code and explanation",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := llm.NewOpenAICompat(ts.URL, "test-key", ts.Client())
	res, err := client.Complete(context.Background(), "You are helpful", "Fix my bug", llm.Params{
		Model:       "gpt-4o-mini",
		Temperature: 0.2,
		MaxTokens:   500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "Fixed code and explanation" {
		t.Fatalf("unexpected response: %s", res)
	}
}

func TestOpenAICompat_Stream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("flusher required")
		}

		chunks := []string{"Hello", " ", "world", "!"}
		for _, c := range chunks {
			chunkJSON := fmt.Sprintf(`{"choices":[{"delta":{"content":%q}}]}`, c)
			fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer ts.Close()

	client := llm.NewOpenAICompat(ts.URL, "test-key", ts.Client())
	var deltas []string
	err := client.Stream(context.Background(), "sys", "user", llm.Params{Model: "gpt-4o-mini"}, func(d string) {
		deltas = append(deltas, d)
	})
	if err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	combined := strings.Join(deltas, "")
	if combined != "Hello world!" {
		t.Fatalf("expected 'Hello world!', got %q", combined)
	}
}

func TestOpenAICompat_ServerErrorReturnsErrUnavailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := llm.NewOpenAICompat(ts.URL, "key", ts.Client())

	_, err := client.Complete(context.Background(), "sys", "user", llm.Params{Model: "gpt-4o-mini"})
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for 500 error, got: %v", err)
	}

	err = client.Stream(context.Background(), "sys", "user", llm.Params{Model: "gpt-4o-mini"}, func(string) {})
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for stream 500 error, got: %v", err)
	}
}

func TestGigaChat_OAuth_Caching_And_Refresh(t *testing.T) {
	var oauthCalls int32
	var tokenExpiresInMs int64 = 30 * 60 * 1000 // 30 minutes in ms

	expectedClientID := "test-client-id"
	expectedSecret := "test-secret"
	expectedBasic := "Basic " + base64.StdEncoding.EncodeToString([]byte(expectedClientID+":"+expectedSecret))
	expectedScope := "GIGACHAT_API_PERS"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/oauth" {
			atomic.AddInt32(&oauthCalls, 1)

			// 1. Verify RqUID is valid UUID
			rquid := r.Header.Get("RqUID")
			if _, err := uuid.Parse(rquid); err != nil {
				t.Errorf("invalid RqUID header: %s", rquid)
			}

			// 2. Verify Basic auth
			if r.Header.Get("Authorization") != expectedBasic {
				t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
			}

			// 3. Verify scope
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.Form.Get("scope") != expectedScope {
				t.Errorf("expected scope %s, got %s", expectedScope, r.Form.Get("scope"))
			}

			expiresAt := time.Now().Add(time.Duration(tokenExpiresInMs) * time.Millisecond).UnixMilli()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("token-call-%d", atomic.LoadInt32(&oauthCalls)),
				"expires_at":   expiresAt,
			})
			return
		}

		if r.URL.Path == "/v1/chat/completions" {
			auth := r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{
						"message": map[string]any{
							"role":    "assistant",
							"content": "Answer with " + auth,
						},
					},
				},
			})
			return
		}

		http.NotFound(w, r)
	}))
	defer ts.Close()

	gc, err := llm.NewGigaChatWithClient(llm.GigaChatConfig{
		ClientID:     expectedClientID,
		ClientSecret: expectedSecret,
		Scope:        expectedScope,
		OAuthURL:     ts.URL + "/api/v2/oauth",
		BaseURL:      ts.URL + "/v1",
		Model:        "GigaChat",
	}, ts.Client())
	if err != nil {
		t.Fatalf("failed to create GigaChat: %v", err)
	}

	// First call should hit OAuth
	res1, err := gc.Complete(context.Background(), "sys", "user", llm.Params{Model: "GigaChat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res1, "Bearer token-call-1") {
		t.Fatalf("expected token-call-1, got %s", res1)
	}
	if calls := atomic.LoadInt32(&oauthCalls); calls != 1 {
		t.Fatalf("expected 1 OAuth call, got %d", calls)
	}

	// Second call within cache window should NOT hit OAuth
	res2, err := gc.Complete(context.Background(), "sys", "user", llm.Params{Model: "GigaChat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res2, "Bearer token-call-1") {
		t.Fatalf("expected cached token-call-1, got %s", res2)
	}
	if calls := atomic.LoadInt32(&oauthCalls); calls != 1 {
		t.Fatalf("expected 1 OAuth call (cached), got %d", calls)
	}

	// Now simulate expired token: token expires in 10s (which is < 60s before expires_at)
	atomic.StoreInt64(&tokenExpiresInMs, 10*1000)
	gc.ExpireTokenNow() // helper for testing to invalidate cache

	res3, err := gc.Complete(context.Background(), "sys", "user", llm.Params{Model: "GigaChat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res3, "Bearer token-call-2") {
		t.Fatalf("expected refreshed token-call-2, got %s", res3)
	}
	if calls := atomic.LoadInt32(&oauthCalls); calls != 2 {
		t.Fatalf("expected 2 OAuth calls after expiration, got %d", calls)
	}
}

func TestGigaChat_CACertValidation(t *testing.T) {
	// 1. Missing CA file should return error
	_, err := llm.NewGigaChat(llm.GigaChatConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		CACert:       "/non/existent/ca.crt",
	})
	if err == nil {
		t.Fatal("expected error for non-existent CA file, got nil")
	}

	// 2. Corrupt / non-PEM CA file should return error
	tmpDir := t.TempDir()
	corruptFile := filepath.Join(tmpDir, "corrupt.crt")
	if err := os.WriteFile(corruptFile, []byte("this is not a valid pem cert"), 0644); err != nil {
		t.Fatalf("failed to write corrupt cert: %v", err)
	}

	_, err = llm.NewGigaChat(llm.GigaChatConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		CACert:       corruptFile,
	})
	if err == nil {
		t.Fatal("expected error for corrupt CA file, got nil")
	}

	// 3. Valid TLS server with custom CA cert
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tls-token",
			"expires_at":   time.Now().Add(30 * time.Minute).UnixMilli(),
		})
	}))
	defer tlsServer.Close()

	// Write server's cert to temporary PEM file
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsServer.Certificate().Raw,
	})
	validCAFile := filepath.Join(tmpDir, "valid_ca.crt")
	if err := os.WriteFile(validCAFile, certPEM, 0644); err != nil {
		t.Fatalf("failed to write valid CA file: %v", err)
	}

	gc, err := llm.NewGigaChat(llm.GigaChatConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		Scope:        "GIGACHAT_API_PERS",
		OAuthURL:     tlsServer.URL,
		CACert:       validCAFile,
	})
	if err != nil {
		t.Fatalf("failed to create GigaChat with valid CA cert: %v", err)
	}

	// Verify TLS connection works with the custom cert and InsecureSkipVerify is false
	token, err := gc.GetToken(context.Background())
	if err != nil {
		t.Fatalf("failed to connect via TLS with valid CA: %v", err)
	}
	if token != "tls-token" {
		t.Fatalf("expected tls-token, got %s", token)
	}
}
