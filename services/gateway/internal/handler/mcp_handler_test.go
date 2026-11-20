package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/mcp"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// MockSearchForHandler mocks SearchClient.
type MockSearchForHandler struct {
	SearchFunc func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error)
}

func (m *MockSearchForHandler) Search(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, req)
	}
	return &search.SearchResponse{}, nil
}

func (m *MockSearchForHandler) RAG(ctx context.Context, req search.SearchRequest) (*search.RAGResult, error) {
	return &search.RAGResult{}, nil
}

// MockGraphForHandler mocks GraphClient.
type MockGraphForHandler struct {
	GetThreadGraphFunc func(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error)
}

func (m *MockGraphForHandler) GetThreadGraph(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
	if m.GetThreadGraphFunc != nil {
		return m.GetThreadGraphFunc(ctx, docIDs)
	}
	return &graph.ThreadGraph{}, nil
}

func (m *MockGraphForHandler) GetThreadDetails(ctx context.Context, docID int64) (*graph.ThreadDetails, error) {
	return &graph.ThreadDetails{}, nil
}

// MockRAGForHandler mocks RAGSynthesizer.
type MockRAGForHandler struct {
	SynthesizeFunc func(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error)
}

func (m *MockRAGForHandler) Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(ctx, query, hits)
	}
	return &search.RAGResult{}, nil
}

// Helper to read SSE events from a stream.
type sseEvent struct {
	Event string
	Data  string
}

