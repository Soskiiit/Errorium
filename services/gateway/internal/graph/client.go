package graph

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested thread does not exist.
var ErrNotFound = errors.New("thread not found")

// GraphClient abstracts Star Map graph construction and thread details lookup.
type GraphClient interface {
	GetThreadGraph(ctx context.Context, docIDs []int64) (*ThreadGraph, error)
	GetThreadDetails(ctx context.Context, docID int64) (*ThreadDetails, error)
}

// ThreadNode represents a node in the Star Map graph.
type ThreadNode struct {
	DocID             int64    `json:"doc_id"`
	Category          string   `json:"category"`
	Score             float64  `json:"score"`
	Title             string   `json:"title"`
	Snippet           string   `json:"snippet"`
	HasAcceptedAnswer bool     `json:"has_accepted_answer"`
	Tags              []string `json:"tags"`
	BestAnswerScore   int      `json:"best_answer_score"`
	BestAnswerBody    string   `json:"best_answer_body"`
	OriginalURL       string   `json:"original_url"`
}

// ThreadEdge represents a directed relationship edge in the Star Map graph.
type ThreadEdge struct {
	SourceDocID int64  `json:"source_doc_id"`
	TargetDocID int64  `json:"target_doc_id"`
	Type        string `json:"type"`
}

// ThreadGraph represents the Star Map graph with nodes and edges.
type ThreadGraph struct {
	Nodes []ThreadNode `json:"nodes"`
	Edges []ThreadEdge `json:"edges"`
}

// ThreadDetails represents full details of a Stack Overflow thread.
type ThreadDetails struct {
	DocID             int64    `json:"doc_id"`
	Title             string   `json:"title"`
	BestAnswerBody    string   `json:"best_answer_body"`
	OriginalURL       string   `json:"original_url"`
	Tags              []string `json:"tags"`
	QuestionScore     int      `json:"question_score"`
	HasAcceptedAnswer bool     `json:"has_accepted_answer"`
}
