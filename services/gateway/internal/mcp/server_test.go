package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// MockSearchClient implements search.SearchClient for testing.
type MockSearchClient struct {
	SearchFunc func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error)
	RAGFunc    func(ctx context.Context, req search.SearchRequest) (*search.RAGResult, error)
}

func (m *MockSearchClient) Search(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, req)
	}
	return &search.SearchResponse{}, nil
}

func (m *MockSearchClient) RAG(ctx context.Context, req search.SearchRequest) (*search.RAGResult, error) {
	if m.RAGFunc != nil {
		return m.RAGFunc(ctx, req)
	}
	return &search.RAGResult{}, nil
}

// MockGraphClient implements graph.GraphClient for testing.
type MockGraphClient struct {
	GetThreadGraphFunc   func(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error)
	GetThreadDetailsFunc func(ctx context.Context, docID int64) (*graph.ThreadDetails, error)
}

func (m *MockGraphClient) GetThreadGraph(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
	if m.GetThreadGraphFunc != nil {
		return m.GetThreadGraphFunc(ctx, docIDs)
	}
	return &graph.ThreadGraph{}, nil
}

func (m *MockGraphClient) GetThreadDetails(ctx context.Context, docID int64) (*graph.ThreadDetails, error) {
	if m.GetThreadDetailsFunc != nil {
		return m.GetThreadDetailsFunc(ctx, docID)
	}
	return &graph.ThreadDetails{}, nil
}

// MockRAGSynthesizer implements RAGSynthesizer for testing.
type MockRAGSynthesizer struct {
	SynthesizeFunc func(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error)
}

func (m *MockRAGSynthesizer) Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(ctx, query, hits)
	}
	return &search.RAGResult{}, nil
}

