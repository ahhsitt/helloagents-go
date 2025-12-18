package rag

import (
	"context"
)

// Retriever 检索器接口
type Retriever interface {
	// Retrieve 检索与查询相关的文档块
	Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error)
}

// VectorRetriever 向量检索器
type VectorRetriever struct {
	store          VectorStore
	embedder       Embedder
	scoreThreshold float32
}

// VectorRetrieverOption 向量检索器选项
type VectorRetrieverOption func(*VectorRetriever)

// WithScoreThreshold 设置分数阈值
func WithScoreThreshold(threshold float32) VectorRetrieverOption {
	return func(r *VectorRetriever) {
		r.scoreThreshold = threshold
	}
}

// NewVectorRetriever 创建向量检索器
func NewVectorRetriever(store VectorStore, embedder Embedder, opts ...VectorRetrieverOption) *VectorRetriever {
	r := &VectorRetriever{
		store:          store,
		embedder:       embedder,
		scoreThreshold: 0.0, // 默认无阈值
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Retrieve 检索与查询相关的文档块
func (r *VectorRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	// 生成查询向量
	embeddings, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, nil
	}

	queryVector := embeddings[0]

	// 从向量存储中搜索
	results, err := r.store.Search(ctx, queryVector, topK)
	if err != nil {
		return nil, err
	}

	// 应用分数阈值过滤
	if r.scoreThreshold > 0 {
		filtered := make([]RetrievalResult, 0, len(results))
		for _, result := range results {
			if result.Score >= r.scoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	return results, nil
}

// MultiRetriever 多检索器（支持多个检索源）
type MultiRetriever struct {
	retrievers []Retriever
	merger     ResultMerger
}

// ResultMerger 结果合并器接口
type ResultMerger interface {
	// Merge 合并多个检索结果
	Merge(results [][]RetrievalResult, topK int) []RetrievalResult
}

// ScoreBasedMerger 基于分数的合并器
type ScoreBasedMerger struct{}

// Merge 按分数合并并排序
func (m *ScoreBasedMerger) Merge(results [][]RetrievalResult, topK int) []RetrievalResult {
	// 收集所有结果
	var all []RetrievalResult
	for _, r := range results {
		all = append(all, r...)
	}

	// 按分数降序排序
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].Score > all[i].Score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	// 返回 top K
	if topK > len(all) {
		topK = len(all)
	}

	return all[:topK]
}

// NewMultiRetriever 创建多检索器
func NewMultiRetriever(retrievers []Retriever, merger ResultMerger) *MultiRetriever {
	if merger == nil {
		merger = &ScoreBasedMerger{}
	}
	return &MultiRetriever{
		retrievers: retrievers,
		merger:     merger,
	}
}

// Retrieve 从多个源检索并合并结果
func (r *MultiRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	results := make([][]RetrievalResult, len(r.retrievers))

	// 从所有检索器获取结果
	for i, retriever := range r.retrievers {
		res, err := retriever.Retrieve(ctx, query, topK)
		if err != nil {
			return nil, err
		}
		results[i] = res
	}

	// 合并结果
	return r.merger.Merge(results, topK), nil
}

// RerankRetriever 重排序检索器（先检索再重排序）
type RerankRetriever struct {
	baseRetriever Retriever
	reranker      Reranker
	initialTopK   int
}

// Reranker 重排序器接口
type Reranker interface {
	// Rerank 重新排序检索结果
	Rerank(ctx context.Context, query string, results []RetrievalResult) ([]RetrievalResult, error)
}

// NewRerankRetriever 创建重排序检索器
func NewRerankRetriever(base Retriever, reranker Reranker, initialTopK int) *RerankRetriever {
	return &RerankRetriever{
		baseRetriever: base,
		reranker:      reranker,
		initialTopK:   initialTopK,
	}
}

// Retrieve 检索并重排序
func (r *RerankRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	// 先获取更多结果
	fetchK := r.initialTopK
	if fetchK < topK {
		fetchK = topK * 3 // 至少获取 3 倍的结果用于重排序
	}

	results, err := r.baseRetriever.Retrieve(ctx, query, fetchK)
	if err != nil {
		return nil, err
	}

	// 重排序
	reranked, err := r.reranker.Rerank(ctx, query, results)
	if err != nil {
		return nil, err
	}

	// 返回 top K
	if topK > len(reranked) {
		topK = len(reranked)
	}

	return reranked[:topK], nil
}

// compile-time interface check
var _ Retriever = (*VectorRetriever)(nil)
var _ Retriever = (*MultiRetriever)(nil)
var _ Retriever = (*RerankRetriever)(nil)
var _ ResultMerger = (*ScoreBasedMerger)(nil)
