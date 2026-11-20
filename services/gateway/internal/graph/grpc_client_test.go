package graph_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	graphpb "github.com/soskiiit/errorium/services/gateway/internal/pb/graph"
)

type mockGraphServer struct {
	graphpb.UnimplementedGraphServiceServer
	getThreadGraphFn   func(ctx context.Context, req *graphpb.ThreadGraphRequest) (*graphpb.ThreadGraphResponse, error)
	getThreadDetailsFn func(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error)
}

func (m *mockGraphServer) GetThreadGraph(ctx context.Context, req *graphpb.ThreadGraphRequest) (*graphpb.ThreadGraphResponse, error) {
	if m.getThreadGraphFn != nil {
		return m.getThreadGraphFn(ctx, req)
	}
	return &graphpb.ThreadGraphResponse{}, nil
}

func (m *mockGraphServer) GetThreadDetails(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error) {
	if m.getThreadDetailsFn != nil {
		return m.getThreadDetailsFn(ctx, req)
	}
	return &graphpb.ThreadDetailsResponse{}, nil
}

func setupBufconnGraphClient(t *testing.T, srv graphpb.GraphServiceServer) (*graph.GRPCClient, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	baseServer := grpc.NewServer()
	graphpb.RegisterGraphServiceServer(baseServer, srv)

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

	client := graph.NewGRPCClient(conn)
	cleanup := func() {
		_ = conn.Close()
		baseServer.Stop()
		_ = lis.Close()
	}
	return client, cleanup
}

func TestGRPCClient_GetThreadGraph_Success(t *testing.T) {
	mockSrv := &mockGraphServer{
		getThreadGraphFn: func(ctx context.Context, req *graphpb.ThreadGraphRequest) (*graphpb.ThreadGraphResponse, error) {
			if len(req.GetDocIds()) != 2 || req.GetDocIds()[0] != 100 || req.GetDocIds()[1] != 200 {
				t.Errorf("unexpected doc_ids: %v", req.GetDocIds())
			}
			return &graphpb.ThreadGraphResponse{
				Nodes: []*graphpb.ThreadNode{
					{
						DocId:             100,
						Category:          graphpb.NodeCategory_NODE_CATEGORY_PRIMARY_RESULT,
						Score:             55.0,
						Title:             "Primary Query 100",
						Snippet:           "Primary snippet",
						HasAcceptedAnswer: true,
						Tags:              []string{"python", "list"},
						BestAnswerScore:   25,
					},
					{
						DocId:             200,
						Category:          graphpb.NodeCategory_NODE_CATEGORY_OTHER_RESULT,
						Score:             30.0,
						Title:             "Other Query 200",
						Snippet:           "Other snippet",
						HasAcceptedAnswer: false,
						Tags:              []string{"python"},
						BestAnswerScore:   10,
					},
					{
						DocId:             300,
						Category:          graphpb.NodeCategory_NODE_CATEGORY_DUPLICATE,
						Score:             15.0,
						Title:             "Duplicate Thread 300",
						Snippet:           "Duplicate snippet",
						HasAcceptedAnswer: true,
						Tags:              []string{"python"},
						BestAnswerScore:   8,
					},
					{
						DocId:             400,
						Category:          graphpb.NodeCategory_NODE_CATEGORY_LINKED,
						Score:             12.0,
						Title:             "Linked Thread 400",
						Snippet:           "Linked snippet",
						HasAcceptedAnswer: false,
						Tags:              []string{"python"},
						BestAnswerScore:   5,
					},
				},
				Edges: []*graphpb.ThreadEdge{
					{
						SourceDocId: 100,
						TargetDocId: 300,
						Type:        graphpb.EdgeType_EDGE_TYPE_DUPLICATE_OF,
					},
					{
						SourceDocId: 100,
						TargetDocId: 400,
						Type:        graphpb.EdgeType_EDGE_TYPE_LINKED_IN_ANSWER,
					},
				},
			}, nil
		},
		getThreadDetailsFn: func(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error) {
			return &graphpb.ThreadDetailsResponse{
				DocId:          req.GetDocId(),
				Title:          "Enriched title",
				BestAnswerBody: "<p>Enriched answer body for thread</p>",
				OriginalUrl:    "https://stackoverflow.com/questions/" + string(rune('0'+req.GetDocId()%10)),
			}, nil
		},
	}

	client, cleanup := setupBufconnGraphClient(t, mockSrv)
	defer cleanup()

	g, err := client.GetThreadGraph(context.Background(), []int64{100, 200})
	if err != nil {
		t.Fatalf("unexpected GetThreadGraph error: %v", err)
	}

	if len(g.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}

	// Verify categories
	expectedCategories := []string{"PRIMARY_RESULT", "OTHER_RESULT", "DUPLICATE", "LINKED"}
	for i, expected := range expectedCategories {
		if g.Nodes[i].Category != expected {
			t.Errorf("node[%d] expected category %s, got %s", i, expected, g.Nodes[i].Category)
		}
		// Verify enrichment
		if g.Nodes[i].BestAnswerBody != "<p>Enriched answer body for thread</p>" {
			t.Errorf("node[%d] best_answer_body not enriched, got: %s", i, g.Nodes[i].BestAnswerBody)
		}
		if g.Nodes[i].OriginalURL == "" {
			t.Errorf("node[%d] original_url empty", i)
		}
	}

	// Verify edge types
	if g.Edges[0].Type != "DUPLICATE_OF" {
		t.Errorf("expected edge 0 type DUPLICATE_OF, got %s", g.Edges[0].Type)
	}
	if g.Edges[1].Type != "LINKED_IN_ANSWER" {
		t.Errorf("expected edge 1 type LINKED_IN_ANSWER, got %s", g.Edges[1].Type)
	}
}

