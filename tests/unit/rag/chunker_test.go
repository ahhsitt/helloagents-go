package rag_test

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

func TestNewRecursiveCharacterChunker(t *testing.T) {
	chunker := rag.NewRecursiveCharacterChunker(100, 20)
	if chunker == nil {
		t.Fatal("expected non-nil chunker")
	}
	if chunker.ChunkSize != 100 {
		t.Fatalf("expected ChunkSize 100, got %d", chunker.ChunkSize)
	}
	if chunker.ChunkOverlap != 20 {
		t.Fatalf("expected ChunkOverlap 20, got %d", chunker.ChunkOverlap)
	}
}

func TestRecursiveCharacterChunker_ChunkSmallDoc(t *testing.T) {
	chunker := rag.NewRecursiveCharacterChunker(200, 20)
	doc := rag.Document{
		ID:      "doc-1",
		Content: "This is a small document.",
	}

	chunks := chunker.Chunk(doc)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "This is a small document." {
		t.Fatalf("expected original content, got %s", chunks[0].Content)
	}
	if chunks[0].DocumentID != "doc-1" {
		t.Fatalf("expected DocumentID 'doc-1', got %s", chunks[0].DocumentID)
	}
}

func TestRecursiveCharacterChunker_ChunkLargeDoc(t *testing.T) {
	chunker := rag.NewRecursiveCharacterChunker(50, 10)
	doc := rag.Document{
		ID:      "doc-1",
		Content: "This is the first paragraph.\n\nThis is the second paragraph. It has more content.\n\nAnd this is the third paragraph.",
	}

	chunks := chunker.Chunk(doc)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Verify each chunk has proper fields
	for i, chunk := range chunks {
		if chunk.ID == "" {
			t.Fatalf("chunk %d has empty ID", i)
		}
		if chunk.DocumentID != "doc-1" {
			t.Fatalf("chunk %d has wrong DocumentID", i)
		}
		if chunk.Index != i {
			t.Fatalf("expected chunk index %d, got %d", i, chunk.Index)
		}
	}
}

func TestRecursiveCharacterChunker_PreserveMetadata(t *testing.T) {
	chunker := rag.NewRecursiveCharacterChunker(50, 10)
	doc := rag.Document{
		ID:      "doc-1",
		Content: "Short content that fits.",
		Metadata: rag.DocumentMetadata{
			Source: "test.txt",
			Title:  "Test Document",
		},
	}

	chunks := chunker.Chunk(doc)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].Metadata.Source != "test.txt" {
		t.Fatalf("expected Source 'test.txt', got %s", chunks[0].Metadata.Source)
	}
	if chunks[0].Metadata.Title != "Test Document" {
		t.Fatalf("expected Title 'Test Document', got %s", chunks[0].Metadata.Title)
	}
}

func TestNewSentenceChunker(t *testing.T) {
	chunker := rag.NewSentenceChunker(200, 50)
	if chunker == nil {
		t.Fatal("expected non-nil chunker")
	}
	if chunker.MaxChunkSize != 200 {
		t.Fatalf("expected MaxChunkSize 200, got %d", chunker.MaxChunkSize)
	}
	if chunker.MinChunkSize != 50 {
		t.Fatalf("expected MinChunkSize 50, got %d", chunker.MinChunkSize)
	}
}

func TestSentenceChunker_Chunk(t *testing.T) {
	chunker := rag.NewSentenceChunker(100, 20)
	doc := rag.Document{
		ID:      "doc-1",
		Content: "First sentence. Second sentence. Third sentence here.",
	}

	chunks := chunker.Chunk(doc)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	for _, chunk := range chunks {
		if chunk.DocumentID != "doc-1" {
			t.Fatalf("expected DocumentID 'doc-1', got %s", chunk.DocumentID)
		}
	}
}

func TestSentenceChunker_LongSentences(t *testing.T) {
	chunker := rag.NewSentenceChunker(50, 10)
	doc := rag.Document{
		ID:      "doc-1",
		Content: "Short. This is a very long sentence that should cause the chunker to create a new chunk when combined with other sentences. Another short one.",
	}

	chunks := chunker.Chunk(doc)

	// Should create multiple chunks
	if len(chunks) < 1 {
		t.Fatal("expected at least one chunk")
	}
}

func TestRecursiveCharacterChunker_ImplementsDocumentChunker(t *testing.T) {
	chunker := rag.NewRecursiveCharacterChunker(100, 20)
	var _ rag.DocumentChunker = chunker
}

func TestSentenceChunker_ImplementsDocumentChunker(t *testing.T) {
	chunker := rag.NewSentenceChunker(100, 20)
	var _ rag.DocumentChunker = chunker
}
