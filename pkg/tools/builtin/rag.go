package builtin

import (
	"context"
	"fmt"

	"github.com/easyops/helloagents-go/pkg/rag"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// RAGTool RAG 检索工具
type RAGTool struct {
	retriever rag.Retriever
	topK      int
}

// RAGToolOption RAG 工具选项
type RAGToolOption func(*RAGTool)

// WithTopK 设置默认检索数量
func WithRAGTopK(topK int) RAGToolOption {
	return func(t *RAGTool) {
		t.topK = topK
	}
}

// NewRAGTool 创建 RAG 检索工具
func NewRAGTool(retriever rag.Retriever, opts ...RAGToolOption) *RAGTool {
	t := &RAGTool{
		retriever: retriever,
		topK:      5,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Definition 返回工具定义
func (t *RAGTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        "rag_search",
		Description: "Search through the knowledge base to find relevant information. Use this tool when you need to look up specific information or answer questions based on stored documents.",
		Parameters: tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"query": {
					Type:        "string",
					Description: "The search query to find relevant documents",
				},
				"top_k": {
					Type:        "integer",
					Description: "Number of results to return (default: 5)",
				},
			},
			Required: []string{"query"},
		},
	}
}

// Name 返回工具名称
func (t *RAGTool) Name() string {
	return "rag_search"
}

// Description 返回工具描述
func (t *RAGTool) Description() string {
	return "Search through the knowledge base to find relevant information. Use this tool when you need to look up specific information or answer questions based on stored documents."
}

// Parameters 返回参数 Schema
func (t *RAGTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"query": {
				Type:        "string",
				Description: "The search query to find relevant documents",
			},
			"top_k": {
				Type:        "integer",
				Description: "Number of results to return (default: 5)",
			},
		},
		Required: []string{"query"},
	}
}

// Execute 执行 RAG 检索
func (t *RAGTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	topK := t.topK
	if topKVal, ok := args["top_k"]; ok {
		switch v := topKVal.(type) {
		case int:
			topK = v
		case float64:
			topK = int(v)
		}
	}

	// 执行检索
	results, err := t.retriever.Retrieve(ctx, query, topK)
	if err != nil {
		return "", fmt.Errorf("retrieval failed: %w", err)
	}

	if len(results) == 0 {
		return "No relevant documents found for the query.", nil
	}

	// 格式化结果
	return formatRAGResults(results), nil
}

// formatRAGResults 格式化 RAG 结果
func formatRAGResults(results []rag.RetrievalResult) string {
	var output string
	output = fmt.Sprintf("Found %d relevant document(s):\n\n", len(results))

	for i, r := range results {
		output += fmt.Sprintf("--- Result %d (Score: %.2f) ---\n", i+1, r.Score)

		if r.Chunk.Metadata.Source != "" {
			output += fmt.Sprintf("Source: %s\n", r.Chunk.Metadata.Source)
		}

		if r.Chunk.Metadata.Title != "" {
			output += fmt.Sprintf("Title: %s\n", r.Chunk.Metadata.Title)
		}

		output += fmt.Sprintf("Content:\n%s\n\n", r.Chunk.Content)
	}

	return output
}

// compile-time interface check
var _ tools.Tool = (*RAGTool)(nil)
