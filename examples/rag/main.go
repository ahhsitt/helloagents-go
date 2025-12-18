// Package main demonstrates RAG (Retrieval-Augmented Generation) capabilities
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/easyops/helloagents-go/pkg/rag"
)

func main() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// 创建 LLM Provider
	provider, err := llm.NewOpenAI(
		llm.WithAPIKey(apiKey),
		llm.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	defer provider.Close()

	fmt.Println("=== RAG Document Q&A Example ===")
	fmt.Println()

	// 创建示例文档
	documents := createSampleDocuments()
	fmt.Printf("Created %d sample documents\n", len(documents))

	// 创建 RAG 组件
	chunker := rag.NewRecursiveCharacterChunker(200, 20)
	store := rag.NewInMemoryVectorStore()
	embedder := &llmEmbedder{provider: provider}

	// 创建 RAG Pipeline
	pipeline := rag.NewRAGPipeline(
		rag.WithChunker(chunker),
		rag.WithStore(store),
		rag.WithEmbedder(embedder),
	)

	// 摄取文档
	fmt.Println("\nIngesting documents...")
	if err := pipeline.Ingest(ctx, documents); err != nil {
		log.Fatalf("Failed to ingest documents: %v", err)
	}
	fmt.Printf("Ingested documents into vector store (size: %d chunks)\n", store.Size())

	// 创建检索器
	retriever := rag.NewVectorRetriever(store, embedder, rag.WithScoreThreshold(0.5))
	pipeline.SetRetriever(retriever)

	// 创建回答生成器
	generator := &llmAnswerGenerator{provider: provider}
	pipeline.SetGenerator(generator)

	// 测试查询
	queries := []string{
		"What is the HelloAgents framework?",
		"How does the ReAct pattern work?",
		"What types of memory does the system support?",
	}

	for _, query := range queries {
		fmt.Printf("\n--- Query: %s ---\n", query)

		response, err := pipeline.Query(ctx, query, 3)
		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		fmt.Printf("\nAnswer:\n%s\n", response.Answer)

		if len(response.Sources) > 0 {
			fmt.Println("\nSources:")
			for i, src := range response.Sources {
				fmt.Printf("  %d. [Score: %.2f] %s\n", i+1, src.Score, truncate(src.Content, 80))
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
}

// createSampleDocuments 创建示例文档
func createSampleDocuments() []rag.Document {
	return []rag.Document{
		{
			ID:      "doc-1",
			Content: `HelloAgents is a Golang AI Agent framework designed for building intelligent agents. It provides a modular architecture with support for multiple LLM providers including OpenAI, DeepSeek, Qwen, Ollama, and vLLM. The framework emphasizes observability through OpenTelemetry integration for tracing, metrics, and logging.`,
			Metadata: rag.DocumentMetadata{
				Source: "helloagents-overview.md",
				Title:  "HelloAgents Overview",
			},
		},
		{
			ID:      "doc-2",
			Content: `The ReAct (Reasoning and Acting) pattern enables agents to think step by step while taking actions. In HelloAgents, the ReActAgent implements a Thought-Action-Observation loop. First, the agent thinks about the current situation. Then, it decides on an action (often a tool call). Finally, it observes the result and continues the cycle until the task is complete.`,
			Metadata: rag.DocumentMetadata{
				Source: "react-pattern.md",
				Title:  "ReAct Pattern",
			},
		},
		{
			ID:      "doc-3",
			Content: `HelloAgents provides three types of memory: Working Memory for short-term conversation history with LRU eviction and TTL expiration. Episodic Memory stores events with timestamps for time-based retrieval. Semantic Memory uses vector embeddings for similarity-based search, powered by OpenAI embeddings or custom embedders.`,
			Metadata: rag.DocumentMetadata{
				Source: "memory-system.md",
				Title:  "Memory System",
			},
		},
		{
			ID:      "doc-4",
			Content: `The tool system in HelloAgents allows agents to execute external actions. Tools are defined with a name, description, and JSON schema for parameters. The ToolRegistry manages tool registration and lookup. Built-in tools include a calculator for mathematical operations and a terminal tool for command execution.`,
			Metadata: rag.DocumentMetadata{
				Source: "tools.md",
				Title:  "Tool System",
			},
		},
		{
			ID:      "doc-5",
			Content: `RAG (Retrieval-Augmented Generation) in HelloAgents combines document retrieval with LLM generation. Documents are loaded, chunked into smaller pieces, and embedded into vectors. When a query arrives, relevant chunks are retrieved using cosine similarity, then used as context for the LLM to generate informed answers with source citations.`,
			Metadata: rag.DocumentMetadata{
				Source: "rag-pipeline.md",
				Title:  "RAG Pipeline",
			},
		},
	}
}

// llmEmbedder LLM 嵌入器包装
type llmEmbedder struct {
	provider llm.Provider
}

func (e *llmEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.provider.Embed(ctx, texts)
}

// llmAnswerGenerator LLM 回答生成器
type llmAnswerGenerator struct {
	provider llm.Provider
}

func (g *llmAnswerGenerator) Generate(ctx context.Context, query string, ragContext *rag.RAGContext) (string, error) {
	// 构建提示
	var contextText strings.Builder
	contextText.WriteString("Based on the following context, answer the question.\n\n")
	contextText.WriteString("Context:\n")
	for _, result := range ragContext.Results {
		contextText.WriteString(fmt.Sprintf("- %s\n", result.Chunk.Content))
	}
	contextText.WriteString(fmt.Sprintf("\nQuestion: %s\n\nAnswer:", query))

	// 调用 LLM
	resp, err := g.provider.Generate(ctx, llm.Request{
		Messages: []message.Message{
			{
				Role:    message.RoleUser,
				Content: contextText.String(),
			},
		},
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

