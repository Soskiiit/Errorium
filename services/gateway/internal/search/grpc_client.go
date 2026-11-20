package search

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	searchpb "github.com/soskiiit/errorium/services/gateway/internal/pb/search"
)

// GRPCClient implements SearchClient by querying the search_engine_core gRPC service.
type GRPCClient struct {
	client searchpb.SearchServiceClient
}

// NewGRPCClient creates a new gRPC search client wrapping an existing connection.
func NewGRPCClient(conn *grpc.ClientConn) *GRPCClient {
	return &GRPCClient{
		client: searchpb.NewSearchServiceClient(conn),
	}
}

// Search queries search_engine_core for BM25 ranking results.
func (c *GRPCClient) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	topK := int32(req.TopK)
	if topK <= 0 {
		topK = 10
	} else if topK > 50 {
		topK = 50
	}

	rpcReq := &searchpb.SearchRequest{
		Query:          req.Query,
		TopK:           topK,
		Tags:           req.Tags,
		Language:       req.Language,
		IncludeRawBody: false,
	}

	rpcResp, err := c.client.Search(ctx, rpcReq)
	if err != nil {
		return nil, fmt.Errorf("search core RPC failed: %w", err)
	}

	tookMs := int64(rpcResp.GetLatencyMs())
	if tookMs <= 0 {
		tookMs = time.Since(start).Milliseconds()
	}

	hits := rpcResp.GetHits()
	results := make([]Source, len(hits))
	for i, hit := range hits {
		results[i] = Source{
			DocID:         hit.GetDocId(),
			Title:         hit.GetTitle(),
			URL:           fmt.Sprintf("https://stackoverflow.com/questions/%d", hit.GetDocId()),
			Snippet:       hit.GetSnippet(),
			Tags:          hit.GetTags(),
			QuestionScore: int(hit.GetQuestionScore()),
			BM25Score:     hit.GetScore(),
			IsAccepted:    hit.GetAcceptedAnswerId() != 0,
		}
	}

	return &SearchResponse{
		Query:     req.Query,
		Results:   results,
		TotalHits: rpcResp.GetTotalHits(),
		TookMs:    tookMs,
	}, nil
}

// RAG retrieves search results and generates a synthesized answer using the template strategy.
func (c *GRPCClient) RAG(ctx context.Context, req SearchRequest) (*RAGResult, error) {
	start := time.Now()

	searchResp, err := c.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	tookMs := time.Since(start).Milliseconds()

	synthesis, code := SynthesizeTemplate(req.Query, searchResp.Results)

	return &RAGResult{
		Query:     req.Query,
		Synthesis: synthesis,
		Code:      code,
		Sources:   searchResp.Results,
		TookMs:    tookMs,
	}, nil
}

// HealthCheck checks whether the core search engine is SERVING.
func (c *GRPCClient) HealthCheck(ctx context.Context) (bool, error) {
	resp, err := c.client.HealthCheck(ctx, &searchpb.HealthCheckRequest{})
	if err != nil {
		return false, err
	}
	return resp.GetStatus() == searchpb.HealthCheckResponse_SERVING, nil
}

// GetStats returns index statistics from the search engine.
func (c *GRPCClient) GetStats(ctx context.Context) (*searchpb.GetStatsResponse, error) {
	return c.client.GetStats(ctx, &searchpb.GetStatsRequest{})
}

// ReloadIndex triggers hot reloading of the index in the core engine.
func (c *GRPCClient) ReloadIndex(ctx context.Context) (*searchpb.ReloadIndexResponse, error) {
	return c.client.ReloadIndex(ctx, &searchpb.ReloadIndexRequest{})
}
