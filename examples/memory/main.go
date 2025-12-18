// Package main demonstrates the memory system capabilities
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/easyops/helloagents-go/pkg/memory"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Memory System Example ===")
	fmt.Println()

	// Demo 1: Working Memory (Conversation History)
	demoWorkingMemory(ctx)

	// Demo 2: Episodic Memory (Event Storage)
	demoEpisodicMemory(ctx)

	// Demo 3: Semantic Memory (Vector Search)
	demoSemanticMemory(ctx)

	fmt.Println("=== Example Complete ===")
}

// demoWorkingMemory demonstrates working memory for conversation history
func demoWorkingMemory(ctx context.Context) {
	fmt.Println("--- Demo 1: Working Memory ---")
	fmt.Println()

	// Create working memory with capacity and TTL settings
	mem := memory.NewWorkingMemory(
		memory.WithMaxSize(5),                    // Keep only last 5 messages
		memory.WithTokenLimit(1000),              // Token limit for LLM context
		memory.WithTTL(10*time.Minute),           // Messages expire after 10 minutes
	)

	// Simulate a conversation
	conversation := []message.Message{
		message.NewUserMessage("Hi, my name is Alice."),
		message.NewAssistantMessage("Hello Alice! Nice to meet you. How can I help you today?"),
		message.NewUserMessage("I'm interested in learning about AI agents."),
		message.NewAssistantMessage("AI agents are fascinating! They can perform tasks autonomously using LLMs."),
		message.NewUserMessage("Can you remember my name?"),
		message.NewAssistantMessage("Of course! Your name is Alice. You told me at the beginning of our conversation."),
		message.NewUserMessage("That's correct! Thank you."),
	}

	// Add messages to memory
	for _, msg := range conversation {
		if err := mem.AddMessage(ctx, msg); err != nil {
			log.Printf("Failed to add message: %v", err)
		}
	}

	fmt.Printf("Total messages added: %d\n", len(conversation))
	fmt.Printf("Messages in memory (with max size 5): %d\n", mem.Size())

	// Get recent history
	history, err := mem.GetRecentHistory(ctx, 3)
	if err != nil {
		log.Printf("Failed to get history: %v", err)
	}

	fmt.Println("\nLast 3 messages:")
	for i, msg := range history {
		fmt.Printf("  %d. [%s] %s\n", i+1, msg.Role, truncate(msg.Content, 50))
	}

	// Get messages within token limit
	tokenLimitedMsgs, err := mem.GetMessagesWithinTokenLimit(ctx)
	if err != nil {
		log.Printf("Failed to get token-limited messages: %v", err)
	}
	fmt.Printf("\nMessages within token limit: %d\n", len(tokenLimitedMsgs))

	fmt.Println()
}

// demoEpisodicMemory demonstrates episodic memory for event storage
func demoEpisodicMemory(ctx context.Context) {
	fmt.Println("--- Demo 2: Episodic Memory ---")
	fmt.Println()

	// Create episodic memory
	mem := memory.NewEpisodicMemory()

	// Record some events
	events := []memory.Episode{
		{
			Type:       "user_action",
			Content:    "User asked about weather in Tokyo",
			Importance: 0.3,
			Metadata:   map[string]interface{}{"location": "Tokyo"},
		},
		{
			Type:       "tool_call",
			Content:    "Called weather API for Tokyo",
			Importance: 0.5,
			Metadata:   map[string]interface{}{"tool": "weather_api", "status": "success"},
		},
		{
			Type:       "user_preference",
			Content:    "User prefers temperature in Celsius",
			Importance: 0.8,
			Metadata:   map[string]interface{}{"preference": "celsius"},
		},
		{
			Type:       "user_action",
			Content:    "User asked about restaurants nearby",
			Importance: 0.4,
		},
		{
			Type:       "error",
			Content:    "Failed to fetch restaurant data",
			Importance: 0.6,
			Metadata:   map[string]interface{}{"error_code": "API_TIMEOUT"},
		},
	}

	for _, ep := range events {
		if err := mem.AddEpisode(ctx, ep); err != nil {
			log.Printf("Failed to add episode: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("Total events recorded: %d\n", mem.Size())

	// Get all events
	allEvents, _ := mem.GetEpisodes(ctx, nil)
	fmt.Println("\nAll events (newest first):")
	for i, ep := range allEvents {
		fmt.Printf("  %d. [%s] %s (importance: %.1f)\n", i+1, ep.Type, truncate(ep.Content, 40), ep.Importance)
	}

	// Filter by type
	userActions, _ := mem.GetByType(ctx, "user_action", 10)
	fmt.Printf("\nUser actions: %d\n", len(userActions))

	// Get most important events
	important, _ := mem.GetMostImportant(ctx, 2)
	fmt.Println("\nTop 2 most important events:")
	for i, ep := range important {
		fmt.Printf("  %d. [%s] %s (importance: %.1f)\n", i+1, ep.Type, truncate(ep.Content, 40), ep.Importance)
	}

	fmt.Println()
}

// demoSemanticMemory demonstrates semantic memory with vector search
func demoSemanticMemory(ctx context.Context) {
	fmt.Println("--- Demo 3: Semantic Memory ---")
	fmt.Println()

	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Skipping semantic memory demo (OPENAI_API_KEY not set)")
		fmt.Println("Set OPENAI_API_KEY to enable vector-based semantic search")
		fmt.Println()
		return
	}

	// Create LLM provider for embeddings
	provider, err := llm.NewOpenAI(
		llm.WithAPIKey(apiKey),
		llm.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		log.Printf("Failed to create LLM provider: %v", err)
		return
	}
	defer provider.Close()

	// Create semantic memory with OpenAI embedder
	embedder := memory.NewOpenAIEmbedder(provider)
	mem := memory.NewSemanticMemory(embedder)

	// Store some knowledge
	knowledge := []struct {
		id       string
		content  string
		metadata map[string]interface{}
	}{
		{
			id:       "pref-1",
			content:  "The user prefers dark mode for the interface",
			metadata: map[string]interface{}{"type": "preference", "category": "ui"},
		},
		{
			id:       "pref-2",
			content:  "The user likes to receive notifications via email",
			metadata: map[string]interface{}{"type": "preference", "category": "notifications"},
		},
		{
			id:       "fact-1",
			content:  "The user's favorite programming language is Go",
			metadata: map[string]interface{}{"type": "fact", "category": "programming"},
		},
		{
			id:       "fact-2",
			content:  "The user is working on an AI agent project",
			metadata: map[string]interface{}{"type": "fact", "category": "projects"},
		},
		{
			id:       "fact-3",
			content:  "The user lives in San Francisco timezone (PST)",
			metadata: map[string]interface{}{"type": "fact", "category": "location"},
		},
	}

	fmt.Println("Storing knowledge in semantic memory...")
	for _, k := range knowledge {
		if err := mem.Store(ctx, k.id, k.content, k.metadata); err != nil {
			log.Printf("Failed to store: %v", err)
			return
		}
	}
	fmt.Printf("Stored %d items\n\n", mem.Size())

	// Search for relevant memories
	queries := []string{
		"What are the user's UI preferences?",
		"What programming does the user like?",
		"Where is the user located?",
	}

	for _, query := range queries {
		fmt.Printf("Query: %s\n", query)
		results, err := mem.Search(ctx, query, 2)
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		fmt.Println("Results:")
		for i, r := range results {
			fmt.Printf("  %d. [Score: %.2f] %s\n", i+1, r.Score, r.Content)
		}
		fmt.Println()
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
