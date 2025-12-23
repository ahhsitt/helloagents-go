package rag_test

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/rag"
)

func TestVectorRetriever_RetrieveWithOptions(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	// Add test chunks
	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Machine learning basics", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", Content: "Deep learning with neural networks", Vector: []float32{0.9, 0.1, 0.0}},
		{ID: "chunk-3", Content: "Natural language processing", Vector: []float32{0.0, 1.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)

	// Test basic retrieve without options
	results, err := retriever.RetrieveWithOptions(ctx, "machine learning", 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestVectorRetriever_WithMQE(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", Content: "Content 2", Vector: []float32{0.9, 0.1, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "expanded query 1\nexpanded query 2", nil
		},
	}

	retriever := rag.NewVectorRetriever(store, embedder)
	results, err := retriever.RetrieveWithOptions(ctx, "query", 2,
		rag.WithMQE(llm, 2),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results with MQE")
	}
}

func TestVectorRetriever_WithHyDE(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "Hypothetical answer document about the topic.", nil
		},
	}

	retriever := rag.NewVectorRetriever(store, embedder)
	results, err := retriever.RetrieveWithOptions(ctx, "what is AI?", 1,
		rag.WithHyDE(llm),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results with HyDE")
	}
}

func TestVectorRetriever_WithRerank(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", Content: "Content 2", Vector: []float32{0.9, 0.1, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	reranker := &mockReranker{}

	retriever := rag.NewVectorRetriever(store, embedder)
	results, err := retriever.RetrieveWithOptions(ctx, "query", 2,
		rag.WithRerank(reranker),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestVectorRetriever_CombinedStrategies(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", Content: "Content 2", Vector: []float32{0.9, 0.1, 0.0}},
		{ID: "chunk-3", Content: "Content 3", Vector: []float32{0.0, 1.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "expanded query", nil
		},
	}
	reranker := &mockReranker{}

	retriever := rag.NewVectorRetriever(store, embedder)

	// Combine MQE + HyDE + Rerank + RRF Fusion
	results, err := retriever.RetrieveWithOptions(ctx, "query", 3,
		rag.WithMQE(llm, 2),
		rag.WithHyDE(llm),
		rag.WithRerank(reranker),
		rag.WithRRFFusion(60),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results with combined strategies")
	}
}

func TestVectorRetriever_WithTimeout(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)

	results, err := retriever.RetrieveWithOptions(ctx, "query", 1,
		rag.WithTimeout(5*time.Second),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestVectorRetriever_WithFetchMultiplier(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)

	// Just test that it doesn't error
	_, err := retriever.RetrieveWithOptions(ctx, "query", 5,
		rag.WithFetchMultiplier(3),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVectorRetriever_WithCustomTransformer(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "custom expanded", nil
		},
	}

	mqeTransformer := rag.NewMultiQueryTransformer(llm,
		rag.WithNumQueries(2),
		rag.WithIncludeOriginal(true),
	)

	retriever := rag.NewVectorRetriever(store, embedder)
	results, err := retriever.RetrieveWithOptions(ctx, "query", 1,
		rag.WithMQETransformer(mqeTransformer),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results with custom transformer")
	}
}

func TestVectorRetriever_ImplementsAdvancedRetriever(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)
	var _ rag.AdvancedRetriever = retriever
}

func TestRetrieveOptions_Defaults(t *testing.T) {
	ctx := context.Background()
	store := rag.NewInMemoryVectorStore()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Content 1", Vector: []float32{1.0, 0.0, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	embedder := newMockEmbedder()
	retriever := rag.NewVectorRetriever(store, embedder)

	// No options - should use defaults
	results, err := retriever.RetrieveWithOptions(ctx, "query", 1)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}
