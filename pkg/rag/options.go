package rag

import (
	"time"
)

// RetrieveOptions 检索选项
type RetrieveOptions struct {
	// Transformers 查询变换器列表（按顺序执行）
	Transformers []QueryTransformer

	// PostProcessors 后处理器列表（按顺序执行）
	PostProcessors []PostProcessor

	// Fusion 融合策略（默认 RRF）
	Fusion FusionStrategy

	// Timeout 超时控制
	Timeout time.Duration

	// FetchMultiplier 多查询时的获取倍数（用于融合后有足够结果）
	FetchMultiplier int
}

// DefaultRetrieveOptions 默认检索选项
func DefaultRetrieveOptions() *RetrieveOptions {
	return &RetrieveOptions{
		Fusion:          NewRRFFusion(60),
		Timeout:         30 * time.Second,
		FetchMultiplier: 2,
	}
}

// RetrieveOption 检索选项函数
type RetrieveOption func(*RetrieveOptions)

// WithTransformer 添加查询变换器
func WithTransformer(transformer QueryTransformer) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	}
}

// WithMQE 启用多查询扩展
func WithMQE(llm LLMProvider, numQueries int) RetrieveOption {
	return func(opts *RetrieveOptions) {
		transformer := NewMultiQueryTransformer(llm, WithNumQueries(numQueries))
		opts.Transformers = append(opts.Transformers, transformer)
	}
}

// WithMQETransformer 使用已配置的 MQE 变换器
func WithMQETransformer(transformer *MultiQueryTransformer) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	}
}

// WithHyDE 启用假设文档嵌入
func WithHyDE(llm LLMProvider) RetrieveOption {
	return func(opts *RetrieveOptions) {
		transformer := NewHyDETransformer(llm)
		opts.Transformers = append(opts.Transformers, transformer)
	}
}

// WithHyDETransformer 使用已配置的 HyDE 变换器
func WithHyDETransformer(transformer *HyDETransformer) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	}
}

// WithPostProcessor 添加后处理器
func WithPostProcessor(processor PostProcessor) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.PostProcessors = append(opts.PostProcessors, processor)
	}
}

// WithRerank 启用重排序
func WithRerank(reranker Reranker) RetrieveOption {
	return func(opts *RetrieveOptions) {
		processor := NewRerankPostProcessor(reranker)
		opts.PostProcessors = append(opts.PostProcessors, processor)
	}
}

// WithFusion 设置融合策略
func WithFusion(fusion FusionStrategy) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Fusion = fusion
	}
}

// WithRRFFusion 使用 RRF 融合策略
func WithRRFFusion(k int) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Fusion = NewRRFFusion(k)
	}
}

// WithScoreBasedFusion 使用基于分数的融合策略
func WithScoreBasedFusion() RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Fusion = NewScoreBasedFusion()
	}
}

// WithTimeout 设置超时
func WithTimeout(timeout time.Duration) RetrieveOption {
	return func(opts *RetrieveOptions) {
		opts.Timeout = timeout
	}
}

// WithFetchMultiplier 设置获取倍数
func WithFetchMultiplier(multiplier int) RetrieveOption {
	return func(opts *RetrieveOptions) {
		if multiplier > 0 {
			opts.FetchMultiplier = multiplier
		}
	}
}

// applyOptions 应用选项
func applyOptions(opts []RetrieveOption) *RetrieveOptions {
	options := DefaultRetrieveOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}
