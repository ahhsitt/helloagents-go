package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/memory"
)

func TestNewEpisodicMemory(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	if mem == nil {
		t.Fatal("expected non-nil memory")
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestEpisodicMemory_AddEpisode(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	ep := memory.Episode{
		Type:       "user_action",
		Content:    "User asked about weather",
		Importance: 0.5,
	}

	err := mem.AddEpisode(ctx, ep)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestEpisodicMemory_AddEpisodeWithAutoID(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	ep := memory.Episode{
		Type:    "test",
		Content: "Test content",
	}

	_ = mem.AddEpisode(ctx, ep)
	episodes, _ := mem.GetEpisodes(ctx, nil)

	if episodes[0].ID == "" {
		t.Fatal("expected auto-generated ID")
	}
}

func TestEpisodicMemory_AddEpisodeWithAutoTimestamp(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	ep := memory.Episode{
		Type:    "test",
		Content: "Test content",
	}

	_ = mem.AddEpisode(ctx, ep)
	episodes, _ := mem.GetEpisodes(ctx, nil)

	if episodes[0].Timestamp == 0 {
		t.Fatal("expected auto-generated timestamp")
	}
}

func TestEpisodicMemory_GetEpisodes(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "1"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "2"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "c", Content: "3"})

	episodes, err := mem.GetEpisodes(ctx, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(episodes) != 3 {
		t.Fatalf("expected 3 episodes, got %d", len(episodes))
	}
}

func TestEpisodicMemory_GetEpisodesByType(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "1"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "error", Content: "2"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "3"})

	episodes, _ := mem.GetByType(ctx, "action", 10)
	if len(episodes) != 2 {
		t.Fatalf("expected 2 action episodes, got %d", len(episodes))
	}
}

func TestEpisodicMemory_GetEpisodesWithFilter(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "1", Importance: 0.3})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "2", Importance: 0.8})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "error", Content: "3", Importance: 0.9})

	filter := &memory.EpisodeFilter{
		Types:         []string{"action"},
		MinImportance: 0.5,
	}

	episodes, _ := mem.GetEpisodes(ctx, filter)
	if len(episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(episodes))
	}
	if episodes[0].Content != "2" {
		t.Fatalf("expected content '2', got %s", episodes[0].Content)
	}
}

func TestEpisodicMemory_GetMostImportant(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "low", Importance: 0.1})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "high", Importance: 0.9})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "c", Content: "medium", Importance: 0.5})

	episodes, _ := mem.GetMostImportant(ctx, 2)
	if len(episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(episodes))
	}
	if episodes[0].Content != "high" {
		t.Fatalf("expected 'high' first, got %s", episodes[0].Content)
	}
}

func TestEpisodicMemory_GetByTimeRange(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	now := time.Now().UnixMilli()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "past", Timestamp: now - 10000})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "now", Timestamp: now})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "c", Content: "future", Timestamp: now + 10000})

	// Get episodes from past to now
	episodes, _ := mem.GetByTimeRange(ctx, now-15000, now+5000)
	if len(episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(episodes))
	}
}

func TestEpisodicMemory_Clear(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "1"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "2"})

	err := mem.Clear(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestEpisodicMemory_FilterWithLimit(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_ = mem.AddEpisode(ctx, memory.Episode{Type: "test", Content: "content"})
	}

	filter := &memory.EpisodeFilter{Limit: 3}
	episodes, _ := mem.GetEpisodes(ctx, filter)

	if len(episodes) != 3 {
		t.Fatalf("expected 3 episodes with limit, got %d", len(episodes))
	}
}

func TestEpisodicMemory_ImplementsEpisodicMemory(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	var _ memory.EpisodicMemory = mem
}
