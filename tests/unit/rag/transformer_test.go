package rag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

// mockLLMProvider implements rag.LLMProvider for testing
type mockLLMProvider struct {
	generateFn func(ctx context.Context, prompt string) (string, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, prompt)
	}
	return "", nil
}

func TestMultiQueryTransformer_Transform(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "query variant 1\nquery variant 2\nquery variant 3", nil
		},
	}

	transformer := rag.NewMultiQueryTransformer(llm, rag.WithNumQueries(3))
	queries, err := transformer.Transform(ctx, "original query")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should include original (default) + 3 expanded queries
	if len(queries) < 3 {
		t.Fatalf("expected at least 3 queries, got %d", len(queries))
	}
}

func TestMultiQueryTransformer_IncludeOriginal(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "expanded query", nil
		},
	}

	// Test with IncludeOriginal=true (default)
	transformer := rag.NewMultiQueryTransformer(llm, rag.WithIncludeOriginal(true))
	queries, _ := transformer.Transform(ctx, "original")

	hasOriginal := false
	for _, q := range queries {
		if q.Query == "original" {
			hasOriginal = true
			break
		}
	}
	if !hasOriginal {
		t.Fatal("expected original query to be included")
	}

	// Test with IncludeOriginal=false
	transformer = rag.NewMultiQueryTransformer(llm, rag.WithIncludeOriginal(false))
	queries, _ = transformer.Transform(ctx, "original")

	for _, q := range queries {
		if q.Query == "original" {
			t.Fatal("expected original query to NOT be included")
		}
	}
}

func TestMultiQueryTransformer_LLMError(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("llm error")
		},
	}

	// With IncludeOriginal=true, should fallback to original query
	transformer := rag.NewMultiQueryTransformer(llm, rag.WithIncludeOriginal(true))
	queries, err := transformer.Transform(ctx, "original")

	if err != nil {
		t.Fatal("expected no error when fallback is available")
	}
	if len(queries) != 1 || queries[0].Query != "original" {
		t.Fatal("expected fallback to original query")
	}

	// With IncludeOriginal=false, should return error
	transformer = rag.NewMultiQueryTransformer(llm, rag.WithIncludeOriginal(false))
	_, err = transformer.Transform(ctx, "original")

	if err == nil {
		t.Fatal("expected error when no fallback available")
	}
}

func TestHyDETransformer_Transform(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "This is a hypothetical document that answers the query.", nil
		},
	}

	transformer := rag.NewHyDETransformer(llm)
	queries, err := transformer.Transform(ctx, "what is machine learning?")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	if queries[0].Query == "what is machine learning?" {
		t.Fatal("expected hypothetical document, got original query")
	}
}

func TestHyDETransformer_LLMError(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("llm error")
		},
	}

	transformer := rag.NewHyDETransformer(llm)
	queries, err := transformer.Transform(ctx, "original query")

	// HyDE should fallback gracefully
	if err != nil {
		t.Fatalf("expected no error on fallback, got %v", err)
	}

	if len(queries) != 1 || queries[0].Query != "original query" {
		t.Fatal("expected fallback to original query")
	}
}

func TestHyDETransformer_EmptyResponse(t *testing.T) {
	ctx := context.Background()

	llm := &mockLLMProvider{
		generateFn: func(ctx context.Context, prompt string) (string, error) {
			return "   \n\t  ", nil // Empty/whitespace response
		},
	}

	transformer := rag.NewHyDETransformer(llm)
	queries, err := transformer.Transform(ctx, "original query")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should fallback to original query
	if len(queries) != 1 || queries[0].Query != "original query" {
		t.Fatal("expected fallback to original query for empty response")
	}
}

func TestHyDETransformer_WithOptions(t *testing.T) {
	llm := &mockLLMProvider{}

	// Test WithHyDEMaxTokens
	transformer := rag.NewHyDETransformer(llm, rag.WithHyDEMaxTokens(256))
	if transformer == nil {
		t.Fatal("expected non-nil transformer")
	}

	// Test WithHyDEPrompt
	transformer = rag.NewHyDETransformer(llm, rag.WithHyDEPrompt("custom prompt: %s"))
	if transformer == nil {
		t.Fatal("expected non-nil transformer")
	}
}

func TestTransformedQuery_WithWeight(t *testing.T) {
	q := rag.NewTransformedQuery("test query")

	if q.Weight != 1.0 {
		t.Fatalf("expected default weight 1.0, got %f", q.Weight)
	}

	q = q.WithWeight(0.5)
	if q.Weight != 0.5 {
		t.Fatalf("expected weight 0.5, got %f", q.Weight)
	}
}

func TestTransformedQuery_WithMetadata(t *testing.T) {
	q := rag.NewTransformedQuery("test query")

	if q.Metadata != nil {
		t.Fatal("expected nil metadata by default")
	}

	q = q.WithMetadata("source", "mqe")
	if q.Metadata == nil || q.Metadata["source"] != "mqe" {
		t.Fatal("expected metadata to be set")
	}
}

func TestQueryTransformer_Interface(t *testing.T) {
	llm := &mockLLMProvider{}
	var _ rag.QueryTransformer = rag.NewMultiQueryTransformer(llm)
	var _ rag.QueryTransformer = rag.NewHyDETransformer(llm)
}
