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

	// Demo 4: MemoryManager (Unified Management)
	demoMemoryManager(ctx)

	// Demo 5: Storage Layer
	demoStorageLayer(ctx)

	fmt.Println("=== Example Complete ===")
}

// demoWorkingMemory demonstrates working memory for conversation history
func demoWorkingMemory(ctx context.Context) {
	fmt.Println("--- Demo 1: Working Memory ---")
	fmt.Println()

	// Create working memory with capacity and TTL settings
	mem := memory.NewWorkingMemory(
		memory.WithMaxSize(5),          // Keep only last 5 messages
		memory.WithTokenLimit(1000),    // Token limit for LLM context
		memory.WithTTL(10*time.Minute), // Messages expire after 10 minutes
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

	// NEW: Demonstrate retrieval with importance-based ranking
	fmt.Println("\nRetrieving memories by query (TF-IDF search):")
	items, err := mem.Retrieve(ctx, "AI agents", memory.WithLimit(3))
	if err != nil {
		log.Printf("Retrieve failed: %v", err)
	} else {
		for i, item := range items {
			fmt.Printf("  %d. [importance: %.2f] %s\n", i+1, item.Importance, truncate(item.Content, 50))
		}
	}

	// NEW: Get context summary
	summary, _ := mem.GetContextSummary(ctx, 500) // maxLength=500
	fmt.Printf("\nContext Summary:\n%s\n", summary)

	fmt.Println()
}

// demoEpisodicMemory demonstrates episodic memory for event storage
func demoEpisodicMemory(ctx context.Context) {
	fmt.Println("--- Demo 2: Episodic Memory ---")
	fmt.Println()

	// Create episodic memory
	mem := memory.NewEpisodicMemory()

	// Record some events with session IDs
	sessionID := "session-001"
	events := []memory.Episode{
		{
			Type:       "user_action",
			Content:    "User asked about weather in Tokyo",
			Importance: 0.3,
			SessionID:  sessionID,
			Metadata:   map[string]interface{}{"location": "Tokyo"},
		},
		{
			Type:       "tool_call",
			Content:    "Called weather API for Tokyo",
			Importance: 0.5,
			SessionID:  sessionID,
			Metadata:   map[string]interface{}{"tool": "weather_api", "status": "success"},
		},
		{
			Type:       "user_preference",
			Content:    "User prefers temperature in Celsius",
			Importance: 0.8,
			SessionID:  sessionID,
			Metadata:   map[string]interface{}{"preference": "celsius"},
		},
		{
			Type:       "user_action",
			Content:    "User asked about restaurants nearby",
			Importance: 0.4,
			SessionID:  sessionID,
		},
		{
			Type:       "error",
			Content:    "Failed to fetch restaurant data",
			Importance: 0.6,
			SessionID:  sessionID,
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

	// NEW: Get session episodes
	sessionEpisodes, _ := mem.GetSessionEpisodes(ctx, sessionID)
	fmt.Printf("\nEpisodes in session '%s': %d\n", sessionID, len(sessionEpisodes))

	// NEW: Find patterns
	patterns, _ := mem.FindPatterns(ctx, memory.WithMaxPatterns(3))
	fmt.Println("\nDetected patterns:")
	for _, p := range patterns {
		fmt.Printf("  - Keywords: %v, Frequency: %d\n", p.Keywords, p.Frequency)
	}

	// NEW: Get timeline
	timeline, _ := mem.GetTimeline(ctx,
		memory.WithTimelineRange(time.Now().Add(-1*time.Hour).UnixMilli(), time.Now().UnixMilli()),
		memory.WithTimelineLimit(5))
	fmt.Printf("\nTimeline entries (last hour): %d\n", len(timeline))

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

	// NEW: Entity and relation management
	fmt.Println("Entity Management:")

	// Add entities
	entities := []*memory.Entity{
		{ID: "user-1", Name: "Alice", Type: "person", Description: "Primary user"},
		{ID: "lang-1", Name: "Go", Type: "programming_language", Description: "User's favorite language"},
		{ID: "proj-1", Name: "AI Agent", Type: "project", Description: "Current project"},
	}

	for _, e := range entities {
		if err := mem.AddEntity(ctx, e); err != nil {
			log.Printf("Failed to add entity: %v", err)
		}
	}
	fmt.Printf("Added %d entities\n", len(entities))

	// Add relations
	relations := []*memory.Relation{
		{ID: "rel-1", FromEntityID: "user-1", ToEntityID: "lang-1", RelationType: "prefers", Strength: 0.9},
		{ID: "rel-2", FromEntityID: "user-1", ToEntityID: "proj-1", RelationType: "works_on", Strength: 0.8},
	}

	for _, r := range relations {
		if err := mem.AddRelation(ctx, r); err != nil {
			log.Printf("Failed to add relation: %v", err)
		}
	}
	fmt.Printf("Added %d relations\n", len(relations))

	// Get related entities
	related, _ := mem.GetRelatedEntities(ctx, "user-1", 2) // maxDepth=2
	fmt.Printf("\nEntities related to 'Alice': %d\n", len(related))
	for _, r := range related {
		fmt.Printf("  - %s (%s)\n", r.Entity.Name, r.Entity.Type)
	}

	fmt.Println()
}

// demoMemoryManager demonstrates the unified memory manager
func demoMemoryManager(ctx context.Context) {
	fmt.Println("--- Demo 4: Memory Manager ---")
	fmt.Println()

	// Create config
	config := memory.DefaultMemoryConfig()

	// Create memory manager
	manager := memory.NewMemoryManager(config,
		memory.WithManagerUserID("user-123"),
	)

	// Create and register memory types
	workingMem := memory.NewWorkingMemory(
		memory.WithMaxSize(100),
		memory.WithTokenLimit(4000),
	)
	episodicMem := memory.NewEpisodicMemory()
	semanticMem := memory.NewSemanticMemory(nil) // nil embedder uses TF-IDF

	manager.RegisterMemory(memory.MemoryTypeWorking, workingMem)
	manager.RegisterMemory(memory.MemoryTypeEpisodic, episodicMem)
	manager.RegisterMemory(memory.MemoryTypeSemantic, semanticMem)

	fmt.Println("Registered 3 memory types: working, episodic, semantic")

	// Add memories (auto-classification)
	memories := []struct {
		content  string
		explicit memory.MemoryType
	}{
		{"The meeting yesterday was very productive", ""},              // Will be classified as episodic
		{"Go is a statically typed programming language", ""},          // Will be classified as semantic
		{"Remember to call John at 3pm", ""},                           // Will be classified as working
		{"The user prefers dark mode", memory.MemoryTypeSemantic},      // Explicit type
		{"Yesterday's event was important", memory.MemoryTypeEpisodic}, // Explicit type
	}

	fmt.Println("\nAdding memories (with auto-classification):")
	for _, m := range memories {
		opts := []memory.AddMemoryOption{}
		if m.explicit != "" {
			opts = append(opts, memory.WithAddMemoryType(m.explicit))
		}

		id, err := manager.AddMemory(ctx, m.content, opts...)
		if err != nil {
			log.Printf("Failed to add memory: %v", err)
			continue
		}
		fmt.Printf("  Added: %s... (ID: %s)\n", truncate(m.content, 30), id[:8])
	}

	// Retrieve memories
	fmt.Println("\nRetrieving memories about 'programming':")
	results, _ := manager.RetrieveMemories(ctx, "programming", memory.WithLimit(3))
	for i, item := range results {
		fmt.Printf("  %d. [%s] %s (importance: %.2f)\n",
			i+1, item.MemoryType, truncate(item.Content, 40), item.Importance)
	}

	// Get statistics
	stats, _ := manager.GetStats(ctx)
	fmt.Printf("\nMemory Statistics:\n")
	fmt.Printf("  Total memories: %d\n", stats.TotalCount)
	for memType, memStats := range stats.ByType {
		fmt.Printf("  - %s: %d items\n", memType, memStats.Count)
	}

	// Demonstrate forget mechanism
	fmt.Println("\nForget low-importance memories:")
	forgotten, _ := manager.ForgetMemories(ctx, memory.ForgetByImportance,
		memory.WithThreshold(0.3))
	fmt.Printf("  Forgotten %d memories\n", forgotten)

	// Demonstrate consolidation
	fmt.Println("\nConsolidate important working memories to episodic:")
	consolidated, _ := manager.ConsolidateMemories(ctx,
		memory.WithConsolidateMinImportance(0.7),
		memory.WithConsolidateTargetType(memory.MemoryTypeEpisodic))
	fmt.Printf("  Consolidated %d memories\n", consolidated)

	fmt.Println()
}

// demoStorageLayer demonstrates the storage layer
func demoStorageLayer(ctx context.Context) {
	fmt.Println("--- Demo 5: Storage Layer ---")
	fmt.Println()

	// Import the store package
	// Note: This demo shows how to use the storage layer independently
	// In practice, storage is integrated with memory types

	fmt.Println("Storage backends available:")
	fmt.Println("  - MemoryStore: In-memory storage (default, for testing)")
	fmt.Println("  - SQLiteStore: Persistent document storage")
	fmt.Println("  - QdrantStore: Vector similarity search")
	fmt.Println("  - Neo4jStore: Graph-based entity/relation storage")

	fmt.Println("\nFactory functions:")
	fmt.Println("  - store.NewDocumentStore(config) -> SQLite or Memory")
	fmt.Println("  - store.NewVectorStore(config) -> Qdrant or Memory")
	fmt.Println("  - store.NewGraphStore(config) -> Neo4j or Memory")

	fmt.Println("\nExample usage:")
	fmt.Println(`
  // Create memory-based vector store (default)
  config := store.DefaultConfig()
  vectorStore, _ := store.NewVectorStore(config)

  // Add vectors
  vectors := []store.VectorRecord{
    {ID: "v1", Vector: []float32{0.1, 0.2, 0.3}, MemoryID: "mem-1"},
  }
  vectorStore.AddVectors(ctx, "memories", vectors)

  // Search similar vectors
  results, _ := vectorStore.SearchSimilar(ctx, "memories", queryVector, 10, nil)
`)

	fmt.Println()
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
