# HelloAgents Go

<p align="center">
  <strong>A high-performance, modular AI Agent framework for Go</strong>
</p>

<p align="center">
  <a href="#features">Features</a> |
  <a href="#quick-start">Quick Start</a> |
  <a href="#agent-types">Agent Types</a> |
  <a href="#documentation">Documentation</a>
</p>

---

HelloAgents Go is a production-ready AI Agent framework that provides a unified, type-safe interface for building intelligent agents with multiple reasoning patterns, tool integration, memory systems, and RAG capabilities.

## Features

### Agent Patterns
- **SimpleAgent** - Conversational agent for Q&A and multi-turn dialogue
- **ReActAgent** - Reasoning + Acting pattern with tool usage
- **ReflectionAgent** - Self-improving through generate-reflect-refine loops
- **PlanAndSolveAgent** - Strategic planning and step-by-step execution

### Tool System
- Flexible tool registration and discovery
- Built-in tools: Calculator, Terminal, RAG
- Easy custom tool creation with `FuncTool` and `SimpleTool`
- Tool chaining and conditional execution
- Timeout and retry mechanisms

### Memory System
- **Working Memory** - Conversation history with LRU eviction and TTL
- **Episodic Memory** - Event storage with timestamps and importance ratings
- **Semantic Memory** - Vector-based similarity search

### RAG Pipeline
- Document loading (text, markdown)
- Recursive character chunking with overlap
- Vector embedding and storage
- Multiple retrieval strategies (similarity, multi-source, reranking)

### LLM Providers
- OpenAI (GPT-4o, GPT-4, GPT-3.5)
- Ollama (local models)
- Qwen (通义千问)
- DeepSeek
- vLLM
- Fallback and health-check wrappers

### Observability
- Full OpenTelemetry integration
- Distributed tracing with span propagation
- Metrics collection (counters, histograms, gauges)
- Structured logging with trace ID correlation

---

## Quick Start

### Installation

```bash
go get github.com/easyops/helloagents-go
```

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/easyops/helloagents-go/pkg/agents"
    "github.com/easyops/helloagents-go/pkg/core/llm"
)

