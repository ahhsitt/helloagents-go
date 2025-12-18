package rag_test

import (
	"context"
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

// mockEmbedder implements rag.Embedder for testing
type mockEmbedder struct {
	embedFn func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	// Default: return simple vectors
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{1.0, 0.0, 0.0}
	}
	return result, nil
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{}
}

func TestNewVectorRetriever(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	embedder := newMockEmbedder()

	retriever := rag.NewVectorRetriever(store, embedder)
	if retriever == nil {
		t.Fatal("expected non-nil retriever")
	}
}

func TestVectorRetriever_Retrieve(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	// Add chunks with vectors
	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Hello world", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", Content: "Goodbye world", Vector: []float32{0.0, 1.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)

	results, err := retriever.Retrieve(ctx, "hello", 2)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestVectorRetriever_WithScoreThreshold(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	// Add chunks with different similarity
	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Hello", Vector: []float32{1.0, 0.0, 0.0}},     // High match
		{ID: "chunk-2", Content: "World", Vector: []float32{0.0, 1.0, 0.0}},     // Low match
		{ID: "chunk-3", Content: "Similar", Vector: []float32{0.9, 0.1, 0.0}},   // Medium-high match
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder, rag.WithScoreThreshold(0.5))

	results, err := retriever.Retrieve(ctx, "hello", 5)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Only high scoring results should be returned
	for _, r := range results {
		if r.Score < 0.5 {
			t.Fatalf("expected score >= 0.5, got %f", r.Score)
		}
	}
}

func TestNewMultiRetriever(t *testing.T) {
	store1 := rag.NewInMemoryVectorStore()
	store2 := rag.NewInMemoryVectorStore()
	embedder := newMockEmbedder()

	retriever1 := rag.NewVectorRetriever(store1, embedder)
	retriever2 := rag.NewVectorRetriever(store2, embedder)

	multiRetriever := rag.NewMultiRetriever([]rag.Retriever{retriever1, retriever2}, nil)

	if multiRetriever == nil {
		t.Fatal("expected non-nil multi-retriever")
	}
}

func TestMultiRetriever_Retrieve(t *testing.T) {
	ctx := context.Background()

	store1 := rag.NewInMemoryVectorStore()
	store2 := rag.NewInMemoryVectorStore()

	// Add different chunks to each store
	_ = store1.Add(ctx, []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Store1 content", Vector: []float32{1.0, 0.0, 0.0}},
	})
	_ = store2.Add(ctx, []rag.DocumentChunk{
		{ID: "chunk-2", Content: "Store2 content", Vector: []float32{0.9, 0.1, 0.0}},
	})

	embedder := newMockEmbedder()
	retriever1 := rag.NewVectorRetriever(store1, embedder)
	retriever2 := rag.NewVectorRetriever(store2, embedder)

	multiRetriever := rag.NewMultiRetriever([]rag.Retriever{retriever1, retriever2}, nil)

	results, err := multiRetriever.Retrieve(ctx, "query", 5)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results from both stores, got %d", len(results))
	}
}

func TestScoreBasedMerger_Merge(t *testing.T) {
	merger := &rag.ScoreBasedMerger{}

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.9},
		},
		{
			{Chunk: rag.DocumentChunk{ID: "c"}, Score: 0.7},
		},
	}

	merged := merger.Merge(results, 2)

	if len(merged) != 2 {
		t.Fatalf("expected 2 results, got %d", len(merged))
	}
	// Should be sorted by score
	if merged[0].Score < merged[1].Score {
		t.Fatal("expected results sorted by score descending")
	}
}

func TestVectorRetriever_ImplementsRetriever(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)
	var _ rag.Retriever = retriever
}

func TestMultiRetriever_ImplementsRetriever(t *testing.T) {
	retriever := rag.NewMultiRetriever(nil, nil)
	var _ rag.Retriever = retriever
}
