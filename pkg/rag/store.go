package rag

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// VectorStore 向量存储接口
type VectorStore interface {
	// Add 添加文档块
	Add(ctx context.Context, chunks []DocumentChunk) error
	// Search 搜索相似文档块
	Search(ctx context.Context, query []float32, topK int) ([]RetrievalResult, error)
	// Delete 删除文档块
	Delete(ctx context.Context, ids []string) error
	// Clear 清空存储
	Clear(ctx context.Context) error
	// Size 返回存储的块数量
	Size() int
}

// Embedder 嵌入器接口
type Embedder interface {
	// Embed 生成文本嵌入向量
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// InMemoryVectorStore 内存向量存储
type InMemoryVectorStore struct {
	chunks map[string]DocumentChunk
	mu     sync.RWMutex
}

// NewInMemoryVectorStore 创建内存向量存储
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		chunks: make(map[string]DocumentChunk),
	}
}

// Add 添加文档块
func (s *InMemoryVectorStore) Add(ctx context.Context, chunks []DocumentChunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chunk := range chunks {
		s.chunks[chunk.ID] = chunk
	}
	return nil
}

// Search 搜索相似文档块
func (s *InMemoryVectorStore) Search(ctx context.Context, query []float32, topK int) ([]RetrievalResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scoredChunk struct {
		chunk DocumentChunk
		score float32
	}

	scored := make([]scoredChunk, 0, len(s.chunks))

	for _, chunk := range s.chunks {
		if len(chunk.Vector) == 0 {
			continue
		}
		score := cosineSimilarity(query, chunk.Vector)
		scored = append(scored, scoredChunk{chunk: chunk, score: score})
	}

	// 按分数降序排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 返回 top K
	if topK > len(scored) {
		topK = len(scored)
	}

	results := make([]RetrievalResult, topK)
	for i := 0; i < topK; i++ {
		results[i] = RetrievalResult{
			Chunk: scored[i].chunk,
			Score: scored[i].score,
		}
	}

	return results, nil
}

// Delete 删除文档块
func (s *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.chunks, id)
	}
	return nil
}

// Clear 清空存储
func (s *InMemoryVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chunks = make(map[string]DocumentChunk)
	return nil
}

// Size 返回存储的块数量
func (s *InMemoryVectorStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chunks)
}

// cosineSimilarity 计算余弦相似度
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

// generateID 生成唯一 ID
func generateID() string {
	return uuid.New().String()
}

// generateChunkID 生成分块 ID
func generateChunkID(docID string, index int) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(docID+string(rune(index)))).String()
}

// compile-time interface check
var _ VectorStore = (*InMemoryVectorStore)(nil)
