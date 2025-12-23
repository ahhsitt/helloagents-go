package memory

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SemanticMemoryStore 语义记忆存储实现
//
// 基于向量相似度的记忆存储，支持语义搜索、实体管理和图遍历。
// 注意：这是一个简化的内存实现，生产环境建议使用专用向量数据库和图数据库。
type SemanticMemoryStore struct {
	embedder Embedder
	records  []semanticRecord
	tfidf    *TFIDFVectorizer // 本地 TF-IDF 回退

	// 实体和关系存储
	entities    map[string]*Entity   // entityID -> Entity
	relations   map[string]*Relation // relationID -> Relation
	entityIndex map[string]string    // entityName (lowercase) -> entityID

	mu sync.RWMutex
}

type semanticRecord struct {
	ID         string
	Content    string
	Vector     []float32
	TFIDFVec   []float32
	Metadata   map[string]interface{}
	Importance float32
	Timestamp  time.Time
	UserID     string
}

// NewSemanticMemory 创建语义记忆存储
func NewSemanticMemory(embedder Embedder) *SemanticMemoryStore {
	return &SemanticMemoryStore{
		embedder:    embedder,
		records:     make([]semanticRecord, 0),
		tfidf:       NewTFIDFVectorizer(),
		entities:    make(map[string]*Entity),
		relations:   make(map[string]*Relation),
		entityIndex: make(map[string]string),
	}
}

// Store 存储文本及其向量
func (m *SemanticMemoryStore) Store(ctx context.Context, id string, content string, metadata map[string]interface{}) error {
	// 生成 ID（如果未提供）
	if id == "" {
		id = uuid.New().String()
	}

	// 生成嵌入向量
	var vector []float32
	if m.embedder != nil {
		vectors, err := m.embedder.Embed(ctx, []string{content})
		if err != nil {
			return err
		}
		if len(vectors) == 0 {
			return ErrEmbeddingFailed
		}
		vector = vectors[0]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成 TF-IDF 向量
	var tfidfVec []float32
	if m.tfidf != nil {
		tfidfVec = m.tfidf.Transform(content)
	}

	// 提取 importance 和其他字段
	var importance float32 = 0.5
	var userID string
	if metadata != nil {
		if imp, ok := metadata["importance"].(float32); ok {
			importance = imp
		} else if imp, ok := metadata["importance"].(float64); ok {
			importance = float32(imp)
		}
		if uid, ok := metadata["user_id"].(string); ok {
			userID = uid
		}
	}

	// 检查是否已存在，如果存在则更新
	for i, rec := range m.records {
		if rec.ID == id {
			m.records[i] = semanticRecord{
				ID:         id,
				Content:    content,
				Vector:     vector,
				TFIDFVec:   tfidfVec,
				Metadata:   metadata,
				Importance: importance,
				Timestamp:  time.Now(),
				UserID:     userID,
			}
			m.rebuildTFIDF()
			return nil
		}
	}

	// 添加新记录
	m.records = append(m.records, semanticRecord{
		ID:         id,
		Content:    content,
		Vector:     vector,
		TFIDFVec:   tfidfVec,
		Metadata:   metadata,
		Importance: importance,
		Timestamp:  time.Now(),
		UserID:     userID,
	})

	m.rebuildTFIDF()
	return nil
}

// rebuildTFIDF 重建 TF-IDF 向量化器
func (m *SemanticMemoryStore) rebuildTFIDF() {
	if len(m.records) == 0 {
		m.tfidf.Clear()
		return
	}

	docs := make([]string, len(m.records))
	for i, rec := range m.records {
		docs[i] = rec.Content
	}

	vectors := m.tfidf.FitTransform(docs)
	for i := range m.records {
		if i < len(vectors) {
			m.records[i].TFIDFVec = vectors[i]
		}
	}
}

// Search 搜索相似内容
func (m *SemanticMemoryStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.records) == 0 {
		return nil, nil
	}

	// 尝试嵌入向量检索
	var results []SearchResult
	if m.embedder != nil {
		vectors, err := m.embedder.Embed(ctx, []string{query})
		if err == nil && len(vectors) > 0 {
			results = m.vectorSearch(vectors[0], topK)
		}
	}

	// 如果嵌入失败或结果不足，使用 TF-IDF
	if len(results) < topK && m.tfidf.VocabularySize() > 0 {
		tfidfResults := m.tfidfSearch(query, topK)
		results = m.mergeResults(results, tfidfResults, topK)
	}

	// 如果仍然不足，使用关键词匹配
	if len(results) < topK {
		keywordResults := m.keywordSearch(query, topK)
		results = m.mergeResults(results, keywordResults, topK)
	}

	return results, nil
}

