package memory

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// SemanticMemoryStore 语义记忆存储实现
//
// 基于向量相似度的记忆存储，支持语义搜索。
// 注意：这是一个简化的内存实现，生产环境建议使用专用向量数据库。
type SemanticMemoryStore struct {
	embedder Embedder
	records  []semanticRecord
	mu       sync.RWMutex
}

type semanticRecord struct {
	ID        string
	Content   string
	Vector    []float32
	Metadata  map[string]interface{}
}

// NewSemanticMemory 创建语义记忆存储
func NewSemanticMemory(embedder Embedder) *SemanticMemoryStore {
	return &SemanticMemoryStore{
		embedder: embedder,
		records:  make([]semanticRecord, 0),
	}
}

// Store 存储文本及其向量
func (m *SemanticMemoryStore) Store(ctx context.Context, id string, content string, metadata map[string]interface{}) error {
	// 生成 ID（如果未提供）
	if id == "" {
		id = uuid.New().String()
	}

	// 生成嵌入向量
	vectors, err := m.embedder.Embed(ctx, []string{content})
	if err != nil {
		return err
	}

	if len(vectors) == 0 {
		return ErrEmbeddingFailed
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在，如果存在则更新
	for i, rec := range m.records {
		if rec.ID == id {
			m.records[i] = semanticRecord{
				ID:       id,
				Content:  content,
				Vector:   vectors[0],
				Metadata: metadata,
			}
			return nil
		}
	}

	// 添加新记录
	m.records = append(m.records, semanticRecord{
		ID:       id,
		Content:  content,
		Vector:   vectors[0],
		Metadata: metadata,
	})

	return nil
}

// Search 搜索相似内容
func (m *SemanticMemoryStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// 生成查询向量
	vectors, err := m.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, ErrEmbeddingFailed
	}

	queryVector := vectors[0]

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 计算所有记录的相似度
	type scoredRecord struct {
		record semanticRecord
		score  float32
	}

	scored := make([]scoredRecord, len(m.records))
	for i, rec := range m.records {
		scored[i] = scoredRecord{
			record: rec,
			score:  cosineSimilarity(queryVector, rec.Vector),
		}
	}

	// 按相似度降序排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 返回 top K
	if topK > len(scored) {
		topK = len(scored)
	}

	results := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		results[i] = SearchResult{
			ID:       scored[i].record.ID,
			Content:  scored[i].record.Content,
			Score:    scored[i].score,
			Metadata: scored[i].record.Metadata,
		}
	}

	return results, nil
}

// SearchWithThreshold 搜索相似内容（带阈值过滤）
func (m *SemanticMemoryStore) SearchWithThreshold(ctx context.Context, query string, topK int, minScore float32) ([]SearchResult, error) {
	results, err := m.Search(ctx, query, topK*2) // 获取更多结果以便过滤
	if err != nil {
		return nil, err
	}

	// 过滤低于阈值的结果
	filtered := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if r.Score >= minScore {
			filtered = append(filtered, r)
		}
	}

	// 限制返回数量
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}

	return filtered, nil
}

// Delete 删除指定记录
func (m *SemanticMemoryStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rec := range m.records {
		if rec.ID == id {
			m.records = append(m.records[:i], m.records[i+1:]...)
			return nil
		}
	}

	return ErrNotFound
}

// Clear 清空所有记录
func (m *SemanticMemoryStore) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = make([]semanticRecord, 0)
	return nil
}

// Size 返回记录数量
func (m *SemanticMemoryStore) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.records)
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// compile-time interface check
var _ VectorMemory = (*SemanticMemoryStore)(nil)
