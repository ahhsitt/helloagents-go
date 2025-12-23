package rag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

// mockReranker implements rag.Reranker for testing
type mockReranker struct {
	rerankFn func(ctx context.Context, query string, results []rag.RetrievalResult) ([]rag.RetrievalResult, error)
}

func (m *mockReranker) Rerank(ctx context.Context, query string, results []rag.RetrievalResult) ([]rag.RetrievalResult, error) {
	if m.rerankFn != nil {
		return m.rerankFn(ctx, query, results)
	}
	// Default: reverse order with new scores
	reranked := make([]rag.RetrievalResult, len(results))
	for i, r := range results {
		reranked[len(results)-1-i] = r
		reranked[len(results)-1-i].Score = float32(len(results)-i) * 0.1
	}
	return reranked, nil
}

func TestRerankPostProcessor_Process(t *testing.T) {
	ctx := context.Background()

	reranker := &mockReranker{}
	processor := rag.NewRerankPostProcessor(reranker)

	results := []rag.RetrievalResult{
		{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
		{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.8},
		{Chunk: rag.DocumentChunk{ID: "c"}, Score: 0.6},
	}

	processed, err := processor.Process(ctx, "test query", results)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(processed) != 3 {
		t.Fatalf("expected 3 results, got %d", len(processed))
	}

	// Default mock reverses order
	if processed[0].Chunk.ID != "c" {
		t.Fatalf("expected 'c' first after reranking, got %s", processed[0].Chunk.ID)
	}
}

func TestRerankPostProcessor_EmptyResults(t *testing.T) {
	ctx := context.Background()

	reranker := &mockReranker{}
	processor := rag.NewRerankPostProcessor(reranker)

	processed, err := processor.Process(ctx, "query", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(processed) != 0 {
		t.Fatalf("expected empty results, got %d", len(processed))
	}
}

func TestRerankPostProcessor_RerankerError(t *testing.T) {
	ctx := context.Background()

	reranker := &mockReranker{
		rerankFn: func(ctx context.Context, query string, results []rag.RetrievalResult) ([]rag.RetrievalResult, error) {
			return nil, errors.New("rerank error")
		},
	}
	processor := rag.NewRerankPostProcessor(reranker)

	results := []rag.RetrievalResult{
		{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
	}

	_, err := processor.Process(ctx, "query", results)

	if err == nil {
		t.Fatal("expected error from reranker")
	}
}

func TestRerankPostProcessor_CustomReranking(t *testing.T) {
	ctx := context.Background()

	// Custom reranker that doubles scores
	reranker := &mockReranker{
		rerankFn: func(ctx context.Context, query string, results []rag.RetrievalResult) ([]rag.RetrievalResult, error) {
			reranked := make([]rag.RetrievalResult, len(results))
			for i, r := range results {
				reranked[i] = r
				reranked[i].Score = r.Score * 2
			}
			return reranked, nil
		},
	}
	processor := rag.NewRerankPostProcessor(reranker)

	results := []rag.RetrievalResult{
		{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
	}

	processed, err := processor.Process(ctx, "query", results)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if processed[0].Score != 1.0 {
		t.Fatalf("expected doubled score 1.0, got %f", processed[0].Score)
	}
}

func TestPostProcessor_Interface(t *testing.T) {
	reranker := &mockReranker{}
	var _ rag.PostProcessor = rag.NewRerankPostProcessor(reranker)
}