func TestGRPCClient_GetThreadGraph_Error(t *testing.T) {
	mockSrv := &mockGraphServer{
		getThreadGraphFn: func(ctx context.Context, req *graphpb.ThreadGraphRequest) (*graphpb.ThreadGraphResponse, error) {
			return nil, errors.New("graph core RPC failure")
		},
	}

	client, cleanup := setupBufconnGraphClient(t, mockSrv)
	defer cleanup()

	g, err := client.GetThreadGraph(context.Background(), []int64{100})
	if err == nil {
		t.Fatalf("expected error, got graph: %+v", g)
	}
}

func TestGRPCClient_GetThreadDetails_Success(t *testing.T) {
	mockSrv := &mockGraphServer{
		getThreadDetailsFn: func(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error) {
			if req.GetDocId() != 42 {
				t.Errorf("unexpected doc_id: %d", req.GetDocId())
			}
			return &graphpb.ThreadDetailsResponse{
				DocId:          42,
				Title:          "Meaning of Life Error",
				BestAnswerBody: "<p>Do not panic</p>",
				OriginalUrl:    "https://stackoverflow.com/questions/42",
			}, nil
		},
	}

	client, cleanup := setupBufconnGraphClient(t, mockSrv)
	defer cleanup()

	details, err := client.GetThreadDetails(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected GetThreadDetails error: %v", err)
	}

	if details.DocID != 42 {
		t.Errorf("expected doc_id 42, got %d", details.DocID)
	}
	if details.Title != "Meaning of Life Error" {
		t.Errorf("expected title 'Meaning of Life Error', got '%s'", details.Title)
	}
	if details.BestAnswerBody != "<p>Do not panic</p>" {
		t.Errorf("expected best_answer_body '<p>Do not panic</p>', got '%s'", details.BestAnswerBody)
	}
	if details.OriginalURL != "https://stackoverflow.com/questions/42" {
		t.Errorf("expected original_url 'https://stackoverflow.com/questions/42', got '%s'", details.OriginalURL)
	}
}

func TestGRPCClient_GetThreadDetails_NotFound(t *testing.T) {
	mockSrv := &mockGraphServer{
		getThreadDetailsFn: func(ctx context.Context, req *graphpb.ThreadDetailsRequest) (*graphpb.ThreadDetailsResponse, error) {
			return nil, status.Error(codes.NotFound, "thread not found in graph.db")
		},
	}

	client, cleanup := setupBufconnGraphClient(t, mockSrv)
	defer cleanup()

	details, err := client.GetThreadDetails(context.Background(), 99999)
	if err == nil {
		t.Fatalf("expected error, got details: %+v", details)
	}
	if !errors.Is(err, graph.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
