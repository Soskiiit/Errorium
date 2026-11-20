package search_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	searchpb "github.com/soskiiit/errorium/services/gateway/internal/pb/search"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

type mockSearchServer struct {
	searchpb.UnimplementedSearchServiceServer
	searchFn      func(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error)
	healthCheckFn func(ctx context.Context, req *searchpb.HealthCheckRequest) (*searchpb.HealthCheckResponse, error)
}

func (m *mockSearchServer) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, req)
	}
	return &searchpb.SearchResponse{}, nil
}

func (m *mockSearchServer) HealthCheck(ctx context.Context, req *searchpb.HealthCheckRequest) (*searchpb.HealthCheckResponse, error) {
	if m.healthCheckFn != nil {
		return m.healthCheckFn(ctx, req)
	}
	return &searchpb.HealthCheckResponse{
		Status:  searchpb.HealthCheckResponse_SERVING,
		Version: "1.0.0",
	}, nil
}

func setupBufconnSearchClient(t *testing.T, srv searchpb.SearchServiceServer) (*search.GRPCClient, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	baseServer := grpc.NewServer()
	searchpb.RegisterSearchServiceServer(baseServer, srv)

	go func() {
		if err := baseServer.Serve(lis); err != nil && !errors.Is(err, net.ErrClosed) {
			_ = err
		}
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
	}

	client := search.NewGRPCClient(conn)
	cleanup := func() {
		_ = conn.Close()
		baseServer.Stop()
		_ = lis.Close()
	}
	return client, cleanup
}

