package memory

import "time"

// MemoryConfig 记忆系统配置
type MemoryConfig struct {
	// StoragePath 存储路径
	StoragePath string
	// MaxCapacity 最大容量
	MaxCapacity int
	// ImportanceThreshold 重要性阈值（低于此值的记忆可能被遗忘）
	ImportanceThreshold float32
	// DecayFactor 衰减因子（用于时间衰减计算）
	DecayFactor float32

	// 工作记忆配置
	WorkingMemoryCapacity int           // 工作记忆最大消息数
	WorkingMemoryTokens   int           // 工作记忆 token 限制
	WorkingMemoryTTL      time.Duration // 工作记忆 TTL

	// 情景记忆配置
	EpisodicMemoryCapacity int // 情景记忆最大事件数

	// 语义记忆配置
	SemanticMemoryCapacity int // 语义记忆最大记录数

	// 嵌入配置
	EmbedderType   string // 嵌入器类型: openai, dashscope, local
	EmbedderModel  string // 嵌入模型名称
	EmbedderAPIKey string // API 密钥

	// 存储配置
	SQLitePath     string // SQLite 数据库路径
	QdrantURL      string // Qdrant 服务地址
	QdrantAPIKey   string // Qdrant API 密钥
	QdrantCollection string // Qdrant 集合名称
	Neo4jURI       string // Neo4j 连接 URI
	Neo4jUsername  string // Neo4j 用户名
	Neo4jPassword  string // Neo4j 密码
}

// MemoryConfigOption 配置选项函数
type MemoryConfigOption func(*MemoryConfig)

// DefaultMemoryConfig 返回默认配置
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		StoragePath:         "./memory_data",
		MaxCapacity:         1000,
		ImportanceThreshold: 0.1,
		DecayFactor:         0.95,

		WorkingMemoryCapacity: 100,
		WorkingMemoryTokens:   4000,
		WorkingMemoryTTL:      2 * time.Hour,

		EpisodicMemoryCapacity: 10000,
		SemanticMemoryCapacity: 10000,

		EmbedderType:  "openai",
		EmbedderModel: "text-embedding-3-small",

		SQLitePath:       "./memory_data/memory.db",
		QdrantURL:        "localhost:6333",
		QdrantCollection: "hello_agents_memory",
		Neo4jURI:         "bolt://localhost:7687",
		Neo4jUsername:    "neo4j",
	}
}

// WithStoragePath 设置存储路径
func WithStoragePath(path string) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.StoragePath = path
	}
}

// WithMaxCapacity 设置最大容量
func WithMaxCapacity(capacity int) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.MaxCapacity = capacity
	}
}

// WithImportanceThreshold 设置重要性阈值
func WithImportanceThreshold(threshold float32) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.ImportanceThreshold = threshold
	}
}

// WithDecayFactor 设置衰减因子
func WithDecayFactor(factor float32) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.DecayFactor = factor
	}
}

// WithWorkingMemoryConfig 设置工作记忆配置
func WithWorkingMemoryConfig(capacity, tokens int, ttl time.Duration) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.WorkingMemoryCapacity = capacity
		c.WorkingMemoryTokens = tokens
		c.WorkingMemoryTTL = ttl
	}
}

// WithEpisodicMemoryCapacity 设置情景记忆容量
func WithEpisodicMemoryCapacity(capacity int) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.EpisodicMemoryCapacity = capacity
	}
}

// WithSemanticMemoryCapacity 设置语义记忆容量
func WithSemanticMemoryCapacity(capacity int) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.SemanticMemoryCapacity = capacity
	}
}

// WithEmbedderConfig 设置嵌入器配置
func WithEmbedderConfig(embedderType, model, apiKey string) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.EmbedderType = embedderType
		c.EmbedderModel = model
		c.EmbedderAPIKey = apiKey
	}
}

// WithSQLiteConfig 设置 SQLite 配置
func WithSQLiteConfig(path string) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.SQLitePath = path
	}
}

// WithQdrantConfig 设置 Qdrant 配置
func WithQdrantConfig(url, apiKey, collection string) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.QdrantURL = url
		c.QdrantAPIKey = apiKey
		c.QdrantCollection = collection
	}
}

// WithNeo4jConfig 设置 Neo4j 配置
func WithNeo4jConfig(uri, username, password string) MemoryConfigOption {
	return func(c *MemoryConfig) {
		c.Neo4jURI = uri
		c.Neo4jUsername = username
		c.Neo4jPassword = password
	}
}

// NewMemoryConfig 创建配置
func NewMemoryConfig(opts ...MemoryConfigOption) *MemoryConfig {
	config := DefaultMemoryConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}
