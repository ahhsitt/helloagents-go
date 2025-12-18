package rag_test

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

func TestDocument(t *testing.T) {
	doc := rag.Document{
		ID:      "doc-1",
		Content: "Test content",
		Metadata: rag.DocumentMetadata{
			Source: "test.txt",
			Title:  "Test Document",
		},
	}

	if doc.ID != "doc-1" {
		t.Fatalf("expected ID 'doc-1', got %s", doc.ID)
	}
	if doc.Content != "Test content" {
		t.Fatalf("expected Content 'Test content', got %s", doc.Content)
	}
	if doc.Metadata.Source != "test.txt" {
		t.Fatalf("expected Source 'test.txt', got %s", doc.Metadata.Source)
	}
}

func TestDocumentChunk(t *testing.T) {
	chunk := rag.DocumentChunk{
		ID:          "chunk-1",
		DocumentID:  "doc-1",
		Content:     "Chunk content",
		Index:       0,
		StartOffset: 0,
		EndOffset:   13,
	}

	if chunk.ID != "chunk-1" {
		t.Fatalf("expected ID 'chunk-1', got %s", chunk.ID)
	}
	if chunk.DocumentID != "doc-1" {
		t.Fatalf("expected DocumentID 'doc-1', got %s", chunk.DocumentID)
	}
	if chunk.Index != 0 {
		t.Fatalf("expected Index 0, got %d", chunk.Index)
	}
}

func TestRetrievalResult(t *testing.T) {
	result := rag.RetrievalResult{
		Chunk: rag.DocumentChunk{
			ID:      "chunk-1",
			Content: "Test content",
		},
		Score: 0.95,
	}

	if result.Score != 0.95 {
		t.Fatalf("expected Score 0.95, got %f", result.Score)
	}
	if result.Chunk.ID != "chunk-1" {
		t.Fatalf("expected chunk ID 'chunk-1', got %s", result.Chunk.ID)
	}
}

func TestRAGContext_FormatContext_Empty(t *testing.T) {
	ctx := &rag.RAGContext{
		Query:   "test query",
		Results: []rag.RetrievalResult{},
	}

	formatted := ctx.FormatContext()
	if formatted != "" {
		t.Fatalf("expected empty string for no results, got %s", formatted)
	}
}

func TestRAGContext_FormatContext_WithResults(t *testing.T) {
	ctx := &rag.RAGContext{
		Query: "test query",
		Results: []rag.RetrievalResult{
			{
				Chunk: rag.DocumentChunk{
					Content: "First chunk content",
					Metadata: rag.DocumentMetadata{
						Source: "source1.txt",
					},
				},
				Score: 0.9,
			},
			{
				Chunk: rag.DocumentChunk{
					Content: "Second chunk content",
				},
				Score: 0.8,
			},
		},
	}

	formatted := ctx.FormatContext()

	if formatted == "" {
		t.Fatal("expected non-empty formatted context")
	}
	if len(formatted) < 50 {
		t.Fatalf("expected longer formatted context, got length %d", len(formatted))
	}
}

func TestDocumentMetadata_CustomFields(t *testing.T) {
	metadata := rag.DocumentMetadata{
		Source: "test.txt",
		Custom: map[string]interface{}{
			"category": "science",
			"priority": 1,
		},
	}

	if metadata.Custom["category"] != "science" {
		t.Fatalf("expected category 'science', got %v", metadata.Custom["category"])
	}
	if metadata.Custom["priority"] != 1 {
		t.Fatalf("expected priority 1, got %v", metadata.Custom["priority"])
	}
}

func TestDocumentMetadata_Tags(t *testing.T) {
	metadata := rag.DocumentMetadata{
		Tags: []string{"important", "reviewed"},
	}

	if len(metadata.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(metadata.Tags))
	}
	if metadata.Tags[0] != "important" {
		t.Fatalf("expected first tag 'important', got %s", metadata.Tags[0])
	}
}
