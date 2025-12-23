package rag

import (
	"context"
	"sync"
)

// Retriever 检索器接口
type Retriever interface {
	// Retrieve 检索与查询相关的文档块
	Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error)
}

// AdvancedRetriever 高级检索器接口（支持策略选项）
type AdvancedRetriever interface {
	Retriever
	// RetrieveWithOptions 使用策略选项检索
	RetrieveWithOptions(ctx context.Context, query string, topK int, opts ...RetrieveOption) ([]RetrievalResult, error)
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

// Retrieve 检索与查询相关的文档块（实现 Retriever 接口）
func (r *VectorRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
	return r.simpleRetrieve(ctx, query, topK)
}

// RetrieveWithOptions 使用策略选项检索（实现 AdvancedRetriever 接口）
func (r *VectorRetriever) RetrieveWithOptions(ctx context.Context, query string, topK int, opts ...RetrieveOption) ([]RetrievalResult, error) {
	// 应用选项
	options := applyOptions(opts)

	// 如果没有变换器，执行简单检索
	if len(options.Transformers) == 0 {
		return r.simpleRetrieve(ctx, query, topK)
	}

	// 执行策略管道
	return r.pipelineRetrieve(ctx, query, topK, options)
}

// simpleRetrieve 简单检索（无策略）
func (r *VectorRetriever) simpleRetrieve(ctx context.Context, query string, topK int) ([]RetrievalResult, error) {
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

// pipelineRetrieve 策略管道检索
func (r *VectorRetriever) pipelineRetrieve(ctx context.Context, query string, topK int, options *RetrieveOptions) ([]RetrievalResult, error) {
	// 设置超时
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 阶段 1: 查询变换
	transformedQueries, err := r.transformQueries(ctx, query, options.Transformers)
	if err != nil {
		return nil, err
	}

	// 如果没有变换结果，降级为原始查询
	if len(transformedQueries) == 0 {
		transformedQueries = []TransformedQuery{NewTransformedQuery(query)}
	}

	// 阶段 2: 并行检索 + 融合
	fetchK := topK
	if len(transformedQueries) > 1 && options.FetchMultiplier > 0 {
		fetchK = topK * options.FetchMultiplier
	}

	results, weights, err := r.parallelRetrieve(ctx, transformedQueries, fetchK)
	if err != nil {
		return nil, err
	}

	// 融合结果
	var fusedResults []RetrievalResult
	if len(results) == 1 {
		// 单查询，无需融合
		fusedResults = results[0]
		if len(fusedResults) > topK {
			fusedResults = fusedResults[:topK]
		}
	} else {
		// 多查询，使用融合策略
		fusion := options.Fusion
		if fusion == nil {
			fusion = NewRRFFusion(60)
		}
		fusedResults = fusion.Fuse(results, weights, topK)
	}

	// 阶段 3: 后处理
	if len(options.PostProcessors) > 0 {
		fusedResults, err = r.postProcess(ctx, query, fusedResults, options.PostProcessors)
		if err != nil {
			return nil, err
		}
	}

	return fusedResults, nil
}

// transformQueries 执行查询变换
func (r *VectorRetriever) transformQueries(ctx context.Context, query string, transformers []QueryTransformer) ([]TransformedQuery, error) {
	// 初始查询
	queries := []TransformedQuery{NewTransformedQuery(query)}

	// 串行执行变换器
	for _, transformer := range transformers {
		var newQueries []TransformedQuery
		for _, q := range queries {
			transformed, err := transformer.Transform(ctx, q.Query)
			if err != nil {
				// 变换失败时保留原查询继续
				newQueries = append(newQueries, q)
				continue
			}
			newQueries = append(newQueries, transformed...)
		}
		if len(newQueries) > 0 {
			queries = newQueries
		}
	}

	return queries, nil
}

// parallelRetrieve 并行检索多个查询
func (r *VectorRetriever) parallelRetrieve(ctx context.Context, queries []TransformedQuery, topK int) ([][]RetrievalResult, []float32, error) {
	results := make([][]RetrievalResult, len(queries))
	weights := make([]float32, len(queries))
	errors := make([]error, len(queries))

	var wg sync.WaitGroup
	wg.Add(len(queries))

	for i, q := range queries {
		go func(idx int, query TransformedQuery) {
			defer wg.Done()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errors[idx] = ctx.Err()
				return
			default:
			}

			// 执行检索
			result, err := r.simpleRetrieve(ctx, query.Query, topK)
			if err != nil {
				errors[idx] = err
				return
			}

			results[idx] = result
			weights[idx] = query.Weight
		}(i, q)
	}

	// 等待完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有查询完成
	case <-ctx.Done():
		// 超时，使用已完成的结果
	}

	// 过滤出成功的结果
	var validResults [][]RetrievalResult
	var validWeights []float32
	for i, result := range results {
		if result != nil {
			validResults = append(validResults, result)
			validWeights = append(validWeights, weights[i])
		}
	}

	// 如果所有查询都失败，返回第一个错误
	if len(validResults) == 0 {
		for _, err := range errors {
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return validResults, validWeights, nil
}

// postProcess 执行后处理
func (r *VectorRetriever) postProcess(ctx context.Context, query string, results []RetrievalResult, processors []PostProcessor) ([]RetrievalResult, error) {
	var err error
	for _, processor := range processors {
		results, err = processor.Process(ctx, query, results)
		if err != nil {
			return nil, err
		}
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
var _ AdvancedRetriever = (*VectorRetriever)(nil)
var _ ResultMerger = (*ScoreBasedMerger)(nil)
