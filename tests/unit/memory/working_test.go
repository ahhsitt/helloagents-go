package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/easyops/helloagents-go/pkg/memory"
)

func TestNewWorkingMemory_Defaults(t *testing.T) {
	mem := memory.NewWorkingMemory()
	if mem == nil {
		t.Fatal("expected non-nil memory")
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestWorkingMemory_AddMessage(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	msg := message.NewUserMessage("Hello")
	err := mem.AddMessage(ctx, msg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mem.Size())
	}
}

func TestWorkingMemory_GetHistory(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("Hello"))
	_ = mem.AddMessage(ctx, message.NewAssistantMessage("Hi there"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("How are you?"))

	history, err := mem.GetHistory(ctx, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(history))
	}
}

func TestWorkingMemory_GetHistoryWithLimit(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("1"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("2"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("3"))

	history, err := mem.GetHistory(ctx, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	// Should return last 2 messages
	if history[0].Content != "2" {
		t.Fatalf("expected '2', got %s", history[0].Content)
	}
}

func TestWorkingMemory_GetRecentHistory(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("1"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("2"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("3"))

	history, err := mem.GetRecentHistory(ctx, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
}

func TestWorkingMemory_Clear(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("Hello"))
	_ = mem.AddMessage(ctx, message.NewAssistantMessage("Hi"))

	err := mem.Clear(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mem.Size())
	}
}

func TestWorkingMemory_MaxSize(t *testing.T) {
	mem := memory.NewWorkingMemory(memory.WithMaxSize(3))
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("1"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("2"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("3"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("4"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("5"))

	if mem.Size() != 3 {
		t.Fatalf("expected size 3, got %d", mem.Size())
	}

	history, _ := mem.GetHistory(ctx, 0)
	// Should keep last 3
	if history[0].Content != "3" {
		t.Fatalf("expected '3', got %s", history[0].Content)
	}
}

func TestWorkingMemory_TokenLimit(t *testing.T) {
	mem := memory.NewWorkingMemory(memory.WithTokenLimit(100))
	ctx := context.Background()

	// Add messages
	_ = mem.AddMessage(ctx, message.NewUserMessage("Short message"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("Another message"))

	msgs, err := mem.GetMessagesWithinTokenLimit(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least one message within token limit")
	}
}

func TestWorkingMemory_TTL(t *testing.T) {
	mem := memory.NewWorkingMemory(memory.WithTTL(50 * time.Millisecond))
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("Old message"))

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	history, _ := mem.GetHistory(ctx, 0)
	if len(history) != 0 {
		t.Fatalf("expected 0 messages after TTL, got %d", len(history))
	}
}

func TestWorkingMemory_ImplementsConversationMemory(t *testing.T) {
	mem := memory.NewWorkingMemory()
	var _ memory.ConversationMemory = mem
}

func TestWorkingMemory_ImplementsMemory(t *testing.T) {
	mem := memory.NewWorkingMemory()
	var _ memory.Memory = mem
}

func TestWorkingMemory_AddWithImportance(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	msg := message.NewUserMessage("Important message")
	err := mem.AddMessageWithImportance(ctx, msg, 0.9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Get important messages
	items, err := mem.GetImportant(ctx, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Importance != 0.9 {
		t.Errorf("expected importance 0.9, got %f", items[0].Importance)
	}
}

func TestWorkingMemory_Add(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	item := memory.NewMemoryItem("Test content", memory.MemoryTypeWorking,
		memory.WithImportance(0.8),
	)

	id, err := mem.Add(ctx, item)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != item.ID {
		t.Errorf("expected id %s, got %s", item.ID, id)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestWorkingMemory_Retrieve(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	// Add messages with different content
	_ = mem.AddMessage(ctx, message.NewUserMessage("The cat sat on the mat"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("The dog ran in the park"))
	_ = mem.AddMessage(ctx, message.NewUserMessage("Birds fly in the sky"))

	// Search for cat-related messages
	results, err := mem.Retrieve(ctx, "cat", memory.WithLimit(2))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestWorkingMemory_Update(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	item := memory.NewMemoryItem("Original content", memory.MemoryTypeWorking)
	id, _ := mem.Add(ctx, item)

	// Update content
	newContent := "Updated content"
	err := mem.Update(ctx, id, memory.WithContentUpdate(newContent))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify update
	history, _ := mem.GetHistory(ctx, 0)
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != newContent {
		t.Errorf("expected content %q, got %q", newContent, history[0].Content)
	}
}

func TestWorkingMemory_Remove(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	item := memory.NewMemoryItem("Test content", memory.MemoryTypeWorking)
	id, _ := mem.Add(ctx, item)

	err := mem.Remove(ctx, id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mem.Size() != 0 {
		t.Errorf("expected size 0, got %d", mem.Size())
	}
}

func TestWorkingMemory_Has(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	item := memory.NewMemoryItem("Test content", memory.MemoryTypeWorking)
	id, _ := mem.Add(ctx, item)

	if !mem.Has(ctx, id) {
		t.Error("expected Has to return true for existing id")
	}
	if mem.Has(ctx, "nonexistent") {
		t.Error("expected Has to return false for nonexistent id")
	}
}

func TestWorkingMemory_GetStats(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Message 1"), 0.5)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Message 2"), 0.8)

	stats, err := mem.GetStats(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stats.Count != 2 {
		t.Errorf("expected count 2, got %d", stats.Count)
	}
	expectedAvg := float32(0.65)
	if stats.AvgImportance < expectedAvg-0.01 || stats.AvgImportance > expectedAvg+0.01 {
		t.Errorf("expected avg importance around %f, got %f", expectedAvg, stats.AvgImportance)
	}
}

func TestWorkingMemory_ForgetByImportance(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Low importance"), 0.1)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("High importance"), 0.9)

	count, err := mem.Forget(ctx, memory.ForgetByImportance, memory.WithThreshold(0.5))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forgotten, got %d", count)
	}
	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}
}

func TestWorkingMemory_ForgetByTime(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	// Add old message
	oldMsg := message.NewUserMessage("Old message")
	oldMsg.Timestamp = time.Now().AddDate(0, 0, -10) // 10 days ago
	_ = mem.AddMessage(ctx, oldMsg)

	// Add new message
	_ = mem.AddMessage(ctx, message.NewUserMessage("New message"))

	count, err := mem.Forget(ctx, memory.ForgetByTime, memory.WithMaxAgeDays(7))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 forgotten, got %d", count)
	}
}

func TestWorkingMemory_ForgetByCapacity(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	// Add messages with different importance
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Low 1"), 0.2)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("High"), 0.9)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Low 2"), 0.3)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Medium"), 0.6)

	count, err := mem.Forget(ctx, memory.ForgetByCapacity, memory.WithTargetCapacity(2))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 forgotten, got %d", count)
	}
	if mem.Size() != 2 {
		t.Errorf("expected size 2, got %d", mem.Size())
	}
}

func TestWorkingMemory_GetImportant(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Low"), 0.2)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("High"), 0.9)
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Medium"), 0.5)

	items, err := mem.GetImportant(ctx, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	// Should be sorted by importance
	if items[0].Importance < items[1].Importance {
		t.Error("expected items sorted by importance descending")
	}
}

func TestWorkingMemory_GetRecent(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("First"))
	time.Sleep(10 * time.Millisecond)
	_ = mem.AddMessage(ctx, message.NewUserMessage("Second"))
	time.Sleep(10 * time.Millisecond)
	_ = mem.AddMessage(ctx, message.NewUserMessage("Third"))

	items, err := mem.GetRecent(ctx, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	// Most recent should be first
	if items[0].Content != "Third" {
		t.Errorf("expected 'Third' first, got %q", items[0].Content)
	}
}

func TestWorkingMemory_GetContextSummary(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	_ = mem.AddMessage(ctx, message.NewUserMessage("Hello"))
	_ = mem.AddMessage(ctx, message.NewAssistantMessage("Hi there"))

	summary, err := mem.GetContextSummary(ctx, 1000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestWorkingMemory_CalculateScore(t *testing.T) {
	mem := memory.NewWorkingMemory()
	ctx := context.Background()

	// Add messages with different importance and timestamps
	_ = mem.AddMessageWithImportance(ctx, message.NewUserMessage("Important recent"), 0.9)

	// The scoring formula should consider similarity, time decay, and importance
	// Testing through retrieve to verify scoring works
	results, _ := mem.Retrieve(ctx, "Important", memory.WithLimit(1))
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}
