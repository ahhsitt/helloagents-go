// Package store provides storage backends for the memory system.
//
// This package defines interfaces for document storage (SQLite),
// vector storage (Qdrant), and graph storage (Neo4j).
package store

import (
	"context"
	"time"
)

// DocumentStore 文档存储接口
//
// 用于存储结构化的记忆数据，支持 CRUD 操作和条件查询。
// 默认实现使用内存存储，生产环境建议使用 SQLite。
type DocumentStore interface {
	// Put 存储文档
	Put(ctx context.Context, collection string, id string, doc Document) error

	// Get 获取文档
	Get(ctx context.Context, collection string, id string) (*Document, error)

	// Delete 删除文档
	Delete(ctx context.Context, collection string, id string) error

	// Query 条件查询
	Query(ctx context.Context, collection string, filter Filter, opts ...QueryOption) ([]Document, error)

	// Count 统计数量
	Count(ctx context.Context, collection string, filter Filter) (int, error)

	// Clear 清空集合
	Clear(ctx context.Context, collection string) error

	// Close 关闭连接
	Close() error
}

// Document 文档结构
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Filter 查询过滤条件
type Filter struct {
	// Field 字段名
	Field string
	// Op 操作符: eq, ne, gt, gte, lt, lte, in, contains
	Op string
	// Value 值
	Value interface{}
	// And 与条件
	And []Filter
	// Or 或条件
	Or []Filter
}

// QueryOption 查询选项
type QueryOption func(*queryOptions)

type queryOptions struct {
	limit   int
	offset  int
	orderBy string
	desc    bool
}

// WithQueryLimit 设置返回数量限制
func WithQueryLimit(limit int) QueryOption {
	return func(o *queryOptions) {
		o.limit = limit
	}
}

// WithQueryOffset 设置偏移量
func WithQueryOffset(offset int) QueryOption {
	return func(o *queryOptions) {
		o.offset = offset
	}
}

// WithQueryOrderBy 设置排序字段
func WithQueryOrderBy(field string, desc bool) QueryOption {
	return func(o *queryOptions) {
		o.orderBy = field
		o.desc = desc
	}
}

// VectorStore 向量存储接口
//
// 用于存储和检索向量数据，支持相似度搜索。
// 默认实现使用内存存储，生产环境建议使用 Qdrant。
type VectorStore interface {
	// AddVectors 批量添加向量
	AddVectors(ctx context.Context, collection string, vectors []VectorRecord) error

	// SearchSimilar 相似度搜索
	SearchSimilar(ctx context.Context, collection string, vector []float32, topK int, filter *VectorFilter) ([]VectorSearchResult, error)

	// DeleteVectors 按 ID 删除向量
	DeleteVectors(ctx context.Context, collection string, ids []string) error

	// DeleteByFilter 按条件删除
	DeleteByFilter(ctx context.Context, collection string, filter *VectorFilter) error

	// Clear 清空集合
	Clear(ctx context.Context, collection string) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context, collection string) (*VectorStoreStats, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// Close 关闭连接
	Close() error
}

// VectorRecord 向量记录
type VectorRecord struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	MemoryID string                 `json:"memory_id,omitempty"`
}

// VectorFilter 向量过滤条件
type VectorFilter struct {
	// MemoryID 按记忆 ID 过滤
	MemoryID string
	// UserID 按用户 ID 过滤
	UserID string
	// MemoryType 按记忆类型过滤
	MemoryType string
	// Conditions 自定义条件
	Conditions map[string]interface{}
}

// VectorSearchResult 向量搜索结果
type VectorSearchResult struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	MemoryID string                 `json:"memory_id,omitempty"`
}

// VectorStoreStats 向量存储统计
type VectorStoreStats struct {
	// VectorCount 向量数量
	VectorCount int `json:"vector_count"`
	// Dimensions 向量维度
	Dimensions int `json:"dimensions"`
	// IndexedCount 已索引数量
	IndexedCount int `json:"indexed_count"`
}