func TestGRPCClient_Search_Success(t *testing.T) {
	mockSrv := &mockSearchServer{
		searchFn: func(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
			if req.GetQuery() != "IndexError" {
				t.Errorf("unexpected query: %s", req.GetQuery())
			}
			if req.GetTopK() != 5 {
				t.Errorf("unexpected top_k: %d", req.GetTopK())
			}
			return &searchpb.SearchResponse{
				Hits: []*searchpb.SearchHit{
					{
						DocId:            101,
						Score:            12.34,
						Title:            "How to fix IndexError",
						Snippet:          "Index out of range explanation",
						Tags:             []string{"python", "list"},
						QuestionScore:    42,
						AcceptedAnswerId: 202,
						BestAnswerScore:  50,
						CreatedAt:        1700000000,
					},
					{
						DocId:            102,
						Score:            9.87,
						Title:            "Another IndexError",
						Snippet:          "Check boundary conditions",
						Tags:             []string{"python"},
						QuestionScore:    10,
						AcceptedAnswerId: 0,
						BestAnswerScore:  5,
						CreatedAt:        1700000010,
					},
				},
				TotalHits:      2,
				LatencyMs:      15.5,
				TokenizedQuery: []string{"indexerror", "index", "error"},
			}, nil
		},
	}

	client, cleanup := setupBufconnSearchClient(t, mockSrv)
	defer cleanup()

	resp, err := client.Search(context.Background(), search.SearchRequest{
		Query: "IndexError",
		TopK:  5,
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if resp.TotalHits != 2 {
		t.Errorf("expected total hits 2, got %d", resp.TotalHits)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	first := resp.Results[0]
	if first.DocID != 101 {
		t.Errorf("expected doc_id 101, got %d", first.DocID)
	}
	if first.BM25Score != 12.34 {
		t.Errorf("expected bm25_score 12.34, got %f", first.BM25Score)
	}
	if first.QuestionScore != 42 {
		t.Errorf("expected question_score 42, got %d", first.QuestionScore)
	}
	if !first.IsAccepted {
		t.Errorf("expected is_accepted true for accepted_answer_id != 0")
	}
	expectedURL := "https://stackoverflow.com/questions/101"
	if first.URL != expectedURL {
		t.Errorf("expected url %s, got %s", expectedURL, first.URL)
	}

	second := resp.Results[1]
	if second.IsAccepted {
		t.Errorf("expected is_accepted false for accepted_answer_id == 0")
	}
}

func TestGRPCClient_Search_Error(t *testing.T) {
	mockSrv := &mockSearchServer{
		searchFn: func(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
			return nil, errors.New("search engine internal failure")
		},
	}

	client, cleanup := setupBufconnSearchClient(t, mockSrv)
	defer cleanup()

	resp, err := client.Search(context.Background(), search.SearchRequest{
		Query: "fail query",
	})
	if err == nil {
		t.Fatalf("expected error, got response: %+v", resp)
	}
}

func TestGRPCClient_RAG_Success(t *testing.T) {
	mockSrv := &mockSearchServer{
		searchFn: func(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
			return &searchpb.SearchResponse{
				Hits: []*searchpb.SearchHit{
					{
						DocId:            27539308, // Known fixture doc ID
						Score:            15.0,
						Title:            "SyntaxError: invalid syntax in Python",
						Snippet:          "Parser encountered code violating grammar rules",
						Tags:             []string{"python", "syntax-error"},
						QuestionScore:    100,
						AcceptedAnswerId: 1,
					},
				},
				TotalHits: 1,
				LatencyMs: 8.0,
			}, nil
		},
	}

	client, cleanup := setupBufconnSearchClient(t, mockSrv)
	defer cleanup()

	ragResp, err := client.RAG(context.Background(), search.SearchRequest{
		Query: "SyntaxError",
	})
	if err != nil {
		t.Fatalf("unexpected RAG error: %v", err)
	}

	if ragResp.Synthesis == "" {
		t.Error("expected non-empty synthesis")
	}
	if ragResp.Code == nil || *ragResp.Code == "" {
		t.Error("expected non-empty code snippet for known fixture")
	}
	if len(ragResp.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(ragResp.Sources))
	}
}

func TestGRPCClient_RAG_EmptyHitsFallback(t *testing.T) {
	mockSrv := &mockSearchServer{
		searchFn: func(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
			return &searchpb.SearchResponse{
				Hits:      []*searchpb.SearchHit{},
				TotalHits: 0,
			}, nil
		},
	}

	client, cleanup := setupBufconnSearchClient(t, mockSrv)
	defer cleanup()

	ragResp, err := client.RAG(context.Background(), search.SearchRequest{
		Query: "no match at all",
	})
	if err != nil {
		t.Fatalf("unexpected RAG error: %v", err)
	}

	if len(ragResp.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(ragResp.Sources))
	}
	if ragResp.Code != nil {
		t.Errorf("expected nil code, got %v", *ragResp.Code)
	}
	if ragResp.Synthesis == "" {
		t.Error("expected non-empty fallback synthesis message")
	}
}

func TestGRPCClient_HealthCheck(t *testing.T) {
	status := searchpb.HealthCheckResponse_SERVING
	var errReturn error

	mockSrv := &mockSearchServer{
		healthCheckFn: func(ctx context.Context, req *searchpb.HealthCheckRequest) (*searchpb.HealthCheckResponse, error) {
			if errReturn != nil {
				return nil, errReturn
			}
			return &searchpb.HealthCheckResponse{Status: status}, nil
		},
	}

	client, cleanup := setupBufconnSearchClient(t, mockSrv)
	defer cleanup()

	// 1. Serving -> true
	ok, err := client.HealthCheck(context.Background())
	if err != nil || !ok {
		t.Errorf("expected true, nil, got %v, %v", ok, err)
	}

	// 2. Not serving -> false, nil
	status = searchpb.HealthCheckResponse_NOT_SERVING
	ok, err = client.HealthCheck(context.Background())
	if err != nil || ok {
		t.Errorf("expected false, nil, got %v, %v", ok, err)
	}

	// 3. Error -> false, err
	errReturn = errors.New("health check RPC failed")
	ok, err = client.HealthCheck(context.Background())
	if err == nil || ok {
		t.Errorf("expected false, error, got %v, %v", ok, err)
	}
}
