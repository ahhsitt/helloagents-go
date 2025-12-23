package rag_test

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/rag"
)

func TestRRFFusion_Fuse(t *testing.T) {
	fusion := rag.NewRRFFusion(60)

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.9},
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.8},
			{Chunk: rag.DocumentChunk{ID: "c"}, Score: 0.7},
		},
		{
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.95},
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.85},
			{Chunk: rag.DocumentChunk{ID: "d"}, Score: 0.6},
		},
	}

	weights := []float32{1.0, 1.0}
	fused := fusion.Fuse(results, weights, 3)

	if len(fused) != 3 {
		t.Fatalf("expected 3 results, got %d", len(fused))
	}

	// Documents appearing in both lists should rank higher
	// 'a' and 'b' appear in both, should be at top
	topIDs := make(map[string]bool)
	for _, r := range fused[:2] {
		topIDs[r.Chunk.ID] = true
	}
	if !topIDs["a"] || !topIDs["b"] {
		t.Fatal("expected 'a' and 'b' to be in top 2 results due to RRF fusion")
	}
}

func TestRRFFusion_EmptyResults(t *testing.T) {
	fusion := rag.NewRRFFusion(60)

	fused := fusion.Fuse(nil, nil, 5)
	if fused != nil {
		t.Fatalf("expected nil for empty input, got %v", fused)
	}

	fused = fusion.Fuse([][]rag.RetrievalResult{}, nil, 5)
	if fused != nil {
		t.Fatalf("expected nil for empty slice, got %v", fused)
	}
}

func TestRRFFusion_SingleQuery(t *testing.T) {
	fusion := rag.NewRRFFusion(60)

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.9},
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.8},
		},
	}

	fused := fusion.Fuse(results, nil, 5)

	if len(fused) != 2 {
		t.Fatalf("expected 2 results, got %d", len(fused))
	}
}

func TestRRFFusion_WithWeights(t *testing.T) {
	fusion := rag.NewRRFFusion(60)

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.9},
		},
		{
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.9},
		},
	}

	// First query has higher weight
	weights := []float32{2.0, 1.0}
	fused := fusion.Fuse(results, weights, 2)

	if len(fused) != 2 {
		t.Fatalf("expected 2 results, got %d", len(fused))
	}

	// 'a' should rank higher due to higher weight
	if fused[0].Chunk.ID != "a" {
		t.Fatal("expected 'a' to rank first due to higher weight")
	}
}

func TestRRFFusion_DefaultK(t *testing.T) {
	// Test that k=0 defaults to 60
	fusion := rag.NewRRFFusion(0)
	if fusion.K != 60 {
		t.Fatalf("expected K=60 for default, got %d", fusion.K)
	}

	fusion = rag.NewRRFFusion(-1)
	if fusion.K != 60 {
		t.Fatalf("expected K=60 for negative input, got %d", fusion.K)
	}
}

func TestScoreBasedFusion_Fuse(t *testing.T) {
	fusion := rag.NewScoreBasedFusion()

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.9},
		},
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.8}, // Same doc, higher score
			{Chunk: rag.DocumentChunk{ID: "c"}, Score: 0.7},
		},
	}

	fused := fusion.Fuse(results, nil, 3)

	if len(fused) != 3 {
		t.Fatalf("expected 3 results, got %d", len(fused))
	}

	// Should be sorted by score descending
	for i := 1; i < len(fused); i++ {
		if fused[i].Score > fused[i-1].Score {
			t.Fatal("expected results sorted by score descending")
		}
	}

	// Document 'a' should have score 0.8 (max score)
	for _, r := range fused {
		if r.Chunk.ID == "a" && r.Score != 0.8 {
			t.Fatalf("expected 'a' to have max score 0.8, got %f", r.Score)
		}
	}
}

func TestScoreBasedFusion_EmptyResults(t *testing.T) {
	fusion := rag.NewScoreBasedFusion()

	fused := fusion.Fuse(nil, nil, 5)
	if fused != nil {
		t.Fatalf("expected nil for empty input, got %v", fused)
	}
}

func TestScoreBasedFusion_WithWeights(t *testing.T) {
	fusion := rag.NewScoreBasedFusion()

	results := [][]rag.RetrievalResult{
		{
			{Chunk: rag.DocumentChunk{ID: "a"}, Score: 0.5},
		},
		{
			{Chunk: rag.DocumentChunk{ID: "b"}, Score: 0.5},
		},
	}

	// First query has higher weight
	weights := []float32{2.0, 1.0}
	fused := fusion.Fuse(results, weights, 2)

	// 'a' should have weighted score 1.0, 'b' should have 0.5
	if fused[0].Chunk.ID != "a" {
		t.Fatal("expected 'a' to rank first due to higher weight")
	}
	if fused[0].Score != 1.0 {
		t.Fatalf("expected weighted score 1.0, got %f", fused[0].Score)
	}
}

func TestFusionStrategy_Interface(t *testing.T) {
	var _ rag.FusionStrategy = rag.NewRRFFusion(60)
	var _ rag.FusionStrategy = rag.NewScoreBasedFusion()
}
