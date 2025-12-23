package rag

import (
	"context"
)

// PostProcessor 后处理器接口
// 对检索结果进行后处理（如重排序、压缩等）
type PostProcessor interface {
	// Process 处理检索结果
	Process(ctx context.Context, query string, results []RetrievalResult) ([]RetrievalResult, error)
}

// RerankPostProcessor 重排序后处理器
// 将现有的 Reranker 适配为 PostProcessor
type RerankPostProcessor struct {
	reranker Reranker
}

// NewRerankPostProcessor 创建重排序后处理器
func NewRerankPostProcessor(reranker Reranker) *RerankPostProcessor {
	return &RerankPostProcessor{
		reranker: reranker,
	}
}

// Process 执行重排序
func (p *RerankPostProcessor) Process(ctx context.Context, query string, results []RetrievalResult) ([]RetrievalResult, error) {
	if len(results) == 0 {
		return results, nil
	}
	return p.reranker.Rerank(ctx, query, results)
}

// compile-time interface check
var _ PostProcessor = (*RerankPostProcessor)(nil)
