// Package memory 提供 Agent 记忆系统的接口和实现
package memory

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// ConversationMemory 对话记忆接口
//
// 用于存储和管理对话历史，支持多轮对话场景。
type ConversationMemory interface {
	// AddMessage 添加消息到记忆
	AddMessage(ctx context.Context, msg message.Message) error

	// GetHistory 获取对话历史
	// limit: 返回的最大消息数，0 表示返回所有
	GetHistory(ctx context.Context, limit int) ([]message.Message, error)

	// GetRecentHistory 获取最近 n 条消息
	GetRecentHistory(ctx context.Context, n int) ([]message.Message, error)

	// Clear 清空记忆
	Clear(ctx context.Context) error

	// Size 返回当前消息数量
	Size() int
}

// VectorMemory 向量记忆接口
//
// 用于语义相似度检索，支持基于内容的记忆查询。
type VectorMemory interface {
	// Store 存储文本及其向量
	Store(ctx context.Context, id string, content string, metadata map[string]interface{}) error

	// Search 搜索相似内容
	// query: 查询文本
	// topK: 返回最相似的 K 条记录
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)

	// Delete 删除指定记录
	Delete(ctx context.Context, id string) error

	// Clear 清空所有记录
	Clear(ctx context.Context) error
}

// SearchResult 向量搜索结果
type SearchResult struct {
	// ID 记录标识
	ID string `json:"id"`
	// Content 内容
	Content string `json:"content"`
	// Score 相似度分数 (0-1，越高越相似)
	Score float32 `json:"score"`
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EpisodicMemory 情景记忆接口
//
// 用于存储和检索特定事件或经历。
type EpisodicMemory interface {
	// AddEpisode 添加事件
	AddEpisode(ctx context.Context, episode Episode) error

	// GetEpisodes 获取事件列表
	// filter: 可选的过滤条件
	GetEpisodes(ctx context.Context, filter *EpisodeFilter) ([]Episode, error)

	// GetByTimeRange 按时间范围获取事件
	GetByTimeRange(ctx context.Context, start, end int64) ([]Episode, error)

	// Clear 清空所有事件
	Clear(ctx context.Context) error
}

// Episode 事件/情景
type Episode struct {
	// ID 唯一标识
	ID string `json:"id"`
	// Type 事件类型
	Type string `json:"type"`
	// Content 事件内容
	Content string `json:"content"`
	// Timestamp Unix 时间戳（毫秒）
	Timestamp int64 `json:"timestamp"`
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Importance 重要性评分 (0-1)
	Importance float32 `json:"importance,omitempty"`
}

// EpisodeFilter 事件过滤器
type EpisodeFilter struct {
	// Types 事件类型列表
	Types []string
	// MinImportance 最小重要性
	MinImportance float32
	// Limit 返回数量限制
	Limit int
}

// Embedder 文本嵌入接口
type Embedder interface {
	// Embed 将文本转换为向量
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
