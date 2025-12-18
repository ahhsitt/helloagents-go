# Memory System Example

This example demonstrates the three types of memory available in the HelloAgents framework: Working Memory, Episodic Memory, and Semantic Memory.

## Overview

The example shows:
- **Working Memory**: Conversation history with LRU eviction and TTL expiration
- **Episodic Memory**: Event storage with timestamps, types, and importance ratings
- **Semantic Memory**: Vector-based storage with similarity search

## Prerequisites

- Go 1.21 or later
- OpenAI API key (optional, for semantic memory demo)

## Setup

1. Optionally set your OpenAI API key for the semantic memory demo:
   ```bash
   export OPENAI_API_KEY="your-api-key"
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

## Memory Types

### Working Memory

Short-term memory for conversation history with automatic cleanup.

```go
mem := memory.NewWorkingMemory(
    memory.WithMaxSize(5),          // Keep only last 5 messages
    memory.WithTokenLimit(1000),    // Token limit for LLM context
    memory.WithTTL(10*time.Minute), // Messages expire after 10 minutes
)

// Add messages
mem.AddMessage(ctx, message.NewUserMessage("Hello"))

// Get recent history
history, _ := mem.GetRecentHistory(ctx, 3)

// Get messages within token budget
msgs, _ := mem.GetMessagesWithinTokenLimit(ctx)
```

### Episodic Memory

Store and retrieve events with metadata and importance ratings.

```go
mem := memory.NewEpisodicMemory()

// Add an episode
mem.AddEpisode(ctx, memory.Episode{
    Type:       "user_preference",
    Content:    "User prefers dark mode",
    Importance: 0.8,
    Metadata:   map[string]interface{}{"category": "ui"},
})

// Query by type
events, _ := mem.GetByType(ctx, "user_preference", 10)

// Get most important events
important, _ := mem.GetMostImportant(ctx, 5)

// Query by time range
events, _ := mem.GetByTimeRange(ctx, startTime, endTime)
```

### Semantic Memory

Vector-based memory for similarity search using embeddings.

```go
// Create with an embedder
embedder := memory.NewOpenAIEmbedder(provider)
mem := memory.NewSemanticMemory(embedder)

// Store knowledge
mem.Store(ctx, "id-1", "User prefers Go programming", nil)

// Search by similarity
results, _ := mem.Search(ctx, "What language does user like?", 3)

// Search with threshold
results, _ := mem.SearchWithThreshold(ctx, "query", 3, 0.7)
```

## Sample Output

```
=== Memory System Example ===

--- Demo 1: Working Memory ---

Total messages added: 7
Messages in memory (with max size 5): 5

Last 3 messages:
  1. [assistant] Of course! Your name is Alice...
  2. [user] That's correct! Thank you.
  3. ...

--- Demo 2: Episodic Memory ---

Total events recorded: 5

All events (newest first):
  1. [error] Failed to fetch restaurant data (importance: 0.6)
  2. [user_action] User asked about restaurants nearby (importance: 0.4)
  ...

Top 2 most important events:
  1. [user_preference] User prefers temperature in Celsius (importance: 0.8)
  2. [error] Failed to fetch restaurant data (importance: 0.6)

--- Demo 3: Semantic Memory ---

Storing knowledge in semantic memory...
Stored 5 items

Query: What are the user's UI preferences?
Results:
  1. [Score: 0.89] The user prefers dark mode for the interface
  ...
```

## Use Cases

### Working Memory
- Multi-turn chat conversations
- Context management for LLM calls
- Automatic pruning of old messages

### Episodic Memory
- Logging user actions and tool calls
- Tracking errors and important events
- Time-based event queries

### Semantic Memory
- Storing user preferences and facts
- Knowledge retrieval based on context
- Long-term user profile building

## Related Examples

- [Simple Chat](../simple/README.md) - Basic agent conversation
- [ReAct Agent](../react/README.md) - Agent with tool calling
- [RAG Pipeline](../rag/README.md) - Document-based Q&A