func TestServer_Initialize(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if string(resp.ID) != "1" {
		t.Fatalf("expected ID 1, got %s", string(resp.ID))
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	initRes, ok := resp.Result.(InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}
	if initRes.ProtocolVersion != "2024-11-05" {
		t.Fatalf("expected protocolVersion 2024-11-05, got %s", initRes.ProtocolVersion)
	}
	if initRes.ServerInfo.Name != "errorium" || initRes.ServerInfo.Version != "1.0.0" {
		t.Fatalf("unexpected serverInfo: %+v", initRes.ServerInfo)
	}
}

func TestServer_NotificationsInitialized(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	resp := server.HandleRequest(ctx, req)
	if resp != nil {
		t.Fatalf("expected nil response for notifications/initialized, got %+v", resp)
	}
}

func TestServer_Ping(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"ping-1"`),
		Method:  "ping",
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if string(resp.ID) != `"ping-1"` {
		t.Fatalf("expected ID ping-1, got %s", string(resp.ID))
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestServer_ToolsList(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	listRes, ok := resp.Result.(ToolsListResult)
	if !ok {
		t.Fatalf("expected ToolsListResult, got %T", resp.Result)
	}

	if len(listRes.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(listRes.Tools))
	}

	toolsMap := make(map[string]ToolDefinition)
	for _, tool := range listRes.Tools {
		toolsMap[tool.Name] = tool
	}

	// 1. errorium_search_solution
	searchTool, ok := toolsMap["errorium_search_solution"]
	if !ok {
		t.Fatalf("missing errorium_search_solution tool")
	}
	if searchTool.InputSchema.Type != "object" {
		t.Fatalf("expected inputSchema.type=object")
	}
	if _, ok := searchTool.InputSchema.Properties["query"]; !ok {
		t.Fatalf("missing query property in search solution schema")
	}

	// 2. errorium_explain_error
	explainTool, ok := toolsMap["errorium_explain_error"]
	if !ok {
		t.Fatalf("missing errorium_explain_error tool")
	}
	if _, ok := explainTool.InputSchema.Properties["error_message"]; !ok {
		t.Fatalf("missing error_message property in explain error schema")
	}

	// 3. errorium_get_related_errors
	relatedTool, ok := toolsMap["errorium_get_related_errors"]
	if !ok {
		t.Fatalf("missing errorium_get_related_errors tool")
	}
	if _, ok := relatedTool.InputSchema.Properties["error_name"]; !ok {
		t.Fatalf("missing error_name property in related errors schema")
	}
}

func TestServer_ToolsCall_SearchSolution(t *testing.T) {
	var capturedReq search.SearchRequest
	mockSearch := &MockSearchClient{
		SearchFunc: func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
			capturedReq = req
			return &search.SearchResponse{
				Results: []search.Source{
					{DocID: 101, Title: "NullPointerException fix", URL: "https://stackoverflow.com/q/101"},
				},
				TotalHits: 1,
			}, nil
		},
	}

	server := NewServer(mockSearch, nil, nil)
	ctx := context.Background()

	paramsJSON := `{"name":"errorium_search_solution","arguments":{"query":"NullPointerException in Java","tags":["java"],"limit":50}}`
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`10`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Check clamped limit
	if capturedReq.TopK != 20 {
		t.Fatalf("expected limit to be clamped to 20, got %d", capturedReq.TopK)
	}

	callRes, ok := resp.Result.(ToolCallResult)
	if !ok {
		t.Fatalf("expected ToolCallResult, got %T", resp.Result)
	}
	if callRes.IsError {
		t.Fatalf("expected isError=false")
	}
	if len(callRes.Content) != 1 || callRes.Content[0].Type != "text" {
		t.Fatalf("unexpected content: %+v", callRes.Content)
	}

	var hits []search.Source
	if err := json.Unmarshal([]byte(callRes.Content[0].Text), &hits); err != nil {
		t.Fatalf("failed to unmarshal content text: %v", err)
	}
	if len(hits) != 1 || hits[0].DocID != 101 {
		t.Fatalf("unexpected hits: %+v", hits)
	}
}

func TestServer_ToolsCall_ExplainError(t *testing.T) {
	mockSearch := &MockSearchClient{
		SearchFunc: func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
			return &search.SearchResponse{
				Results: []search.Source{
					{DocID: 202, Title: "Index out of range", Snippet: "Check list bounds"},
				},
			}, nil
		},
	}

	var capturedQuery string
	codeSnippet := "if idx < len(lst):"
	mockRAG := &MockRAGSynthesizer{
		SynthesizeFunc: func(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error) {
			capturedQuery = query
			return &search.RAGResult{
				Query:     query,
				Synthesis: "The index is out of bounds.",
				Code:      &codeSnippet,
				Sources:   hits,
			}, nil
		},
	}

	server := NewServer(mockSearch, nil, mockRAG)
	ctx := context.Background()

	paramsJSON := `{"name":"errorium_explain_error","arguments":{"error_message":"IndexError: list index out of range","stack_trace":"File main.py, line 5"}}`
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`11`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	expectedQuery := "IndexError: list index out of range\nFile main.py, line 5"
	if capturedQuery != expectedQuery {
		t.Fatalf("expected query %q, got %q", expectedQuery, capturedQuery)
	}

	callRes, ok := resp.Result.(ToolCallResult)
	if !ok {
		t.Fatalf("expected ToolCallResult, got %T", resp.Result)
	}
	if callRes.IsError {
		t.Fatalf("expected isError=false")
	}

	var explainData struct {
		Synthesis string          `json:"synthesis"`
		Code      *string         `json:"code"`
		Sources   []search.Source `json:"sources"`
	}
	if err := json.Unmarshal([]byte(callRes.Content[0].Text), &explainData); err != nil {
		t.Fatalf("failed to unmarshal explain JSON: %v", err)
	}
	if explainData.Synthesis != "The index is out of bounds." {
		t.Fatalf("unexpected synthesis: %q", explainData.Synthesis)
	}
	if explainData.Code == nil || *explainData.Code != codeSnippet {
		t.Fatalf("unexpected code: %v", explainData.Code)
	}
	if len(explainData.Sources) != 1 || explainData.Sources[0].DocID != 202 {
		t.Fatalf("unexpected sources: %+v", explainData.Sources)
	}
}

func TestServer_ToolsCall_GetRelatedErrors(t *testing.T) {
	mockSearch := &MockSearchClient{
		SearchFunc: func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
			if req.Query == "ZeroDivisionError" {
				return &search.SearchResponse{
					Results: []search.Source{{DocID: 303}},
				}, nil
			}
			return &search.SearchResponse{}, nil
		},
	}

	var capturedDocIDs []int64
	mockGraph := &MockGraphClient{
		GetThreadGraphFunc: func(ctx context.Context, docIDs []int64) (*graph.ThreadGraph, error) {
			capturedDocIDs = docIDs
			return &graph.ThreadGraph{
				Nodes: []graph.ThreadNode{
					{DocID: 303, Title: "ZeroDivisionError fix"},
				},
				Edges: []graph.ThreadEdge{
					{SourceDocID: 303, TargetDocID: 304, Type: "DUPLICATE_OF"},
				},
			}, nil
		},
	}

	server := NewServer(mockSearch, mockGraph, nil)
	ctx := context.Background()

	paramsJSON := `{"name":"errorium_get_related_errors","arguments":{"error_name":"ZeroDivisionError"}}`
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`12`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if len(capturedDocIDs) != 1 || capturedDocIDs[0] != 303 {
		t.Fatalf("expected GetThreadGraph to be called with [303], got %v", capturedDocIDs)
	}

	callRes, ok := resp.Result.(ToolCallResult)
	if !ok {
		t.Fatalf("expected ToolCallResult, got %T", resp.Result)
	}
	if callRes.IsError {
		t.Fatalf("expected isError=false")
	}

	var threadGraph graph.ThreadGraph
	if err := json.Unmarshal([]byte(callRes.Content[0].Text), &threadGraph); err != nil {
		t.Fatalf("failed to unmarshal graph JSON: %v", err)
	}
	if len(threadGraph.Nodes) != 1 || threadGraph.Nodes[0].DocID != 303 {
		t.Fatalf("unexpected graph nodes: %+v", threadGraph.Nodes)
	}
}

func TestServer_ToolsCall_ToolExecutionError(t *testing.T) {
	mockSearch := &MockSearchClient{
		SearchFunc: func(ctx context.Context, req search.SearchRequest) (*search.SearchResponse, error) {
			return nil, errors.New("search backend connection refused")
		},
	}

	server := NewServer(mockSearch, nil, nil)
	ctx := context.Background()

	paramsJSON := `{"name":"errorium_search_solution","arguments":{"query":"something"}}`
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`13`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("tool execution error should be returned in result, not JSON-RPC error")
	}

	callRes, ok := resp.Result.(ToolCallResult)
	if !ok {
		t.Fatalf("expected ToolCallResult, got %T", resp.Result)
	}
	if !callRes.IsError {
		t.Fatalf("expected isError=true for backend failure")
	}
	if callRes.Content[0].Text != "search backend connection refused" {
		t.Fatalf("unexpected error text: %q", callRes.Content[0].Text)
	}
}

func TestServer_UnknownTool(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	paramsJSON := `{"name":"unknown_nonexistent_tool","arguments":{}}`
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`14`),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatalf("expected JSON-RPC error for unknown tool")
	}
	if resp.Error.Code != -32602 || resp.Error.Message != "unknown tool" {
		t.Fatalf("expected error -32602 'unknown tool', got %+v", resp.Error)
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	server := NewServer(nil, nil, nil)
	ctx := context.Background()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`15`),
		Method:  "custom/unknownMethod",
	}

	resp := server.HandleRequest(ctx, req)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatalf("expected JSON-RPC error for unknown method")
	}
	if resp.Error.Code != -32601 || resp.Error.Message != "method not found" {
		t.Fatalf("expected error -32601 'method not found', got %+v", resp.Error)
	}
}
