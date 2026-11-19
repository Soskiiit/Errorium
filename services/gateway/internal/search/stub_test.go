package search_test

import (
	"context"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

func TestSearchStub_Determinism(t *testing.T) {
	stub := search.NewStub()
	ctx := context.Background()

	req := search.SearchRequest{
		Query: "invalid syntax python",
		TopK:  5,
	}

	resp1, err1 := stub.Search(ctx, req)
	if err1 != nil {
		t.Fatalf("unexpected error on first search: %v", err1)
	}

	resp2, err2 := stub.Search(ctx, req)
	if err2 != nil {
		t.Fatalf("unexpected error on second search: %v", err2)
	}

	if resp1.TotalHits != resp2.TotalHits {
		t.Errorf("expected total_hits %d == %d", resp1.TotalHits, resp2.TotalHits)
	}
	if len(resp1.Results) != len(resp2.Results) {
		t.Fatalf("expected len(results) %d == %d", len(resp1.Results), len(resp2.Results))
	}

	for i := range resp1.Results {
		r1 := resp1.Results[i]
		r2 := resp2.Results[i]
		if r1.DocID != r2.DocID || r1.BM25Score != r2.BM25Score || r1.Title != r2.Title {
			t.Errorf("result[%d] mismatch: %+v vs %+v", i, r1, r2)
		}
	}
}

func TestSearchStub_ScoreOrdering(t *testing.T) {
	stub := search.NewStub()
	ctx := context.Background()

	req := search.SearchRequest{
		Query: "pointer exception error",
		TopK:  10,
	}

	resp, err := stub.Search(ctx, req)
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if len(resp.Results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(resp.Results))
	}

	for i := 0; i < len(resp.Results)-1; i++ {
		if resp.Results[i].BM25Score < resp.Results[i+1].BM25Score {
			t.Errorf("results not sorted by bm25_score DESC: [%d]=%f < [%d]=%f",
				i, resp.Results[i].BM25Score, i+1, resp.Results[i+1].BM25Score)
		}
	}
}

func TestSearchStub_TopKAndClamping(t *testing.T) {
	stub := search.NewStub()
	ctx := context.Background()

	// TopK = 2
	resp2, err := stub.Search(ctx, search.SearchRequest{
		Query: "python",
		TopK:  2,
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(resp2.Results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(resp2.Results))
	}
	if resp2.TotalHits < int64(len(resp2.Results)) {
		t.Errorf("total_hits %d should be >= results len %d", resp2.TotalHits, len(resp2.Results))
	}

	// Default TopK when <= 0
	respDef, err := stub.Search(ctx, search.SearchRequest{
		Query: "python",
		TopK:  0,
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(respDef.Results) > 10 {
		t.Errorf("expected default top_k=10, got %d results", len(respDef.Results))
	}
}

func TestSearchStub_TagFiltering(t *testing.T) {
	stub := search.NewStub()
	ctx := context.Background()

	// Filter by tag "go"
	respGo, err := stub.Search(ctx, search.SearchRequest{
		Query: "pointer",
		Tags:  []string{"go"},
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	for _, r := range respGo.Results {
		hasGo := false
		for _, tag := range r.Tags {
			if tag == "go" {
				hasGo = true
				break
			}
		}
		if !hasGo {
			t.Errorf("result %d (%s) does not contain requested tag 'go'", r.DocID, r.Title)
		}
	}

	// Non-matching tag
	respNone, err := stub.Search(ctx, search.SearchRequest{
		Query: "syntax",
		Tags:  []string{"nonexistent-tag-xyz"},
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(respNone.Results) != 0 || respNone.TotalHits != 0 {
		t.Errorf("expected 0 results for nonexistent tag, got %d hits", respNone.TotalHits)
	}
}

func TestSearchStub_RAG(t *testing.T) {
	stub := search.NewStub()
	ctx := context.Background()

	// 1. Successful RAG with hits
	ragResp, err := stub.RAG(ctx, search.SearchRequest{
		Query: "invalid syntax python",
		TopK:  3,
	})
	if err != nil {
		t.Fatalf("unexpected RAG error: %v", err)
	}

	if ragResp.Synthesis == "" {
		t.Error("expected non-empty synthesis")
	}
	if ragResp.Code == nil || *ragResp.Code == "" {
		t.Error("expected non-empty code snippet in RAG result")
	}
	if len(ragResp.Sources) == 0 {
		t.Error("expected non-empty sources in RAG result")
	}

	// 2. Determinism of RAG
	ragResp2, err := stub.RAG(ctx, search.SearchRequest{
		Query: "invalid syntax python",
		TopK:  3,
	})
	if err != nil {
		t.Fatalf("unexpected RAG error on second call: %v", err)
	}
	if ragResp.Synthesis != ragResp2.Synthesis {
		t.Errorf("RAG synthesis not deterministic")
	}
	if *ragResp.Code != *ragResp2.Code {
		t.Errorf("RAG code not deterministic")
	}

	// 3. Fallback when no hits match
	ragFallback, err := stub.RAG(ctx, search.SearchRequest{
		Query: "completelyunrelatedgibberish1234567890",
	})
	if err != nil {
		t.Fatalf("unexpected RAG fallback error: %v", err)
	}
	if ragFallback.Code != nil {
		t.Errorf("expected nil code snippet for non-matching query, got %v", *ragFallback.Code)
	}
	if len(ragFallback.Sources) != 0 {
		t.Errorf("expected empty sources for non-matching query, got %d", len(ragFallback.Sources))
	}
	if ragFallback.Synthesis == "" {
		t.Error("expected non-empty fallback synthesis message")
	}
}
