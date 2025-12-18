package rag_test

import (
	"context"
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

func TestNewInMemoryVectorStore(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.Size() != 0 {
		t.Fatalf("expected size 0, got %d", store.Size())
	}
}

func TestInMemoryVectorStore_Add(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Hello"},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "World"},
	}

	err := store.Add(ctx, chunks)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.Size() != 2 {
		t.Fatalf("expected size 2, got %d", store.Size())
	}
}

func TestInMemoryVectorStore_Search(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	// Add chunks with vectors
	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Hello", Vector: []float32{1.0, 0.0, 0.0}},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "World", Vector: []float32{0.0, 1.0, 0.0}},
		{ID: "chunk-3", DocumentID: "doc-1", Content: "Foo", Vector: []float32{0.5, 0.5, 0.0}},
	}
	_ = store.Add(ctx, chunks)

	// Search with a query vector
	query := []float32{1.0, 0.0, 0.0}
	results, err := store.Search(ctx, query, 2)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// First result should be chunk-1 (exact match)
	if results[0].Chunk.ID != "chunk-1" {
		t.Fatalf("expected first result to be chunk-1, got %s", results[0].Chunk.ID)
	}
	if results[0].Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestInMemoryVectorStore_SearchEmptyStore(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 5)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestInMemoryVectorStore_Delete(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Hello"},
		{ID: "chunk-2", Content: "World"},
	}
	_ = store.Add(ctx, chunks)

	err := store.Delete(ctx, []string{"chunk-1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.Size() != 1 {
		t.Fatalf("expected size 1, got %d", store.Size())
	}
}

func TestInMemoryVectorStore_Clear(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	ctx := context.Background()

	chunks := []rag.DocumentChunk{
		{ID: "chunk-1", Content: "Hello"},
		{ID: "chunk-2", Content: "World"},
	}
	_ = store.Add(ctx, chunks)

	err := store.Clear(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.Size() != 0 {
		t.Fatalf("expected size 0, got %d", store.Size())
	}
}

func TestInMemoryVectorStore_ImplementsVectorStore(t *testing.T) {
	store := rag.NewInMemoryVectorStore()
	var _ rag.VectorStore = store
}