func readSSEEvents(r *bufio.Reader) (sseEvent, error) {
	var ev sseEvent
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return ev, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// Blank line terminates an event
			if ev.Event != "" || ev.Data != "" {
				return ev, nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			// Comment / keepalive line
			continue
		}
		if strings.HasPrefix(line, "event:") {
			ev.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			ev.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}

func TestMCPHandler_FullJSONRPCFlow(t *testing.T) {
	mockSearch := &MockSearchForHandler{
		SearchFunc: func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
			return &search.SearchResponse{
				Results: []search.Source{
					{DocID: 100, Title: "SO Solution 100"},
				},
				TotalHits: 1,
			}, nil
		},
	}
	mockGraph := &MockGraphForHandler{
		GetThreadGraphFunc: func(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
			return &graph.ThreadGraph{
				Nodes: []graph.ThreadNode{{DocID: 100, Title: "SO Solution 100"}},
				Edges: []graph.ThreadEdge{},
			}, nil
		},
	}
	codeFix := "val = 42"
	mockRAG := &MockRAGForHandler{
		SynthesizeFunc: func(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
			return &search.RAGResult{
				Query:     query,
				Synthesis: "Analysis of the error",
				Code:      &codeFix,
				Sources:   hits,
			}, nil
		},
	}

	server := mcp.NewServer(mockSearch, mockGraph, mockRAG)
	limiter := mcp.NewLimiter(mcp.LimiterConfig{
		MaxSessionsPerIP: 10,
		MaxSessions:      20,
		CallsPerMin:      60,
	})
	handler := NewMCPHandler(server, limiter)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Connect to GET /sse
	sseCtx, cancelSSE := context.WithCancel(context.Background())
	defer cancelSSE()

	sseReq, err := http.NewRequestWithContext(sseCtx, "GET", ts.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("failed to create SSE request: %v", err)
	}
	sseReq.Header.Set("Authorization", "Bearer ignored-token-adr-012")

	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /sse, got %d", sseResp.StatusCode)
	}
	if ct := sseResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}
	if cc := sseResp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("expected Cache-Control no-cache, got %q", cc)
	}
	if xab := sseResp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("expected X-Accel-Buffering no, got %q", xab)
	}

	reader := bufio.NewReader(sseResp.Body)

	// Read initial endpoint event
	firstEv, err := readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read initial SSE event: %v", err)
	}
	if firstEv.Event != "endpoint" {
		t.Fatalf("expected event: endpoint, got %q", firstEv.Event)
	}
	if !strings.HasPrefix(firstEv.Data, "/mcp/messages?sessionId=") {
		t.Fatalf("expected data to start with /mcp/messages?sessionId=, got %q", firstEv.Data)
	}

	sessionID := strings.TrimPrefix(firstEv.Data, "/mcp/messages?sessionId=")
	if sessionID == "" {
		t.Fatalf("empty sessionId in endpoint event")
	}

	postMessage := func(payload string) (*http.Response, error) {
		url := ts.URL + "/mcp/messages?sessionId=" + sessionID
		req, err := http.NewRequest("POST", url, bytes.NewBufferString(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return http.DefaultClient.Do(req)
	}

	// 2. initialize
	resp, err := postMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	if err != nil {
		t.Fatalf("initialize POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err := readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for initialize: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}
	var initResp mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(ev.Data), &initResp); err != nil {
		t.Fatalf("failed to unmarshal JSON-RPC initialize response: %v", err)
	}
	if string(initResp.ID) != "1" {
		t.Fatalf("expected id 1, got %s", string(initResp.ID))
	}

	// 3. notifications/initialized (no response in SSE expected)
	resp, err = postMessage(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if err != nil {
		t.Fatalf("notifications/initialized POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 4. tools/list
	resp, err = postMessage(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if err != nil {
		t.Fatalf("tools/list POST failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for tools/list: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}
	var listResp mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(ev.Data), &listResp); err != nil {
		t.Fatalf("failed to unmarshal JSON-RPC tools/list response: %v", err)
	}
	if string(listResp.ID) != "2" {
		t.Fatalf("expected id 2, got %s", string(listResp.ID))
	}

	// 5. tools/call: errorium_search_solution
	resp, err = postMessage(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"errorium_search_solution","arguments":{"query":"test error"}}}`)
	if err != nil {
		t.Fatalf("tools/call search_solution failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for search_solution: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}

	// 6. tools/call: errorium_explain_error
	resp, err = postMessage(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"errorium_explain_error","arguments":{"error_message":"NullPointerException"}}}`)
	if err != nil {
		t.Fatalf("tools/call explain_error failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for explain_error: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}

	// 7. tools/call: errorium_get_related_errors
	resp, err = postMessage(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"errorium_get_related_errors","arguments":{"error_name":"IndexError"}}}`)
	if err != nil {
		t.Fatalf("tools/call get_related_errors failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for get_related_errors: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}

	// 8. ping
	resp, err = postMessage(`{"jsonrpc":"2.0","id":6,"method":"ping"}`)
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for ping: %v", err)
	}
	if ev.Event != "message" {
		t.Fatalf("expected event: message, got %q", ev.Event)
	}

	// 9. unknown method
	resp, err = postMessage(`{"jsonrpc":"2.0","id":7,"method":"foo/bar"}`)
	if err != nil {
		t.Fatalf("unknown method failed: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	ev, err = readSSEEvents(reader)
	if err != nil {
		t.Fatalf("failed to read SSE event for unknown method: %v", err)
	}
	var errResp mcp.JSONRPCResponse
	if err := json.Unmarshal([]byte(ev.Data), &errResp); err != nil {
		t.Fatalf("failed to unmarshal JSON-RPC error response: %v", err)
	}
	if errResp.Error == nil || errResp.Error.Code != -32601 {
		t.Fatalf("expected error code -32601, got %+v", errResp.Error)
	}
}

func TestMCPHandler_InvalidOrMissingSession(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	limiter := mcp.NewLimiter(mcp.LimiterConfig{})
	handler := NewMCPHandler(server, limiter)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Missing sessionId
	resp, err := http.Post(ts.URL+"/mcp/messages", "application/json", strings.NewReader(`{"jsonrpc":"2.0"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing sessionId, got %d", resp.StatusCode)
	}

	// Unknown sessionId
	resp2, err := http.Post(ts.URL+"/mcp/messages?sessionId=00000000-0000-0000-0000-000000000000", "application/json", strings.NewReader(`{"jsonrpc":"2.0"}`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown sessionId, got %d", resp2.StatusCode)
	}
}

func TestMCPHandler_MalformedJSON(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	limiter := mcp.NewLimiter(mcp.LimiterConfig{})
	handler := NewMCPHandler(server, limiter)

	// Register a session manually in handler and limiter
	sessID := "test-sess-bad-json"
	limiter.RegisterSession(sessID, "127.0.0.1")
	handler.registerSessionForTest(sessID)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp/messages?sessionId="+sessID, "application/json", strings.NewReader(`{bad json`))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for malformed JSON, got %d", resp.StatusCode)
	}
}

func TestMCPHandler_CallRateLimit(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	// Allow only 2 calls per minute
	limiter := mcp.NewLimiter(mcp.LimiterConfig{
		CallsPerMin: 2,
	})
	handler := NewMCPHandler(server, limiter)

	sessID := "test-sess-ratelimit"
	limiter.RegisterSession(sessID, "127.0.0.1")
	msgChan := handler.registerSessionForTest(sessID)

	// Consume messages in background so chan does not block
	go func() {
		for range msgChan {
		}
	}()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	callPayload := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"errorium_search_solution","arguments":{"query":"q"}}}`

	// Call 1: 202
	resp1, err := http.Post(ts.URL+"/mcp/messages?sessionId="+sessID, "application/json", strings.NewReader(callPayload))
	if err != nil || resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("call 1 failed: %v, status: %d", err, resp1.StatusCode)
	}
	_ = resp1.Body.Close()

	// Call 2: 202
	resp2, err := http.Post(ts.URL+"/mcp/messages?sessionId="+sessID, "application/json", strings.NewReader(callPayload))
	if err != nil || resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("call 2 failed: %v, status: %d", err, resp2.StatusCode)
	}
	_ = resp2.Body.Close()

	// Call 3: 429 Too Many Requests
	resp3, err := http.Post(ts.URL+"/mcp/messages?sessionId="+sessID, "application/json", strings.NewReader(callPayload))
	if err != nil {
		t.Fatalf("call 3 request failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests for call 3, got %d", resp3.StatusCode)
	}
	if retryAfter := resp3.Header.Get("Retry-After"); retryAfter != "5" {
		t.Fatalf("expected Retry-After: 5, got %q", retryAfter)
	}

	var errBody map[string]string
	_ = json.NewDecoder(resp3.Body).Decode(&errBody)
	if errBody["error"] != "rate limit exceeded" {
		t.Fatalf("expected error: rate limit exceeded, got %q", errBody["error"])
	}

	// Other methods (like ping) should still pass through because rate limit applies only to tools/call!
	pingPayload := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	respPing, err := http.Post(ts.URL+"/mcp/messages?sessionId="+sessID, "application/json", strings.NewReader(pingPayload))
	if err != nil || respPing.StatusCode != http.StatusAccepted {
		t.Fatalf("ping failed when tools/call rate limited: status %d", respPing.StatusCode)
	}
	_ = respPing.Body.Close()
}

func TestMCPHandler_SessionLimitsAndCleanup(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	// Limits: 2 sessions per IP, 3 global
	limiter := mcp.NewLimiter(mcp.LimiterConfig{
		MaxSessionsPerIP: 2,
		MaxSessions:      3,
		CallsPerMin:      60,
	})
	handler := NewMCPHandler(server, limiter)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	connectSSE := func(ip string, ctx context.Context) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", ts.URL+"/sse", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Real-IP", ip)
		return http.DefaultClient.Do(req)
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	resp1, err := connectSSE("192.168.1.1", ctx1)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("conn 1 failed: status %d", resp1.StatusCode)
	}
	defer resp1.Body.Close()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resp2, err := connectSSE("192.168.1.1", ctx2)
	if err != nil || resp2.StatusCode != http.StatusOK {
		t.Fatalf("conn 2 failed: status %d", resp2.StatusCode)
	}
	defer resp2.Body.Close()

	// 3rd session on same IP -> 429
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	resp3, err := connectSSE("192.168.1.1", ctx3)
	if err != nil {
		t.Fatalf("conn 3 failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for 3rd session on same IP, got %d", resp3.StatusCode)
	}
	if resp3.Header.Get("Retry-After") != "5" {
		t.Fatalf("expected Retry-After: 5, got %q", resp3.Header.Get("Retry-After"))
	}

	// 3rd global session on DIFFERENT IP -> 200 OK
	ctx4, cancel4 := context.WithCancel(context.Background())
	defer cancel4()
	resp4, err := connectSSE("192.168.1.2", ctx4)
	if err != nil || resp4.StatusCode != http.StatusOK {
		t.Fatalf("conn 4 failed: status %d", resp4.StatusCode)
	}
	defer resp4.Body.Close()

	// 4th global session -> 503 Service Unavailable
	ctx5, cancel5 := context.WithCancel(context.Background())
	defer cancel5()
	resp5, err := connectSSE("192.168.1.3", ctx5)
	if err != nil {
		t.Fatalf("conn 5 failed: %v", err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for global ceiling, got %d", resp5.StatusCode)
	}
	if resp5.Header.Get("Retry-After") != "5" {
		t.Fatalf("expected Retry-After: 5, got %q", resp5.Header.Get("Retry-After"))
	}

	// Now cancel session 1 -> frees both per-IP slot and global slot
	cancel1()
	// Allow a few milliseconds for goroutine cleanup
	time.Sleep(50 * time.Millisecond)

	// Now 192.168.1.1 can connect again!
	ctx6, cancel6 := context.WithCancel(context.Background())
	defer cancel6()
	resp6, err := connectSSE("192.168.1.1", ctx6)
	if err != nil || resp6.StatusCode != http.StatusOK {
		t.Fatalf("conn 6 should succeed after slot freed, got %d", resp6.StatusCode)
	}
	defer resp6.Body.Close()
}

func TestMCPHandler_KeepalivePing(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	limiter := mcp.NewLimiter(mcp.LimiterConfig{})
	handler := NewMCPHandler(server, limiter)
	// Use 50ms keepalive for test
	handler.SetPingInterval(50 * time.Millisecond)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/sse", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// Read first line (endpoint event)
	ev, err := readSSEEvents(reader)
	if err != nil || ev.Event != "endpoint" {
		t.Fatalf("failed reading endpoint event: %v, ev: %+v", err, ev)
	}

	// Read next lines to find `: ping`
	gotPing := false
	deadline := time.After(500 * time.Millisecond)
	for !gotPing {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for : ping keepalive")
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("error reading line: %v", err)
			}
			if strings.TrimSpace(line) == ": ping" {
				gotPing = true
			}
		}
	}
}

func TestMCPHandler_Race(t *testing.T) {
	server := mcp.NewServer(nil, nil, nil)
	limiter := mcp.NewLimiter(mcp.LimiterConfig{
		MaxSessionsPerIP: 100,
		MaxSessions:      200,
		CallsPerMin:      1000,
	})
	handler := NewMCPHandler(server, limiter)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/sse", nil)
			req.Header.Set("X-Real-IP", fmt.Sprintf("10.0.0.%d", id))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			reader := bufio.NewReader(resp.Body)
			ev, err := readSSEEvents(reader)
			if err != nil {
				return
			}
			sessID := strings.TrimPrefix(ev.Data, "/mcp/messages?sessionId=")
			if sessID != "" {
				postURL := ts.URL + "/mcp/messages?sessionId=" + sessID
				_, _ = http.Post(postURL, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
			}
		}(i)
	}
	wg.Wait()
}
