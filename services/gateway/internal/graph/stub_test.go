package graph_test

import (
	"context"
	"errors"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/graph"
)

func TestGraphStub_CategoriesAndNodes(t *testing.T) {
	stub := graph.NewStub()
	ctx := context.Background()

	// Query with 2 known search fixtures (e.g. 27539308 and 15478423)
	docIDs := []int64{27539308, 15478423}

	tg, err := stub.GetThreadGraph(ctx, docIDs)
	if err != nil {
		t.Fatalf("unexpected graph error: %v", err)
	}

	if len(tg.Nodes) == 0 {
		t.Fatal("expected non-empty graph nodes")
	}

	// Verify categories
	categoryCounts := make(map[string]int)
	seenDocs := make(map[int64]bool)

	for i, n := range tg.Nodes {
		if seenDocs[n.DocID] {
			t.Errorf("duplicate node doc_id detected: %d", n.DocID)
		}
		seenDocs[n.DocID] = true
		categoryCounts[n.Category]++

		if i == 0 && n.Category != "PRIMARY_RESULT" {
			t.Errorf("first node should be PRIMARY_RESULT, got %s", n.Category)
		}
		if i == 1 && n.Category != "OTHER_RESULT" {
			t.Errorf("second node should be OTHER_RESULT, got %s", n.Category)
		}
	}

	if categoryCounts["PRIMARY_RESULT"] != 1 {
		t.Errorf("expected exactly 1 PRIMARY_RESULT, got %d", categoryCounts["PRIMARY_RESULT"])
	}
	if categoryCounts["OTHER_RESULT"] != 1 {
		t.Errorf("expected exactly 1 OTHER_RESULT, got %d", categoryCounts["OTHER_RESULT"])
	}
	if categoryCounts["DUPLICATE"] != 3 {
		t.Errorf("expected exactly 3 DUPLICATE nodes, got %d", categoryCounts["DUPLICATE"])
	}
	if categoryCounts["LINKED"] < 1 {
		t.Errorf("expected at least 1 LINKED node, got %d", categoryCounts["LINKED"])
	}
}

func TestGraphStub_EdgesNoSelfLoops(t *testing.T) {
	stub := graph.NewStub()
	ctx := context.Background()

	docIDs := []int64{27539308, 15478423}

	tg, err := stub.GetThreadGraph(ctx, docIDs)
	if err != nil {
		t.Fatalf("unexpected graph error: %v", err)
	}

	if len(tg.Edges) == 0 {
		t.Fatal("expected non-empty edges in graph")
	}

	nodeDocIDs := make(map[int64]bool)
	for _, n := range tg.Nodes {
		nodeDocIDs[n.DocID] = true
	}

	for _, e := range tg.Edges {
		if e.SourceDocID == e.TargetDocID {
			t.Errorf("self-loop detected: source == target (%d)", e.SourceDocID)
		}
		if !nodeDocIDs[e.SourceDocID] {
			t.Errorf("edge source doc %d not in graph nodes", e.SourceDocID)
		}
		if !nodeDocIDs[e.TargetDocID] {
			t.Errorf("edge target doc %d not in graph nodes", e.TargetDocID)
		}
		if e.Type != "DUPLICATE_OF" && e.Type != "LINKED_IN_ANSWER" {
			t.Errorf("unexpected edge type: %q", e.Type)
		}
	}
}

func TestGraphStub_Determinism(t *testing.T) {
	stub := graph.NewStub()
	ctx := context.Background()

	docIDs := []int64{27539308, 15478423}

	tg1, err1 := stub.GetThreadGraph(ctx, docIDs)
	if err1 != nil {
		t.Fatalf("error on first call: %v", err1)
	}

	tg2, err2 := stub.GetThreadGraph(ctx, docIDs)
	if err2 != nil {
		t.Fatalf("error on second call: %v", err2)
	}

	if len(tg1.Nodes) != len(tg2.Nodes) {
		t.Fatalf("nodes length mismatch: %d vs %d", len(tg1.Nodes), len(tg2.Nodes))
	}
	if len(tg1.Edges) != len(tg2.Edges) {
		t.Fatalf("edges length mismatch: %d vs %d", len(tg1.Edges), len(tg2.Edges))
	}

	for i := range tg1.Nodes {
		if tg1.Nodes[i].DocID != tg2.Nodes[i].DocID || tg1.Nodes[i].Category != tg2.Nodes[i].Category {
			t.Errorf("node[%d] mismatch", i)
		}
	}
	for i := range tg1.Edges {
		if tg1.Edges[i] != tg2.Edges[i] {
			t.Errorf("edge[%d] mismatch", i)
		}
	}
}

func TestGraphStub_GetThreadDetails(t *testing.T) {
	stub := graph.NewStub()
	ctx := context.Background()

	// 1. Known doc from search fixtures
	detail1, err1 := stub.GetThreadDetails(ctx, 27539308)
	if err1 != nil {
		t.Fatalf("unexpected error for search fixture: %v", err1)
	}
	if detail1.DocID != 27539308 {
		t.Errorf("expected doc_id 27539308, got %d", detail1.DocID)
	}
	if detail1.Title == "" || detail1.BestAnswerBody == "" || detail1.OriginalURL == "" {
		t.Error("expected non-empty fields in thread details")
	}

	// 2. Known doc from graph pool
	detail2, err2 := stub.GetThreadDetails(ctx, 1132941)
	if err2 != nil {
		t.Fatalf("unexpected error for graph pool thread: %v", err2)
	}
	if detail2.DocID != 1132941 {
		t.Errorf("expected doc_id 1132941, got %d", detail2.DocID)
	}

	// 3. Unknown doc
	_, errNotFound := stub.GetThreadDetails(ctx, 999999999999)
	if errNotFound == nil {
		t.Fatal("expected error for unknown doc_id, got nil")
	}
	if !errors.Is(errNotFound, graph.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", errNotFound)
	}
}

func TestGraphStub_EmptyOrUnknownDocIDs(t *testing.T) {
	stub := graph.NewStub()
	ctx := context.Background()

	// Empty docIDs
	tgEmpty, err := stub.GetThreadGraph(ctx, []int64{})
	if err != nil {
		t.Fatalf("unexpected error for empty docIDs: %v", err)
	}
	if len(tgEmpty.Nodes) != 0 || len(tgEmpty.Edges) != 0 {
		t.Errorf("expected empty graph, got %d nodes, %d edges", len(tgEmpty.Nodes), len(tgEmpty.Edges))
	}

	// Unknown docIDs
	tgUnknown, err := stub.GetThreadGraph(ctx, []int64{88888888, 99999999})
	if err != nil {
		t.Fatalf("unexpected error for unknown docIDs: %v", err)
	}
	if len(tgUnknown.Nodes) != 0 {
		t.Errorf("expected empty graph for unknown docIDs, got %d nodes", len(tgUnknown.Nodes))
	}
}
