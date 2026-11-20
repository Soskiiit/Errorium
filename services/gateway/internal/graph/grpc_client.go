package graph

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	graphpb "github.com/soskiiit/errorium/services/gateway/internal/pb/graph"
)

// GRPCClient implements GraphClient by calling search_engine_core GraphService via gRPC.
type GRPCClient struct {
	client graphpb.GraphServiceClient
}

// NewGRPCClient constructs a new gRPC GraphClient from an existing gRPC connection.
func NewGRPCClient(conn *grpc.ClientConn) *GRPCClient {
	return &GRPCClient{
		client: graphpb.NewGraphServiceClient(conn),
	}
}

func mapNodeCategory(cat graphpb.NodeCategory) string {
	switch cat {
	case graphpb.NodeCategory_NODE_CATEGORY_PRIMARY_RESULT:
		return "PRIMARY_RESULT"
	case graphpb.NodeCategory_NODE_CATEGORY_OTHER_RESULT:
		return "OTHER_RESULT"
	case graphpb.NodeCategory_NODE_CATEGORY_DUPLICATE:
		return "DUPLICATE"
	case graphpb.NodeCategory_NODE_CATEGORY_LINKED:
		return "LINKED"
	default:
		return "OTHER_RESULT"
	}
}

func mapEdgeType(edgeType graphpb.EdgeType) string {
	switch edgeType {
	case graphpb.EdgeType_EDGE_TYPE_DUPLICATE_OF:
		return "DUPLICATE_OF"
	case graphpb.EdgeType_EDGE_TYPE_LINKED_IN_ANSWER:
		return "LINKED_IN_ANSWER"
	default:
		return "LINKED_IN_ANSWER"
	}
}

// GetThreadGraph queries the core graph service and enriches nodes with details sequentially.
func (c *GRPCClient) GetThreadGraph(ctx context.Context, docIDs []int64) (*ThreadGraph, error) {
	resp, err := c.client.GetThreadGraph(ctx, &graphpb.ThreadGraphRequest{
		DocIds: docIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("core graph RPC failed: %w", err)
	}

	protoNodes := resp.GetNodes()
	nodes := make([]ThreadNode, 0, len(protoNodes))

	// Sequentially enrich each node with best_answer_body and original_url (≤ 21 calls)
	for _, pn := range protoNodes {
		docID := pn.GetDocId()
		origURL := fmt.Sprintf("https://stackoverflow.com/questions/%d", docID)
		bestAnswerBody := ""

		details, err := c.client.GetThreadDetails(ctx, &graphpb.ThreadDetailsRequest{
			DocId: docID,
		})
		if err == nil && details != nil {
			bestAnswerBody = details.GetBestAnswerBody()
			if details.GetOriginalUrl() != "" {
				origURL = details.GetOriginalUrl()
			}
		}

		nodes = append(nodes, ThreadNode{
			DocID:             docID,
			Category:          mapNodeCategory(pn.GetCategory()),
			Score:             pn.GetScore(),
			Title:             pn.GetTitle(),
			Snippet:           pn.GetSnippet(),
			HasAcceptedAnswer: pn.GetHasAcceptedAnswer(),
			Tags:              pn.GetTags(),
			BestAnswerScore:   int(pn.GetBestAnswerScore()),
			BestAnswerBody:    bestAnswerBody,
			OriginalURL:       origURL,
		})
	}

	protoEdges := resp.GetEdges()
	edges := make([]ThreadEdge, 0, len(protoEdges))
	for _, pe := range protoEdges {
		edges = append(edges, ThreadEdge{
			SourceDocID: pe.GetSourceDocId(),
			TargetDocID: pe.GetTargetDocId(),
			Type:        mapEdgeType(pe.GetType()),
		})
	}

	return &ThreadGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetThreadDetails queries details for a single thread and maps NotFound to ErrNotFound.
func (c *GRPCClient) GetThreadDetails(ctx context.Context, docID int64) (*ThreadDetails, error) {
	resp, err := c.client.GetThreadDetails(ctx, &graphpb.ThreadDetailsRequest{
		DocId: docID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("core graph details RPC failed: %w", err)
	}

	origURL := resp.GetOriginalUrl()
	if origURL == "" {
		origURL = fmt.Sprintf("https://stackoverflow.com/questions/%d", docID)
	}

	return &ThreadDetails{
		DocID:          resp.GetDocId(),
		Title:          resp.GetTitle(),
		BestAnswerBody: resp.GetBestAnswerBody(),
		OriginalURL:    origURL,
	}, nil
}
