package rag

import (
	"sort"
)

// FusionStrategy 融合策略接口
// 合并多个查询的检索结果
type FusionStrategy interface {
	// Fuse 融合多个结果集
	// results: 每个查询的检索结果列表
	// weights: 每个查询的权重（可选，长度应与 results 相同）
	// topK: 返回的最大结果数
	Fuse(results [][]RetrievalResult, weights []float32, topK int) []RetrievalResult
}

// RRFFusion 倒数排名融合 (Reciprocal Rank Fusion)
// 使用公式: score = sum(1 / (k + rank)) 计算融合分数
type RRFFusion struct {
	// K 排名常数，默认 60
	K int
}

// NewRRFFusion 创建 RRF 融合策略
func NewRRFFusion(k int) *RRFFusion {
	if k <= 0 {
		k = 60 // 默认值
	}
	return &RRFFusion{K: k}
}

// Fuse 执行 RRF 融合
func (f *RRFFusion) Fuse(results [][]RetrievalResult, weights []float32, topK int) []RetrievalResult {
	if len(results) == 0 {
		return nil
	}

	// 使用 chunk ID 作为唯一标识
	scoreMap := make(map[string]float32)
	chunkMap := make(map[string]RetrievalResult)

	for queryIdx, queryResults := range results {
		// 获取权重
		weight := float32(1.0)
		if queryIdx < len(weights) && weights[queryIdx] > 0 {
			weight = weights[queryIdx]
		}

		for rank, result := range queryResults {
			chunkID := result.Chunk.ID
			// RRF 公式: 1 / (k + rank)，rank 从 1 开始
			rrfScore := weight * (1.0 / float32(f.K+rank+1))
			scoreMap[chunkID] += rrfScore

			// 保留原始结果（用于返回）
			if _, exists := chunkMap[chunkID]; !exists {
				chunkMap[chunkID] = result
			}
		}
	}

	// 转换为结果列表并排序
	fusedResults := make([]RetrievalResult, 0, len(scoreMap))
	for chunkID, score := range scoreMap {
		result := chunkMap[chunkID]
		result.Score = score
		fusedResults = append(fusedResults, result)
	}

	// 按分数降序排序
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	// 返回 top K
	if topK > len(fusedResults) {
		topK = len(fusedResults)
	}

	return fusedResults[:topK]
}

// ScoreBasedFusion 基于分数的融合策略
// 合并所有结果，去重后按最高分数排序
type ScoreBasedFusion struct{}

// NewScoreBasedFusion 创建基于分数的融合策略
func NewScoreBasedFusion() *ScoreBasedFusion {
	return &ScoreBasedFusion{}
}

// Fuse 执行基于分数的融合
func (f *ScoreBasedFusion) Fuse(results [][]RetrievalResult, weights []float32, topK int) []RetrievalResult {
	if len(results) == 0 {
		return nil
	}

	// 使用 chunk ID 去重，保留最高分数
	scoreMap := make(map[string]float32)
	chunkMap := make(map[string]RetrievalResult)

	for queryIdx, queryResults := range results {
		// 获取权重
		weight := float32(1.0)
		if queryIdx < len(weights) && weights[queryIdx] > 0 {
			weight = weights[queryIdx]
		}

		for _, result := range queryResults {
			chunkID := result.Chunk.ID
			weightedScore := result.Score * weight

			if existingScore, exists := scoreMap[chunkID]; !exists || weightedScore > existingScore {
				scoreMap[chunkID] = weightedScore
				chunkMap[chunkID] = result
			}
		}
	}

	// 转换为结果列表并排序
	fusedResults := make([]RetrievalResult, 0, len(scoreMap))
	for chunkID, score := range scoreMap {
		result := chunkMap[chunkID]
		result.Score = score
		fusedResults = append(fusedResults, result)
	}

	// 按分数降序排序
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})

	// 返回 top K
	if topK > len(fusedResults) {
		topK = len(fusedResults)
	}

	return fusedResults[:topK]
}

// compile-time interface check
var _ FusionStrategy = (*RRFFusion)(nil)
var _ FusionStrategy = (*ScoreBasedFusion)(nil)
