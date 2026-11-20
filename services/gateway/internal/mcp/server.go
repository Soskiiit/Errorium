package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// JSONRPCRequest represents a JSON-RPC 2.0 request or notification.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ServerInfo describes the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Capabilities describes server capabilities.
type Capabilities struct {
	Tools map[string]any `json:"tools"`
}

// InitializeResult is the result for initialize method.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// ToolsListResult is the result for tools/list method.
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolCallParams describes arguments passed to tools/call.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ExplainResult represents the result payload for errorium_explain_error.
type ExplainResult struct {
	Synthesis string          `json:"synthesis"`
	Code      *string         `json:"code"`
	Sources   []search.Source `json:"sources"`
}

// RAGSynthesizer abstracts RAG synthesis orchestration.
type RAGSynthesizer interface {
	Synthesize(ctx context.Context, query string, hits []search.Source) (*search.RAGResult, error)
}

// Server implements Model Context Protocol (MCP) JSON-RPC 2.0 handlers.
type Server struct {
	searchClient search.SearchClient
	graphClient  graph.GraphClient
	ragOrch      RAGSynthesizer
	tools        []ToolDefinition
}

// NewServer creates a new MCP Server with dependencies.
func NewServer(searchClient search.SearchClient, graphClient graph.GraphClient, ragOrch RAGSynthesizer) *Server {
	return &Server{
		searchClient: searchClient,
		graphClient:  graphClient,
		ragOrch:      ragOrch,
		tools:        DefaultTools(),
	}
}

// HandleRequest processes an incoming JSON-RPC request and returns the response.
// If the request is a notification (e.g. notifications/initialized), returns nil.
func (s *Server) HandleRequest(ctx context.Context, req JSONRPCRequest) *JSONRPCResponse {
	// Notifications (no ID)
	if len(req.ID) == 0 || string(req.ID) == "null" {
		if req.Method == "notifications/initialized" {
			return nil
		}
	}

	makeResponse := func(result any, err *JSONRPCError) *JSONRPCResponse {
		id := req.ID
		if len(id) == 0 {
			id = json.RawMessage(`null`)
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  result,
			Error:   err,
		}
	}

	switch req.Method {
	case "initialize":
		res := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools: map[string]any{},
			},
			ServerInfo: ServerInfo{
				Name:    "errorium",
				Version: "1.0.0",
			},
		}
		return makeResponse(res, nil)

	case "notifications/initialized":
		return nil

	case "ping":
		return makeResponse(map[string]any{}, nil)

	case "tools/list":
		return makeResponse(ToolsListResult{Tools: s.tools}, nil)

	case "tools/call":
		res, jsonRPCErr := s.handleToolCall(ctx, req.Params)
		if jsonRPCErr != nil {
			return makeResponse(nil, jsonRPCErr)
		}
		if res != nil {
			return makeResponse(*res, nil)
		}
		return makeResponse(nil, nil)

	default:
		return makeResponse(nil, &JSONRPCError{
			Code:    CodeMethodNotFound,
			Message: "method not found",
		})
	}
}

func (s *Server) handleToolCall(ctx context.Context, rawParams json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var params ToolCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, &JSONRPCError{
			Code:    CodeInvalidParams,
			Message: "invalid params",
		}
	}

	// Validate tool existence
	var toolFound bool
	for _, t := range s.tools {
		if t.Name == params.Name {
			toolFound = true
			break
		}
	}
	if !toolFound {
		return nil, &JSONRPCError{
			Code:    CodeInvalidParams, // -32602
			Message: "unknown tool",
		}
	}

	switch params.Name {
	case "errorium_search_solution":
		return s.executeSearchSolution(ctx, params.Arguments)
	case "errorium_explain_error":
		return s.executeExplainError(ctx, params.Arguments)
	case "errorium_get_related_errors":
		return s.executeGetRelatedErrors(ctx, params.Arguments)
	default:
		return nil, &JSONRPCError{
			Code:    CodeInvalidParams,
			Message: "unknown tool",
		}
	}
}

func (s *Server) executeSearchSolution(ctx context.Context, rawArgs json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var args struct {
		Query string   `json:"query"`
		Tags  []string `json:"tags"`
		Limit int      `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}

	if strings.TrimSpace(args.Query) == "" {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "query is required"}},
			IsError: true,
		}, nil
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 5
	} else if limit > 20 {
		limit = 20
	}

	if s.searchClient == nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "search client unavailable"}},
			IsError: true,
		}, nil
	}

	resp, err := s.searchClient.Search(ctx, search.SearchRequest{
		Query: args.Query,
		Tags:  args.Tags,
		TopK:  limit,
	})
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	results := resp.Results
	if results == nil {
		results = []search.Source{}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

func (s *Server) executeExplainError(ctx context.Context, rawArgs json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var args struct {
		ErrorMessage string `json:"error_message"`
		StackTrace   string `json:"stack_trace"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}

	if strings.TrimSpace(args.ErrorMessage) == "" {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "error_message is required"}},
			IsError: true,
		}, nil
	}

	query := args.ErrorMessage
	if strings.TrimSpace(args.StackTrace) != "" {
		query = args.ErrorMessage + "\n" + args.StackTrace
	}

	var hits []search.Source
	if s.searchClient != nil {
		searchResp, err := s.searchClient.Search(ctx, search.SearchRequest{
			Query: query,
			TopK:  5,
		})
		if err == nil && searchResp != nil {
			hits = searchResp.Results
		}
	}

	var ragRes *search.RAGResult
	var err error
	if s.ragOrch != nil {
		ragRes, err = s.ragOrch.Synthesize(ctx, query, hits)
	} else if s.searchClient != nil {
		ragRes, err = s.searchClient.RAG(ctx, search.SearchRequest{
			Query: query,
			TopK:  5,
		})
	} else {
		err = errors.New("rag orchestrator unavailable")
	}

	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	sources := ragRes.Sources
	if sources == nil {
		sources = []search.Source{}
	}

	explainRes := ExplainResult{
		Synthesis: ragRes.Synthesis,
		Code:      ragRes.Code,
		Sources:   sources,
	}

	data, err := json.Marshal(explainRes)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

func (s *Server) executeGetRelatedErrors(ctx context.Context, rawArgs json.RawMessage) (*ToolCallResult, *JSONRPCError) {
	var args struct {
		ErrorName string `json:"error_name"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}

	if strings.TrimSpace(args.ErrorName) == "" {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "error_name is required"}},
			IsError: true,
		}, nil
	}

	if s.searchClient == nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "search client unavailable"}},
			IsError: true,
		}, nil
	}

	searchResp, err := s.searchClient.Search(ctx, search.SearchRequest{
		Query: args.ErrorName,
		TopK:  1,
	})
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	var threadGraph *graph.ThreadGraph
	if len(searchResp.Results) > 0 && s.graphClient != nil {
		topDocID := searchResp.Results[0].DocID
		threadGraph, err = s.graphClient.GetThreadGraph(ctx, []int64{topDocID})
		if err != nil {
			return &ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
	} else {
		threadGraph = &graph.ThreadGraph{
			Nodes: []graph.ThreadNode{},
			Edges: []graph.ThreadEdge{},
		}
	}

	if threadGraph.Nodes == nil {
		threadGraph.Nodes = []graph.ThreadNode{}
	}
	if threadGraph.Edges == nil {
		threadGraph.Edges = []graph.ThreadEdge{}
	}

	data, err := json.Marshal(threadGraph)
	if err != nil {
		return &ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &ToolCallResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}