// vectorSearch 向量相似度搜索
func (m *SemanticMemoryStore) vectorSearch(queryVector []float32, topK int) []SearchResult {
	type scoredRecord struct {
		record semanticRecord
		score  float32
	}

	scored := make([]scoredRecord, 0, len(m.records))
	now := time.Now()

	for _, rec := range m.records {
		if rec.Vector == nil {
			continue
		}
		similarity := cosineSimilarity(queryVector, rec.Vector)
		ageDays := float32(now.Sub(rec.Timestamp).Hours() / 24)
		score := m.calculateScore(similarity, ageDays, rec.Importance)
		scored = append(scored, scoredRecord{record: rec, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

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
	return results
}

// tfidfSearch TF-IDF 语义检索
func (m *SemanticMemoryStore) tfidfSearch(query string, topK int) []SearchResult {
	queryVector := m.tfidf.Transform(query)
	if queryVector == nil {
		return nil
	}

	type scoredRecord struct {
		record semanticRecord
		score  float32
	}

	scored := make([]scoredRecord, 0, len(m.records))
	now := time.Now()

	for _, rec := range m.records {
		if rec.TFIDFVec == nil {
			continue
		}
		similarity := m.tfidf.CosineSimilarity(queryVector, rec.TFIDFVec)
		ageDays := float32(now.Sub(rec.Timestamp).Hours() / 24)
		score := m.calculateScore(similarity, ageDays, rec.Importance)
		scored = append(scored, scoredRecord{record: rec, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

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
	return results
}

// keywordSearch 关键词匹配检索
func (m *SemanticMemoryStore) keywordSearch(query string, topK int) []SearchResult {
	query = strings.ToLower(query)
	keywords := strings.Fields(query)
	if len(keywords) == 0 {
		return nil
	}

	type scoredRecord struct {
		record semanticRecord
		score  float32
	}

	scored := make([]scoredRecord, 0, len(m.records))
	now := time.Now()

	for _, rec := range m.records {
		content := strings.ToLower(rec.Content)
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				matchCount++
			}
		}

		if matchCount > 0 {
			similarity := float32(matchCount) / float32(len(keywords))
			ageDays := float32(now.Sub(rec.Timestamp).Hours() / 24)
			score := m.calculateScore(similarity, ageDays, rec.Importance)
			scored = append(scored, scoredRecord{record: rec, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

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
	return results
}

// calculateScore 计算综合得分
//
// 公式：(相似度 × 0.7 + 时间近因性 × 0.3) × (0.8 + 重要性 × 0.4)
func (m *SemanticMemoryStore) calculateScore(similarity, ageDays, importance float32) float32 {
	recency := 1.0 / (1.0 + ageDays)
	mixedScore := similarity*0.7 + recency*0.3
	importanceWeight := 0.8 + importance*0.4
	return mixedScore * importanceWeight
}

// mergeResults 合并并去重结果
func (m *SemanticMemoryStore) mergeResults(a, b []SearchResult, limit int) []SearchResult {
	seen := make(map[string]struct{})
	merged := make([]SearchResult, 0, len(a)+len(b))

	for _, r := range a {
		if _, ok := seen[r.ID]; !ok {
			seen[r.ID] = struct{}{}
			merged = append(merged, r)
		}
	}
	for _, r := range b {
		if _, ok := seen[r.ID]; !ok {
			seen[r.ID] = struct{}{}
			merged = append(merged, r)
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
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
			m.rebuildTFIDF()
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
	m.entities = make(map[string]*Entity)
	m.relations = make(map[string]*Relation)
	m.entityIndex = make(map[string]string)
	m.tfidf.Clear()
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
var _ Memory = (*SemanticMemoryStore)(nil)

// ============================================================================
// Memory 接口实现
// ============================================================================

// Add 添加记忆项（实现 Memory 接口）
func (m *SemanticMemoryStore) Add(ctx context.Context, item *MemoryItem) (string, error) {
	if err := item.Validate(); err != nil {
		return "", err
	}

	metadata := item.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["importance"] = item.Importance
	metadata["user_id"] = item.UserID

	if err := m.Store(ctx, item.ID, item.Content, metadata); err != nil {
		return "", err
	}

	return item.ID, nil
}

// Retrieve 检索记忆（实现 Memory 接口）
func (m *SemanticMemoryStore) Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error) {
	options := &retrieveOptions{
		limit: 10,
	}
	for _, opt := range opts {
		opt(options)
	}

	results, err := m.Search(ctx, query, options.limit)
	if err != nil {
		return nil, err
	}

	items := make([]*MemoryItem, 0, len(results))
	for _, r := range results {
		if options.minScore > 0 && r.Score < options.minScore {
			continue
		}
		item := m.resultToItem(r)
		items = append(items, item)
	}

	return items, nil
}

// resultToItem 将搜索结果转换为 MemoryItem
func (m *SemanticMemoryStore) resultToItem(r SearchResult) *MemoryItem {
	metadata := r.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["score"] = r.Score

	var importance float32 = 0.5
	if imp, ok := metadata["importance"].(float32); ok {
		importance = imp
	} else if imp, ok := metadata["importance"].(float64); ok {
		importance = float32(imp)
	}

	var userID string
	if uid, ok := metadata["user_id"].(string); ok {
		userID = uid
	}

	return &MemoryItem{
		ID:         r.ID,
		Content:    r.Content,
		MemoryType: MemoryTypeSemantic,
		UserID:     userID,
		Timestamp:  time.Now(),
		Importance: importance,
		Metadata:   metadata,
	}
}

// Update 更新记忆（实现 Memory 接口）
func (m *SemanticMemoryStore) Update(ctx context.Context, id string, opts ...UpdateOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &updateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	for i := range m.records {
		if m.records[i].ID == id {
			if options.content != nil {
				m.records[i].Content = *options.content
				// 重新生成向量
				if m.embedder != nil {
					vectors, err := m.embedder.Embed(ctx, []string{*options.content})
					if err == nil && len(vectors) > 0 {
						m.records[i].Vector = vectors[0]
					}
				}
			}
			if options.importance != nil {
				m.records[i].Importance = *options.importance
			}
			if options.metadata != nil {
				m.records[i].Metadata = options.metadata
			}
			m.rebuildTFIDF()
			return nil
		}
	}

	return ErrNotFound
}

// Remove 删除记忆（实现 Memory 接口）
func (m *SemanticMemoryStore) Remove(ctx context.Context, id string) error {
	return m.Delete(ctx, id)
}

// Has 检查记忆是否存在（实现 Memory 接口）
func (m *SemanticMemoryStore) Has(ctx context.Context, id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, rec := range m.records {
		if rec.ID == id {
			return true
		}
	}
	return false
}

// GetStats 获取统计信息（实现 Memory 接口）
func (m *SemanticMemoryStore) GetStats(ctx context.Context) (*MemoryStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.records) == 0 {
		return &MemoryStats{Count: 0}, nil
	}

	var totalImportance float32
	var oldestTs, newestTs time.Time

	for i, rec := range m.records {
		totalImportance += rec.Importance
		if i == 0 || rec.Timestamp.Before(oldestTs) {
			oldestTs = rec.Timestamp
		}
		if i == 0 || rec.Timestamp.After(newestTs) {
			newestTs = rec.Timestamp
		}
	}

	return &MemoryStats{
		Count:           len(m.records),
		OldestTimestamp: oldestTs.UnixMilli(),
		NewestTimestamp: newestTs.UnixMilli(),
		AvgImportance:   totalImportance / float32(len(m.records)),
	}, nil
}

// ============================================================================
// 实体管理
// ============================================================================

// AddEntity 添加实体
func (m *SemanticMemoryStore) AddEntity(ctx context.Context, entity *Entity) error {
	if entity == nil || entity.Name == "" {
		return ErrInvalidInput
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在同名实体
	nameLower := strings.ToLower(entity.Name)
	if existingID, ok := m.entityIndex[nameLower]; ok {
		// 更新频率
		if existing, ok := m.entities[existingID]; ok {
			existing.IncrementFrequency()
			return nil
		}
	}

	// 生成 ID（如果未提供）
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}

	// 生成向量（如果有嵌入器）
	if m.embedder != nil && entity.Vector == nil {
		desc := entity.Name
		if entity.Description != "" {
			desc = entity.Name + ": " + entity.Description
		}
		vectors, err := m.embedder.Embed(ctx, []string{desc})
		if err == nil && len(vectors) > 0 {
			entity.Vector = vectors[0]
		}
	}

	m.entities[entity.ID] = entity
	m.entityIndex[nameLower] = entity.ID
	return nil
}

// GetEntity 获取实体
func (m *SemanticMemoryStore) GetEntity(ctx context.Context, id string) (*Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entity, ok := m.entities[id]; ok {
		return entity, nil
	}
	return nil, ErrNotFound
}

// GetEntityByName 按名称获取实体
func (m *SemanticMemoryStore) GetEntityByName(ctx context.Context, name string) (*Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nameLower := strings.ToLower(name)
	if id, ok := m.entityIndex[nameLower]; ok {
		if entity, ok := m.entities[id]; ok {
			return entity, nil
		}
	}
	return nil, ErrNotFound
}

// SearchEntities 搜索实体
func (m *SemanticMemoryStore) SearchEntities(ctx context.Context, pattern string, limit int) ([]*Entity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	patternLower := strings.ToLower(pattern)
	results := make([]*Entity, 0)

	for _, entity := range m.entities {
		nameLower := strings.ToLower(entity.Name)
		if strings.Contains(nameLower, patternLower) {
			results = append(results, entity)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	// 按频率排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Frequency > results[j].Frequency
	})

	return results, nil
}

// DeleteEntity 删除实体
func (m *SemanticMemoryStore) DeleteEntity(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entity, ok := m.entities[id]
	if !ok {
		return ErrNotFound
	}

	// 删除相关关系
	for relID, rel := range m.relations {
		if rel.FromEntityID == id || rel.ToEntityID == id {
			delete(m.relations, relID)
		}
	}

	// 从索引中删除
	delete(m.entityIndex, strings.ToLower(entity.Name))
	delete(m.entities, id)
	return nil
}

// EntityCount 返回实体数量
func (m *SemanticMemoryStore) EntityCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entities)
}

// ============================================================================
// 关系管理
// ============================================================================

// AddRelation 添加关系
func (m *SemanticMemoryStore) AddRelation(ctx context.Context, relation *Relation) error {
	if relation == nil || relation.FromEntityID == "" || relation.ToEntityID == "" {
		return ErrInvalidInput
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证实体存在
	if _, ok := m.entities[relation.FromEntityID]; !ok {
		return ErrNotFound
	}
	if _, ok := m.entities[relation.ToEntityID]; !ok {
		return ErrNotFound
	}

	// 生成 ID（如果未提供）
	if relation.ID == "" {
		relation.ID = uuid.New().String()
	}

	m.relations[relation.ID] = relation
	return nil
}

// GetRelation 获取关系
func (m *SemanticMemoryStore) GetRelation(ctx context.Context, id string) (*Relation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if rel, ok := m.relations[id]; ok {
		return rel, nil
	}
	return nil, ErrNotFound
}

// GetRelatedEntities 获取相关实体（图遍历）
func (m *SemanticMemoryStore) GetRelatedEntities(ctx context.Context, entityID string, maxDepth int) ([]GraphSearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.entities[entityID]; !ok {
		return nil, ErrNotFound
	}

	if maxDepth <= 0 {
		maxDepth = 2
	}

	results := make([]GraphSearchResult, 0)
	visited := make(map[string]struct{})
	visited[entityID] = struct{}{}

	// BFS 遍历
	type queueItem struct {
		id    string
		depth int
		path  []*Relation
	}

	queue := []queueItem{{id: entityID, depth: 0, path: nil}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// 查找所有相关关系
		for _, rel := range m.relations {
			var nextID string
			if rel.FromEntityID == current.id {
				nextID = rel.ToEntityID
			} else if rel.ToEntityID == current.id {
				nextID = rel.FromEntityID
			} else {
				continue
			}

			if _, seen := visited[nextID]; seen {
				continue
			}
			visited[nextID] = struct{}{}

			entity := m.entities[nextID]
			if entity == nil {
				continue
			}

			newPath := make([]*Relation, len(current.path)+1)
			copy(newPath, current.path)
			newPath[len(current.path)] = rel

			score := rel.Strength / float32(current.depth+1)
			results = append(results, GraphSearchResult{
				Entity: entity,
				Depth:  current.depth + 1,
				Path:   newPath,
				Score:  score,
			})

			queue = append(queue, queueItem{
				id:    nextID,
				depth: current.depth + 1,
				path:  newPath,
			})
		}
	}

	// 按得分排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// DeleteRelation 删除关系
func (m *SemanticMemoryStore) DeleteRelation(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.relations[id]; !ok {
		return ErrNotFound
	}
	delete(m.relations, id)
	return nil
}

// RelationCount 返回关系数量
func (m *SemanticMemoryStore) RelationCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.relations)
}

// ============================================================================
// 实体和关系提取
// ============================================================================

// 预编译正则表达式
var (
	// 人名模式（简化版，支持中英文）
	personPatternEN = regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)+\b`)
	personPatternCN = regexp.MustCompile(`[张李王刘陈杨黄赵周吴徐孙马朱胡郭何高林][一-龥]{1,3}`)

	// 组织模式
	orgPatternEN = regexp.MustCompile(`\b(?:[A-Z][a-z]*\s*)+(?:Inc|Corp|Ltd|LLC|Company|Organization|Group|Foundation|University|Institute)\b`)
	orgPatternCN = regexp.MustCompile(`[一-龥]+(?:公司|集团|组织|协会|基金会|大学|学院|研究院)`)

	// 地点模式
	locationPatternCN = regexp.MustCompile(`[一-龥]+(?:市|省|县|区|镇|村|路|街|国|洲)`)
)

// ExtractEntities 从文本中提取实体
func (m *SemanticMemoryStore) ExtractEntities(content string) []ExtractedEntity {
	entities := make([]ExtractedEntity, 0)
	seen := make(map[string]struct{})

	// 提取人名
	for _, match := range personPatternEN.FindAllStringIndex(content, -1) {
		name := content[match[0]:match[1]]
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       EntityTypePerson,
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.7,
		})
	}

	for _, match := range personPatternCN.FindAllStringIndex(content, -1) {
		name := content[match[0]:match[1]]
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       EntityTypePerson,
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.6,
		})
	}

	// 提取组织
	for _, match := range orgPatternEN.FindAllStringIndex(content, -1) {
		name := content[match[0]:match[1]]
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       EntityTypeOrganization,
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.8,
		})
	}

	for _, match := range orgPatternCN.FindAllStringIndex(content, -1) {
		name := content[match[0]:match[1]]
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       EntityTypeOrganization,
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.7,
		})
	}

	// 提取地点
	for _, match := range locationPatternCN.FindAllStringIndex(content, -1) {
		name := content[match[0]:match[1]]
		if _, ok := seen[strings.ToLower(name)]; ok {
			continue
		}
		seen[strings.ToLower(name)] = struct{}{}
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       EntityTypeLocation,
			StartPos:   match[0],
			EndPos:     match[1],
			Confidence: 0.7,
		})
	}

	return entities
}

// ExtractRelations 从文本中提取关系（基于共现）
func (m *SemanticMemoryStore) ExtractRelations(content string, entities []ExtractedEntity) []ExtractedRelation {
	relations := make([]ExtractedRelation, 0)

	if len(entities) < 2 {
		return relations
	}

	// 按句子分割
	sentences := splitSentences(content)

	for _, sentence := range sentences {
		// 找出该句子中的实体
		var entitiesInSentence []ExtractedEntity
		sentenceLower := strings.ToLower(sentence)

		for _, e := range entities {
			if strings.Contains(sentenceLower, strings.ToLower(e.Name)) {
				entitiesInSentence = append(entitiesInSentence, e)
			}
		}

		// 基于共现生成关系
		for i := 0; i < len(entitiesInSentence); i++ {
			for j := i + 1; j < len(entitiesInSentence); j++ {
				e1, e2 := entitiesInSentence[i], entitiesInSentence[j]

				relType := inferRelationType(e1.Type, e2.Type, sentence)
				confidence := (e1.Confidence + e2.Confidence) / 2 * 0.5 // 共现关系置信度较低

				relations = append(relations, ExtractedRelation{
					FromEntity:   e1.Name,
					ToEntity:     e2.Name,
					RelationType: relType,
					Confidence:   confidence,
					Context:      sentence,
				})
			}
		}
	}

	return relations
}

// splitSentences 分割句子
func splitSentences(text string) []string {
	// 简单的句子分割
	separators := []string{"。", "！", "？", ".", "!", "?", "\n"}
	sentences := []string{text}

	for _, sep := range separators {
		var newSentences []string
		for _, s := range sentences {
			parts := strings.Split(s, sep)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					newSentences = append(newSentences, p)
				}
			}
		}
		sentences = newSentences
	}

	return sentences
}

// inferRelationType 推断关系类型
func inferRelationType(type1, type2 EntityType, context string) RelationType {
	contextLower := strings.ToLower(context)

	// 基于关键词推断
	if strings.Contains(contextLower, "work") || strings.Contains(contextLower, "employ") ||
		strings.Contains(contextLower, "工作") || strings.Contains(contextLower, "任职") {
		if type1 == EntityTypePerson && type2 == EntityTypeOrganization {
			return RelationTypeWorksAt
		}
	}

	if strings.Contains(contextLower, "located") || strings.Contains(contextLower, "in") ||
		strings.Contains(contextLower, "位于") || strings.Contains(contextLower, "在") {
		if type2 == EntityTypeLocation {
			return RelationTypeLocatedIn
		}
	}

	if strings.Contains(contextLower, "know") || strings.Contains(contextLower, "meet") ||
		strings.Contains(contextLower, "认识") || strings.Contains(contextLower, "遇见") {
		if type1 == EntityTypePerson && type2 == EntityTypePerson {
			return RelationTypeKnows
		}
	}

	if strings.Contains(contextLower, "part of") || strings.Contains(contextLower, "belong") ||
		strings.Contains(contextLower, "属于") || strings.Contains(contextLower, "隶属") {
		return RelationTypePartOf
	}

	if strings.Contains(contextLower, "created") || strings.Contains(contextLower, "founded") ||
		strings.Contains(contextLower, "创建") || strings.Contains(contextLower, "创立") {
		return RelationTypeCreatedBy
	}

	return RelationTypeRelatedTo
}

// ============================================================================
// Forget 方法
// ============================================================================

// Forget 执行遗忘
func (m *SemanticMemoryStore) Forget(ctx context.Context, strategy ForgetStrategy, opts ...ForgetOption) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	options := &forgetOptions{}
	for _, opt := range opts {
		opt(options)
	}

	originalCount := len(m.records)
	var remaining []semanticRecord

	switch strategy {
	case ForgetByImportance:
		threshold := options.threshold
		if threshold <= 0 {
			threshold = 0.3
		}
		for _, rec := range m.records {
			if rec.Importance >= threshold {
				remaining = append(remaining, rec)
			}
		}

	case ForgetByTime:
		maxAgeDays := options.maxAgeDays
		if maxAgeDays <= 0 {
			maxAgeDays = 30
		}
		cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
		for _, rec := range m.records {
			if rec.Timestamp.After(cutoff) || rec.Timestamp.Equal(cutoff) {
				remaining = append(remaining, rec)
			}
		}

	case ForgetByCapacity:
		targetCapacity := options.targetCapacity
		if targetCapacity <= 0 {
			targetCapacity = 1000
		}
		if len(m.records) <= targetCapacity {
			return 0, nil
		}
		// 按重要性排序，保留最重要的
		sorted := make([]semanticRecord, len(m.records))
		copy(sorted, m.records)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Importance > sorted[j].Importance
		})
		remaining = sorted[:targetCapacity]
		// 按时间重新排序
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].Timestamp.Before(remaining[j].Timestamp)
		})
	}

	m.records = remaining
	m.rebuildTFIDF()

	return originalCount - len(m.records), nil
}
