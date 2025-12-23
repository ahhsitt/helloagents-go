package store

// NewDocumentStore 根据配置创建文档存储
func NewDocumentStore(config *Config) (DocumentStore, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case StoreTypeSQLite:
		return NewSQLiteDocumentStore(config.SQLitePath)
	case StoreTypeMemory:
		fallthrough
	default:
		return NewMemoryDocumentStore(), nil
	}
}

// NewVectorStore 根据配置创建向量存储
func NewVectorStore(config *Config) (VectorStore, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case StoreTypeQdrant:
		return NewQdrantVectorStore(QdrantConfig{
			URL:        config.QdrantURL,
			APIKey:     config.QdrantAPIKey,
			Dimensions: config.VectorDimensions,
		})
	case StoreTypeMemory:
		fallthrough
	default:
		return NewMemoryVectorStore(), nil
	}
}

// NewGraphStore 根据配置创建图存储
func NewGraphStore(config *Config) (GraphStore, error) {
	if config == nil {
		config = DefaultConfig()
	}

	switch config.Type {
	case StoreTypeNeo4j:
		return NewNeo4jGraphStore(Neo4jConfig{
			URI:      config.Neo4jURI,
			Username: config.Neo4jUsername,
			Password: config.Neo4jPassword,
		})
	case StoreTypeMemory:
		fallthrough
	default:
		return NewMemoryGraphStore(), nil
	}
}
