package memory_test

import (
	"context"
	"testing"

	"github.com/easyops/helloagents-go/pkg/memory"
)

// mockEmbedder implements memory.Embedder for testing
type mockEmbedder struct {
	embedFn func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	// Return simple embeddings - each text gets a unique vector
	result := make([][]float32, len(texts))
	for i := range texts {
		vec := make([]float32, 128)
		// Simple hash-based embedding for testing
		for j, c := range texts[i] {
			if j < 128 {
				vec[j] = float32(c) / 256.0
			}
		}
		result[i] = vec
	}
	return result, nil
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{}
}

func TestNewSemanticMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	if mem == nil {
		t.Fatal("expected non-nil memory")
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestSemanticMemory_Store(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Store(ctx, "id-1", "Hello world", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_StoreWithAutoID(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Store(ctx, "", "Hello world", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_StoreWithMetadata(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	metadata := map[string]interface{}{
		"source": "test",
		"type":   "preference",
	}

	err := mem.Store(ctx, "id-1", "User preference", metadata)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSemanticMemory_Search(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Goodbye world", nil)
	_ = mem.Store(ctx, "id-3", "Hello there", nil)

	results, err := mem.Search(ctx, "Hello", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestSemanticMemory_SearchWithThreshold(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Something completely different", nil)

	results, err := mem.SearchWithThreshold(ctx, "Hello", 5, 0.9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Only high-scoring results should be returned
	for _, r := range results {
		if r.Score < 0.9 {
			t.Fatalf("expected score >= 0.9, got %f", r.Score)
		}
	}
}

func TestSemanticMemory_Delete(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello world", nil)
	_ = mem.Store(ctx, "id-2", "Goodbye world", nil)

	err := mem.Delete(ctx, "id-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestSemanticMemory_DeleteNotFound(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	err := mem.Delete(ctx, "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestSemanticMemory_Clear(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Hello", nil)
	_ = mem.Store(ctx, "id-2", "World", nil)

	err := mem.Clear(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestSemanticMemory_Update(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	_ = mem.Store(ctx, "id-1", "Original content", nil)
	_ = mem.Store(ctx, "id-1", "Updated content", nil)

	if mem.Size() != 1 {
		t.Fatalf("expected size 1 after update, got %d", mem.Size())
	}

	results, _ := mem.Search(ctx, "Updated", 1)
	if len(results) == 0 {
		t.Fatal("expected to find updated content")
	}
}

func TestSemanticMemory_SearchResultFields(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	ctx := context.Background()

	metadata := map[string]interface{}{"key": "value"}
	_ = mem.Store(ctx, "id-1", "Test content", metadata)

	results, _ := mem.Search(ctx, "Test", 1)
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}

	r := results[0]
	if r.ID != "id-1" {
		t.Fatalf("expected ID 'id-1', got %s", r.ID)
	}
	if r.Content != "Test content" {
		t.Fatalf("expected content 'Test content', got %s", r.Content)
	}
	if r.Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestSemanticMemory_ImplementsVectorMemory(t *testing.T) {
	embedder := newMockEmbedder()
	mem := memory.NewSemanticMemory(embedder)
	var _ memory.VectorMemory = mem
}
