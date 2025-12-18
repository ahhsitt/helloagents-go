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