func main() {
    ctx := context.Background()

    // Create LLM provider
    provider, err := llm.NewOpenAI(
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel("gpt-4o-mini"),
    )
    if err != nil {
        panic(err)
    }
    defer provider.Close()

    // Create agent
    agent, err := agents.NewSimple(provider,
        agents.WithName("Assistant"),
        agents.WithSystemPrompt("You are a helpful AI assistant."),
    )
    if err != nil {
        panic(err)
    }

    // Run
    output, err := agent.Run(ctx, agents.Input{
        Query: "What is the capital of France?",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(output.Response)
    fmt.Printf("Tokens: %d, Duration: %s\n", output.TokenUsage.TotalTokens, output.Duration)
}
```

### ReAct Agent with Tools

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/easyops/helloagents-go/pkg/agents"
    "github.com/easyops/helloagents-go/pkg/core/llm"
    "github.com/easyops/helloagents-go/pkg/tools"
    "github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func main() {
    ctx := context.Background()

    // Create provider
    provider, _ := llm.NewOpenAI(
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel("gpt-4o"),
    )
    defer provider.Close()

    // Setup tools
    registry := tools.NewRegistry()
    registry.Register(builtin.NewCalculator())
    registry.Register(builtin.NewTerminal())

    // Create ReAct agent
    agent, _ := agents.NewReAct(provider, registry,
        agents.WithName("ToolAgent"),
        agents.WithMaxIterations(10),
    )

    // Run with tool usage
    output, _ := agent.Run(ctx, agents.Input{
        Query: "What is 123 * 456? Then list files in current directory.",
    })

    fmt.Println(output.Response)

    // Print reasoning steps
    for _, step := range output.Steps {
        fmt.Printf("[%s] %s\n", step.Type, step.Content)
    }
}
```

---

## Agent Types

### SimpleAgent

Basic conversational agent for Q&A and multi-turn dialogue.

```go
agent, _ := agents.NewSimple(provider,
    agents.WithName("ChatBot"),
    agents.WithSystemPrompt("You are a friendly assistant."),
    agents.WithAgentTemperature(0.7),
    agents.WithAgentMaxTokens(2048),
)

// Multi-turn conversation
output1, _ := agent.Run(ctx, agents.Input{Query: "Hi, I'm Alice"})
output2, _ := agent.Run(ctx, agents.Input{Query: "What's my name?"}) // Remembers context
```

### ReActAgent

Reasoning and Acting agent that uses tools to complete complex tasks.

```go
agent, _ := agents.NewReAct(provider, registry,
    agents.WithName("ToolUser"),
    agents.WithMaxIterations(10),
    agents.WithSystemPrompt("You are a helpful assistant with access to tools."),
)

// Agent will reason and call tools as needed
output, _ := agent.Run(ctx, agents.Input{
    Query: "Calculate the area of a circle with radius 5, then save it to area.txt",
})
```

### ReflectionAgent

Self-improving agent that generates, critiques, and refines responses.

```go
agent, _ := agents.NewReflection(provider,
    agents.WithMaxIterations(3),
    agents.WithSystemPrompt("You are an expert code reviewer."),
)

output, _ := agent.Run(ctx, agents.Input{
    Query: "Write a function to check if a number is prime",
})
// Agent will generate, reflect on quality, and improve the code
```

### PlanAndSolveAgent

Strategic agent that creates and executes step-by-step plans.

```go
agent, _ := agents.NewPlanAndSolve(provider, registry,
    agents.WithMaxIterations(5),
)

output, _ := agent.Run(ctx, agents.Input{
    Query: "Research the top 3 Go web frameworks and compare them",
})
// Agent will create a plan, execute each step, and synthesize results
```

---

## LLM Providers

### OpenAI

```go
provider, _ := llm.NewOpenAI(
    llm.WithAPIKey("sk-..."),
    llm.WithModel("gpt-4o"),           // or gpt-4o-mini, gpt-4, gpt-3.5-turbo
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(4096),
    llm.WithMaxRetries(3),
)
```

### Ollama (Local)

```go
provider := llm.NewOllamaClient(
    llm.WithOllamaBaseURL("http://localhost:11434"),
    llm.WithOllamaModel("llama3.2"),
)
```

### Qwen (通义千问)

```go
provider := llm.NewQwenClient(
    llm.WithQwenAPIKey("your-api-key"),
    llm.WithQwenModel("qwen-turbo"),
)
```

### DeepSeek

```go
provider := llm.NewDeepSeekClient(
    llm.WithDeepSeekAPIKey("your-api-key"),
    llm.WithDeepSeekModel("deepseek-chat"),
)
```

### vLLM

```go
provider := llm.NewVLLMClient(
    llm.WithVLLMBaseURL("http://localhost:8000"),
    llm.WithVLLMModel("meta-llama/Llama-2-7b-chat-hf"),
)
```

### Fallback Provider

```go
provider := llm.NewFallbackProvider(
    primaryProvider,
    fallbackProvider,
)
```

---

## Tool System

### Built-in Tools

```go
import "github.com/easyops/helloagents-go/pkg/tools/builtin"

registry := tools.NewRegistry()

// Calculator - evaluates math expressions
registry.Register(builtin.NewCalculator())

// Terminal - executes shell commands (with timeout)
registry.Register(builtin.NewTerminal())

// RAG - retrieves from document store
registry.Register(builtin.NewRAGTool(retriever))
```

### Custom Tools with FuncTool

```go
weatherTool := tools.NewFuncTool(
    "get_weather",
    "Get current weather for a city",
    tools.ParameterSchema{
        Type: "object",
        Properties: map[string]tools.PropertySchema{
            "city": {
                Type:        "string",
                Description: "City name",
            },
            "unit": {
                Type:        "string",
                Description: "Temperature unit",
                Enum:        []string{"celsius", "fahrenheit"},
                Default:     "celsius",
            },
        },
        Required: []string{"city"},
    },
    func(ctx context.Context, args map[string]interface{}) (string, error) {
        city := args["city"].(string)
        // Call weather API...
        return fmt.Sprintf("Weather in %s: Sunny, 25°C", city), nil
    },
)
registry.Register(weatherTool)
```

### Simple Tools (Single String Parameter)

```go
greetTool := tools.NewSimpleTool(
    "greet",
    "Greet someone by name",
    "name",
    "Person's name",
    func(ctx context.Context, name string) (string, error) {
        return "Hello, " + name + "!", nil
    },
)
```

### Tool Executor with Timeout

```go
executor := tools.NewExecutor(registry,
    tools.WithTimeout(30*time.Second),
    tools.WithMaxRetries(3),
)

result, err := executor.Execute(ctx, "calculator", map[string]interface{}{
    "expression": "2 + 2",
})
```

---

## Memory System

### Working Memory

Short-term conversation history with capacity management.

```go
mem := memory.NewWorkingMemory(
    memory.WithMaxSize(100),           // Max messages
    memory.WithTokenLimit(4000),       // Token budget
    memory.WithTTL(30*time.Minute),    // Expiration
)

// Store messages
mem.AddMessage(ctx, message.NewUserMessage("Hello"))
mem.AddMessage(ctx, message.NewAssistantMessage("Hi there!"))

// Retrieve history
history, _ := mem.GetHistory(ctx, 10)       // Last 10 messages
recent, _ := mem.GetRecentHistory(ctx, 5)   // Last 5 messages
```

### Episodic Memory

Event-based memory with importance ratings.

```go
mem := memory.NewEpisodicMemory()

// Store episodes
mem.AddEpisode(ctx, memory.Episode{
    Type:       "user_preference",
    Content:    "User prefers dark mode",
    Importance: 0.8,
})

// Query episodes
important, _ := mem.GetMostImportant(ctx, 5)
byType, _ := mem.GetByType(ctx, "user_preference", 10)
byTime, _ := mem.GetByTimeRange(ctx, startTime, endTime)
```

### Semantic Memory

Vector-based memory for similarity search.

```go
embedder := memory.NewOpenAIEmbedder(provider)
mem := memory.NewSemanticMemory(embedder)

// Store with optional metadata
mem.Store(ctx, "id-1", "User likes Go programming", map[string]interface{}{
    "category": "preferences",
})

// Semantic search
results, _ := mem.Search(ctx, "programming language preferences", 5)
for _, r := range results {
    fmt.Printf("%.2f: %s\n", r.Score, r.Content)
}
```

---

## RAG Pipeline

### Complete RAG Example

```go
// 1. Create components
chunker := rag.NewRecursiveCharacterChunker(512, 50)
store := rag.NewInMemoryVectorStore()
embedder := &OpenAIEmbedder{provider: provider}

// 2. Load and process documents
loader := rag.NewTextLoader()
doc, _ := loader.Load("document.txt")
chunks := chunker.Chunk(doc)

// 3. Generate embeddings and store
embeddings, _ := embedder.Embed(ctx, getContents(chunks))
for i, chunk := range chunks {
    chunk.Vector = embeddings[i]
}
store.Add(ctx, chunks)

// 4. Create retriever
retriever := rag.NewVectorRetriever(store, embedder,
    rag.WithScoreThreshold(0.7),
)

// 5. Query
results, _ := retriever.Retrieve(ctx, "What is HelloAgents?", 3)
for _, r := range results {
    fmt.Printf("Score: %.2f\nContent: %s\n\n", r.Score, r.Chunk.Content)
}
```

### RAG Pipeline Wrapper

```go
pipeline := rag.NewPipeline(embedder, store, answerGenerator)

// Ingest documents
docs := []rag.Document{doc1, doc2, doc3}
pipeline.Ingest(ctx, docs)

// Query with source tracking
response, _ := pipeline.Query(ctx, "What features does HelloAgents have?")
fmt.Println("Answer:", response.Answer)
fmt.Println("Sources:", response.Sources)
```

### Multi-Source Retrieval

```go
multiRetriever := rag.NewMultiRetriever(
    []rag.Retriever{retriever1, retriever2, retriever3},
    &rag.ScoreBasedMerger{},
)
```

---

## Observability

### Setup OpenTelemetry

```go
config := otel.Config{
    Enabled:        true,
    ServiceName:    "my-agent-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    Tracing: otel.TracingConfig{
        Enabled:    true,
        Endpoint:   "localhost:4317",
        SampleRate: 1.0,
    },
    Metrics: otel.MetricsConfig{
        Enabled:  true,
        Endpoint: "localhost:4317",
    },
}

provider, _ := otel.NewProvider(config)
defer provider.Shutdown(ctx)
```

### Traced LLM Provider

```go
tracedProvider := otel.NewTracedProvider(llmProvider, tracer, metrics)

// All LLM calls are automatically traced with:
// - Span for each Generate/GenerateStream call
// - Token usage metrics
// - Latency histograms
// - Error tracking
```

### Custom Tracing

```go
tracer := otel.NewTracer(otelTracer)

ctx, span := tracer.Start(ctx, "my-operation",
    otel.WithSpanKind(otel.SpanKindInternal),
)
defer span.End()

span.SetAttributes(
    attribute.String("query", input.Query),
    attribute.Int("iteration", i),
)
span.AddEvent("tool_executed", attribute.String("tool", toolName))
```

### Metrics

```go
metrics := otel.NewInMemoryMetrics()

// Counter
counter := metrics.Counter("agent.requests")
counter.Add(ctx, 1, otel.NewAttr("agent_type", "react"))

// Histogram
histogram := metrics.Histogram("agent.latency_seconds")
histogram.Record(ctx, duration.Seconds())

// Gauge
gauge := metrics.Gauge("agent.active_sessions")
gauge.Set(ctx, float64(activeSessions))
```

---

## Project Structure

```
HelloAgents/
├── pkg/
│   ├── core/                    # Core abstractions
│   │   ├── llm/                 # LLM providers (OpenAI, Ollama, etc.)
│   │   ├── message/             # Message types and roles
│   │   ├── config/              # Configuration management
│   │   └── errors/              # Error types and handling
│   ├── agents/                  # Agent implementations
│   │   ├── simple.go            # SimpleAgent
│   │   ├── react.go             # ReActAgent
│   │   ├── reflection.go        # ReflectionAgent
│   │   └── plansolve.go         # PlanAndSolveAgent
│   ├── tools/                   # Tool system
│   │   ├── registry.go          # Tool registration
│   │   ├── executor.go          # Tool execution
│   │   ├── func_tool.go         # FuncTool/SimpleTool
│   │   └── builtin/             # Built-in tools
│   ├── memory/                  # Memory system
│   │   ├── working.go           # Working memory
│   │   ├── episodic.go          # Episodic memory
│   │   └── semantic.go          # Semantic memory
│   ├── rag/                     # RAG pipeline
│   │   ├── chunker.go           # Document chunking
│   │   ├── store.go             # Vector storage
│   │   ├── retriever.go         # Retrieval strategies
│   │   └── pipeline.go          # Pipeline orchestration
│   └── otel/                    # Observability
│       ├── tracer.go            # Distributed tracing
│       ├── metrics.go           # Metrics collection
│       └── traced_provider.go   # Traced LLM wrapper
├── examples/                    # Example applications
│   ├── simple/                  # Basic chat
│   ├── react/                   # Tool-using agent
│   ├── memory/                  # Memory demo
│   └── rag/                     # RAG Q&A
├── tests/                       # Test suites
│   ├── unit/                    # Unit tests
│   └── integration/             # Integration tests
└── docs/                        # Documentation
    └── ARCHITECTURE.md          # Architecture guide
```

---

## Configuration

### Environment Variables

```bash
# LLM Configuration
export OPENAI_API_KEY=sk-...
export HELLOAGENTS_LLM_MODEL=gpt-4o
export HELLOAGENTS_LLM_TIMEOUT=30s
export HELLOAGENTS_LLM_MAX_RETRIES=3

# Agent Configuration
export HELLOAGENTS_AGENT_MAX_ITERATIONS=10
export HELLOAGENTS_AGENT_TEMPERATURE=0.7
export HELLOAGENTS_AGENT_MAX_TOKENS=4096

# Observability
export HELLOAGENTS_OTEL_ENABLED=true
export HELLOAGENTS_OTEL_ENDPOINT=localhost:4317
```

### Config File (YAML)

```yaml
llm:
  api_key: ${OPENAI_API_KEY}
  model: gpt-4o
  timeout: 30s
  max_retries: 3

agent:
  max_iterations: 10
  temperature: 0.7
  max_tokens: 4096

observability:
  enabled: true
  service_name: my-agent
  tracing:
    enabled: true
    endpoint: localhost:4317
```

---

## Examples

```bash
# Clone the repository
git clone https://github.com/easyops/helloagents-go.git
cd helloagents-go

# Set API key
export OPENAI_API_KEY=your-key

# Run examples
cd examples/simple && go run main.go     # Basic chat
cd examples/react && go run main.go      # ReAct with tools
cd examples/memory && go run main.go     # Memory system
cd examples/rag && go run main.go        # RAG document Q&A
```

---

## Testing

```bash
# Run all tests
go test ./...

# Run unit tests only
go test ./tests/unit/...

# With coverage
go test ./tests/unit/... -coverpkg=./pkg/...

# Verbose output
go test ./tests/unit/... -v
```

---

## Documentation

- **[Architecture Guide](docs/ARCHITECTURE.md)** - Detailed system architecture
- **[Examples](examples/)** - Working code examples

---

## Requirements

- Go 1.24 or later
- LLM provider API key (OpenAI, or alternative)

---

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.

---

## License

MIT License - see [LICENSE](LICENSE) for details.
