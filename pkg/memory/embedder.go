package memory

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/llm"
)

// OpenAIEmbedder OpenAI 嵌入实现
type OpenAIEmbedder struct {
	provider llm.Provider
}

// NewOpenAIEmbedder 创建 OpenAI 嵌入器
func NewOpenAIEmbedder(provider llm.Provider) *OpenAIEmbedder {
	return &OpenAIEmbedder{provider: provider}
}

// Embed 将文本转换为向量
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.provider.Embed(ctx, texts)
}

// compile-time interface check
var _ Embedder = (*OpenAIEmbedder)(nil)
