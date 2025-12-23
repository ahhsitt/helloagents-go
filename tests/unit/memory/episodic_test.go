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

func TestEpisodicMemory_ImplementsMemory(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	var _ memory.Memory = mem
}

func TestEpisodicMemory_SessionManagement(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	// Add episodes with session IDs
	_ = mem.AddEpisode(ctx, memory.Episode{
		Type:      "action",
		Content:   "Started session",
		SessionID: "session1",
	})
	_ = mem.AddEpisode(ctx, memory.Episode{
		Type:      "action",
		Content:   "Did something",
		SessionID: "session1",
	})
	_ = mem.AddEpisode(ctx, memory.Episode{
		Type:      "action",
		Content:   "Different session",
		SessionID: "session2",
	})

	// Get sessions
	sessions := mem.GetSessions(ctx)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Get session episodes
	eps, err := mem.GetSessionEpisodes(ctx, "session1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 episodes in session1, got %d", len(eps))
	}
}

func TestEpisodicMemory_Add(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	item := memory.NewMemoryItem("Test event", memory.MemoryTypeEpisodic,
		memory.WithImportance(0.8),
		memory.WithMetadataKV("type", "test"),
		memory.WithMetadataKV("session_id", "sess1"),
	)

	id, err := mem.Add(ctx, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != item.ID {
		t.Errorf("expected id %s, got %s", item.ID, id)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestEpisodicMemory_Retrieve(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	// Add episodes with different content
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "The cat sat on the mat"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "The dog ran in the park"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "Birds fly in the sky"})

	// Search for cat-related episodes
	results, err := mem.Retrieve(ctx, "cat", memory.WithLimit(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestEpisodicMemory_Update(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	ep := memory.Episode{
		ID:      "ep1",
		Type:    "action",
		Content: "Original content",
	}
	_ = mem.AddEpisode(ctx, ep)

	// Update content
	newContent := "Updated content"
	err := mem.Update(ctx, "ep1", memory.WithContentUpdate(newContent))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify update
	episodes, _ := mem.GetEpisodes(ctx, nil)
	if episodes[0].Content != newContent {
		t.Errorf("expected content %q, got %q", newContent, episodes[0].Content)
	}
}

func TestEpisodicMemory_Remove(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	ep := memory.Episode{
		ID:        "ep1",
		Type:      "action",
		Content:   "Test",
		SessionID: "sess1",
	}
	_ = mem.AddEpisode(ctx, ep)

	err := mem.Remove(ctx, "ep1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem.Size() != 0 {
		t.Errorf("expected size 0, got %d", mem.Size())
	}

	// Session should be empty too
	eps, _ := mem.GetSessionEpisodes(ctx, "sess1")
	if len(eps) != 0 {
		t.Errorf("expected 0 episodes in session, got %d", len(eps))
	}
}

func TestEpisodicMemory_Has(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{ID: "ep1", Type: "test", Content: "Test"})

	if !mem.Has(ctx, "ep1") {
		t.Error("expected Has to return true for existing id")
	}
	if mem.Has(ctx, "nonexistent") {
		t.Error("expected Has to return false for nonexistent id")
	}
}

func TestEpisodicMemory_GetStats(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	now := time.Now().UnixMilli()
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "1", Importance: 0.5, Timestamp: now - 1000})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "2", Importance: 0.8, Timestamp: now})

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Count != 2 {
		t.Errorf("expected count 2, got %d", stats.Count)
	}
	expectedAvg := float32(0.65)
	if stats.AvgImportance < expectedAvg-0.01 || stats.AvgImportance > expectedAvg+0.01 {
		t.Errorf("expected avg importance around %f, got %f", expectedAvg, stats.AvgImportance)
	}
}

func TestEpisodicMemory_FindPatterns(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	// Add episodes with repeated keywords
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "User asked about weather forecast"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "User checked weather again"})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "action", Content: "Weather update requested"})

	patterns, err := mem.FindPatterns(ctx, memory.WithMinFrequency(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) == 0 {
		t.Error("expected at least one pattern")
	}

	// "weather" should be a frequent keyword
	found := false
	for _, p := range patterns {
		for _, kw := range p.Keywords {
			if kw == "weather" {
				found = true
				if p.Frequency < 3 {
					t.Errorf("expected weather frequency >= 3, got %d", p.Frequency)
				}
			}
		}
	}
	if !found {
		t.Error("expected 'weather' pattern")
	}
}

func TestEpisodicMemory_GetTimeline(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	now := time.Now().UnixMilli()
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "First", Timestamp: now - 2000})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "Second", Timestamp: now - 1000})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "c", Content: "Third", Timestamp: now})

	timeline, err := mem.GetTimeline(ctx, memory.WithTimelineLimit(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(timeline) != 3 {
		t.Errorf("expected 3 entries, got %d", len(timeline))
	}

	// Should be in chronological order
	if timeline[0].Episode.Content != "First" {
		t.Errorf("expected 'First' first, got %q", timeline[0].Episode.Content)
	}

	// Each entry should have a relative time
	for _, entry := range timeline {
		if entry.RelativeTime == "" {
			t.Error("expected non-empty RelativeTime")
		}
	}
}

func TestEpisodicMemory_ForgetByImportance(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "Low", Importance: 0.1})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "High", Importance: 0.9})

	count, err := mem.Forget(ctx, memory.ForgetByImportance, memory.WithThreshold(0.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forgotten, got %d", count)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestEpisodicMemory_ForgetByTime(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	now := time.Now()
	oldTs := now.AddDate(0, 0, -40).UnixMilli() // 40 days ago
	newTs := now.UnixMilli()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "Old", Timestamp: oldTs})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "New", Timestamp: newTs})

	count, err := mem.Forget(ctx, memory.ForgetByTime, memory.WithMaxAgeDays(30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forgotten, got %d", count)
	}
}

func TestEpisodicMemory_ForgetByCapacity(t *testing.T) {
	mem := memory.NewEpisodicMemory()
	ctx := context.Background()

	_ = mem.AddEpisode(ctx, memory.Episode{Type: "a", Content: "Low 1", Importance: 0.2})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "b", Content: "High", Importance: 0.9})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "c", Content: "Low 2", Importance: 0.3})
	_ = mem.AddEpisode(ctx, memory.Episode{Type: "d", Content: "Medium", Importance: 0.6})

	count, err := mem.Forget(ctx, memory.ForgetByCapacity, memory.WithTargetCapacity(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 forgotten, got %d", count)
	}
	if mem.Size() != 2 {
		t.Errorf("expected size 2, got %d", mem.Size())
	}
}