// GraphStore 图存储接口
//
// 用于存储实体和关系，支持图遍历查询。
// 默认实现使用内存存储，生产环境建议使用 Neo4j。
type GraphStore interface {
	// AddEntity 添加/更新实体节点
	AddEntity(ctx context.Context, entity *GraphEntity) error

	// GetEntity 获取实体
	GetEntity(ctx context.Context, id string) (*GraphEntity, error)

	// SearchEntities 按名称模式搜索实体
	SearchEntities(ctx context.Context, pattern string, entityType string, limit int) ([]*GraphEntity, error)

	// DeleteEntity 删除实体及其关系
	DeleteEntity(ctx context.Context, id string) error

	// AddRelation 添加/更新关系
	AddRelation(ctx context.Context, relation *GraphRelation) error

	// GetRelations 获取实体的所有关系
	GetRelations(ctx context.Context, entityID string) ([]*GraphRelation, error)

	// FindRelatedEntities 图遍历查找相关实体
	FindRelatedEntities(ctx context.Context, entityID string, relationType string, maxDepth int) ([]*GraphTraversalResult, error)

	// GetShortestPath 最短路径查询
	GetShortestPath(ctx context.Context, fromID, toID string) ([]*GraphEntity, []*GraphRelation, error)

	// DeleteRelation 删除关系
	DeleteRelation(ctx context.Context, id string) error

	// Clear 清空所有数据
	Clear(ctx context.Context) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context) (*GraphStoreStats, error)

	// Close 关闭连接
	Close() error
}

// GraphEntity 图实体
type GraphEntity struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Frequency   int                    `json:"frequency"`
	Vector      []float32              `json:"-"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// GraphRelation 图关系
type GraphRelation struct {
	ID           string                 `json:"id"`
	FromEntityID string                 `json:"from_entity_id"`
	ToEntityID   string                 `json:"to_entity_id"`
	Type         string                 `json:"type"`
	Strength     float32                `json:"strength"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Evidence     []string               `json:"evidence,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// GraphTraversalResult 图遍历结果
type GraphTraversalResult struct {
	Entity   *GraphEntity     `json:"entity"`
	Depth    int              `json:"depth"`
	Path     []*GraphRelation `json:"path,omitempty"`
	Score    float32          `json:"score"`
	Relation *GraphRelation   `json:"relation,omitempty"`
}

// GraphStoreStats 图存储统计
type GraphStoreStats struct {
	// EntityCount 实体数量
	EntityCount int `json:"entity_count"`
	// RelationCount 关系数量
	RelationCount int `json:"relation_count"`
	// EntityTypes 实体类型分布
	EntityTypes map[string]int `json:"entity_types,omitempty"`
	// RelationTypes 关系类型分布
	RelationTypes map[string]int `json:"relation_types,omitempty"`
}

// StoreType 存储类型
type StoreType string

const (
	// StoreTypeMemory 内存存储
	StoreTypeMemory StoreType = "memory"
	// StoreTypeSQLite SQLite 存储
	StoreTypeSQLite StoreType = "sqlite"
	// StoreTypeQdrant Qdrant 存储
	StoreTypeQdrant StoreType = "qdrant"
	// StoreTypeNeo4j Neo4j 存储
	StoreTypeNeo4j StoreType = "neo4j"
)

// Config 存储配置
type Config struct {
	// Type 存储类型
	Type StoreType `json:"type"`

	// SQLite 配置
	SQLitePath string `json:"sqlite_path,omitempty"`

	// Qdrant 配置
	QdrantURL    string `json:"qdrant_url,omitempty"`
	QdrantAPIKey string `json:"qdrant_api_key,omitempty"`

	// Neo4j 配置
	Neo4jURI      string `json:"neo4j_uri,omitempty"`
	Neo4jUsername string `json:"neo4j_username,omitempty"`
	Neo4jPassword string `json:"neo4j_password,omitempty"`

	// 向量维度
	VectorDimensions int `json:"vector_dimensions,omitempty"`
}

// DefaultConfig 返回默认配置（内存存储）
func DefaultConfig() *Config {
	return &Config{
		Type:             StoreTypeMemory,
		VectorDimensions: 128,
	}
}
