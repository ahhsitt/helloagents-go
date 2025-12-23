package store

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Memory Document Store
// ============================================================================

// MemoryDocumentStore 内存文档存储
//
// 基于 map 的简单实现，适用于测试和轻量级场景。
type MemoryDocumentStore struct {
	collections map[string]map[string]*Document
	mu          sync.RWMutex
}

// NewMemoryDocumentStore 创建内存文档存储
func NewMemoryDocumentStore() *MemoryDocumentStore {
	return &MemoryDocumentStore{
		collections: make(map[string]map[string]*Document),
	}
}

// Put 存储文档
func (s *MemoryDocumentStore) Put(ctx context.Context, collection string, id string, doc Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collections[collection] == nil {
		s.collections[collection] = make(map[string]*Document)
	}

	doc.ID = id
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = time.Now()

	s.collections[collection][id] = &doc
	return nil
}

// Get 获取文档
func (s *MemoryDocumentStore) Get(ctx context.Context, collection string, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collections[collection] == nil {
		return nil, ErrNotFound
	}

	doc, exists := s.collections[collection][id]
	if !exists {
		return nil, ErrNotFound
	}

	return doc, nil
}

// Delete 删除文档
func (s *MemoryDocumentStore) Delete(ctx context.Context, collection string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collections[collection] == nil {
		return ErrNotFound
	}

	if _, exists := s.collections[collection][id]; !exists {
		return ErrNotFound
	}

	delete(s.collections[collection], id)
	return nil
}

// Query 条件查询
func (s *MemoryDocumentStore) Query(ctx context.Context, collection string, filter Filter, opts ...QueryOption) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	options := &queryOptions{limit: 100}
	for _, opt := range opts {
		opt(options)
	}

	if s.collections[collection] == nil {
		return nil, nil
	}

	var results []Document
	for _, doc := range s.collections[collection] {
		if s.matchFilter(doc, filter) {
			results = append(results, *doc)
		}
	}

	// 排序
	if options.orderBy != "" {
		sort.Slice(results, func(i, j int) bool {
			vi := s.getFieldValue(&results[i], options.orderBy)
			vj := s.getFieldValue(&results[j], options.orderBy)
			if options.desc {
				return s.compareValues(vi, vj) > 0
			}
			return s.compareValues(vi, vj) < 0
		})
	}

	// 分页
	if options.offset > 0 && options.offset < len(results) {
		results = results[options.offset:]
	}
	if options.limit > 0 && options.limit < len(results) {
		results = results[:options.limit]
	}

	return results, nil
}

// Count 统计数量
func (s *MemoryDocumentStore) Count(ctx context.Context, collection string, filter Filter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collections[collection] == nil {
		return 0, nil
	}

	count := 0
	for _, doc := range s.collections[collection] {
		if s.matchFilter(doc, filter) {
			count++
		}
	}

	return count, nil
}

// Clear 清空集合
func (s *MemoryDocumentStore) Clear(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.collections[collection] = make(map[string]*Document)
	return nil
}

// Close 关闭连接
func (s *MemoryDocumentStore) Close() error {
	return nil
}

// matchFilter 检查文档是否匹配过滤条件
func (s *MemoryDocumentStore) matchFilter(doc *Document, filter Filter) bool {
	// 空过滤器匹配所有
	if filter.Field == "" && len(filter.And) == 0 && len(filter.Or) == 0 {
		return true
	}

	// 处理 And 条件
	if len(filter.And) > 0 {
		for _, f := range filter.And {
			if !s.matchFilter(doc, f) {
				return false
			}
		}
		return true
	}

	// 处理 Or 条件
	if len(filter.Or) > 0 {
		for _, f := range filter.Or {
			if s.matchFilter(doc, f) {
				return true
			}
		}
		return false
	}

	// 处理单个条件
	value := s.getFieldValue(doc, filter.Field)
	return s.matchCondition(value, filter.Op, filter.Value)
}

// getFieldValue 获取文档字段值
func (s *MemoryDocumentStore) getFieldValue(doc *Document, field string) interface{} {
	switch field {
	case "id":
		return doc.ID
	case "content":
		return doc.Content
	case "created_at":
		return doc.CreatedAt
	case "updated_at":
		return doc.UpdatedAt
	default:
		if doc.Metadata != nil {
			return doc.Metadata[field]
		}
		return nil
	}
}

// matchCondition 匹配条件
func (s *MemoryDocumentStore) matchCondition(value interface{}, op string, target interface{}) bool {
	switch op {
	case "eq", "":
		return s.compareValues(value, target) == 0
	case "ne":
		return s.compareValues(value, target) != 0
	case "gt":
		return s.compareValues(value, target) > 0
	case "gte":
		return s.compareValues(value, target) >= 0
	case "lt":
		return s.compareValues(value, target) < 0
	case "lte":
		return s.compareValues(value, target) <= 0
	case "contains":
		if str, ok := value.(string); ok {
			if tgt, ok := target.(string); ok {
				return strings.Contains(strings.ToLower(str), strings.ToLower(tgt))
			}
		}
		return false
	case "in":
		if arr, ok := target.([]interface{}); ok {
			for _, v := range arr {
				if s.compareValues(value, v) == 0 {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// compareValues 比较两个值
func (s *MemoryDocumentStore) compareValues(a, b interface{}) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
	case int:
		if bv, ok := b.(int); ok {
			return av - bv
		}
	case int64:
		if bv, ok := b.(int64); ok {
			if av < bv {
				return -1
			} else if av > bv {
				return 1
			}
			return 0
		}
	case float64:
		if bv, ok := b.(float64); ok {
			if av < bv {
				return -1
			} else if av > bv {
				return 1
			}
			return 0
		}
	case float32:
		if bv, ok := b.(float32); ok {
			if av < bv {
				return -1
			} else if av > bv {
				return 1
			}
			return 0
		}
	case time.Time:
		if bv, ok := b.(time.Time); ok {
			if av.Before(bv) {
				return -1
			} else if av.After(bv) {
				return 1
			}
			return 0
		}
	}

	return 0
}

// Compile-time interface check
var _ DocumentStore = (*MemoryDocumentStore)(nil)

// ============================================================================
// Memory Vector Store
// ============================================================================

// MemoryVectorStore 内存向量存储
//
// 基于暴力搜索的简单实现，适用于测试和小数据量场景。
type MemoryVectorStore struct {
	collections map[string][]VectorRecord
	mu          sync.RWMutex
}

// NewMemoryVectorStore 创建内存向量存储
func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{
		collections: make(map[string][]VectorRecord),
	}
}

// AddVectors 批量添加向量
func (s *MemoryVectorStore) AddVectors(ctx context.Context, collection string, vectors []VectorRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collections[collection] == nil {
		s.collections[collection] = make([]VectorRecord, 0)
	}

	for _, v := range vectors {
		// 检查是否已存在，如果存在则更新
		found := false
		for i, existing := range s.collections[collection] {
			if existing.ID == v.ID {
				s.collections[collection][i] = v
				found = true
				break
			}
		}
		if !found {
			s.collections[collection] = append(s.collections[collection], v)
		}
	}

	return nil
}

// SearchSimilar 相似度搜索
func (s *MemoryVectorStore) SearchSimilar(ctx context.Context, collection string, vector []float32, topK int, filter *VectorFilter) ([]VectorSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collections[collection] == nil {
		return nil, nil
	}

	type scoredRecord struct {
		record VectorRecord
		score  float32
	}

	var scored []scoredRecord
	for _, rec := range s.collections[collection] {
		// 应用过滤器
		if filter != nil && !s.matchVectorFilter(rec, filter) {
			continue
		}

		score := cosineSimilarity(vector, rec.Vector)
		scored = append(scored, scoredRecord{record: rec, score: score})
	}

	// 按得分排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 限制返回数量
	if topK > 0 && topK < len(scored) {
		scored = scored[:topK]
	}

	results := make([]VectorSearchResult, len(scored))
	for i, s := range scored {
		results[i] = VectorSearchResult{
			ID:       s.record.ID,
			Score:    s.score,
			Payload:  s.record.Payload,
			MemoryID: s.record.MemoryID,
		}
	}

	return results, nil
}

// DeleteVectors 按 ID 删除向量
func (s *MemoryVectorStore) DeleteVectors(ctx context.Context, collection string, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collections[collection] == nil {
		return nil
	}

	idSet := make(map[string]struct{})
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	var remaining []VectorRecord
	for _, rec := range s.collections[collection] {
		if _, found := idSet[rec.ID]; !found {
			remaining = append(remaining, rec)
		}
	}

	s.collections[collection] = remaining
	return nil
}

// DeleteByFilter 按条件删除
func (s *MemoryVectorStore) DeleteByFilter(ctx context.Context, collection string, filter *VectorFilter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collections[collection] == nil {
		return nil
	}

	var remaining []VectorRecord
	for _, rec := range s.collections[collection] {
		if !s.matchVectorFilter(rec, filter) {
			remaining = append(remaining, rec)
		}
	}

	s.collections[collection] = remaining
	return nil
}

// Clear 清空集合
func (s *MemoryVectorStore) Clear(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.collections[collection] = make([]VectorRecord, 0)
	return nil
}

// GetStats 获取统计信息
func (s *MemoryVectorStore) GetStats(ctx context.Context, collection string) (*VectorStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.collections[collection]
	if len(records) == 0 {
		return &VectorStoreStats{}, nil
	}

	dims := 0
	if len(records) > 0 && len(records[0].Vector) > 0 {
		dims = len(records[0].Vector)
	}

	return &VectorStoreStats{
		VectorCount:  len(records),
		Dimensions:   dims,
		IndexedCount: len(records),
	}, nil
}

// HealthCheck 健康检查
func (s *MemoryVectorStore) HealthCheck(ctx context.Context) error {
	return nil
}

// Close 关闭连接
func (s *MemoryVectorStore) Close() error {
	return nil
}

// matchVectorFilter 匹配向量过滤器
func (s *MemoryVectorStore) matchVectorFilter(rec VectorRecord, filter *VectorFilter) bool {
	if filter == nil {
		return true
	}

	if filter.MemoryID != "" && rec.MemoryID != filter.MemoryID {
		return false
	}

	if filter.UserID != "" {
		if rec.Payload == nil {
			return false
		}
		if uid, ok := rec.Payload["user_id"].(string); ok && uid != filter.UserID {
			return false
		}
	}

	if filter.MemoryType != "" {
		if rec.Payload == nil {
			return false
		}
		if mt, ok := rec.Payload["memory_type"].(string); ok && mt != filter.MemoryType {
			return false
		}
	}

	// 自定义条件
	for key, value := range filter.Conditions {
		if rec.Payload == nil {
			return false
		}
		if rec.Payload[key] != value {
			return false
		}
	}

	return true
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

// Compile-time interface check
var _ VectorStore = (*MemoryVectorStore)(nil)

// ============================================================================
// Memory Graph Store
// ============================================================================

// MemoryGraphStore 内存图存储
//
// 基于 map 的简单实现，适用于测试和轻量级场景。
type MemoryGraphStore struct {
	entities  map[string]*GraphEntity
	relations map[string]*GraphRelation
	// 索引
	nameIndex     map[string]string // name (lowercase) -> id
	relationIndex map[string][]string // entityID -> relationIDs
	mu            sync.RWMutex
}

// NewMemoryGraphStore 创建内存图存储
func NewMemoryGraphStore() *MemoryGraphStore {
	return &MemoryGraphStore{
		entities:      make(map[string]*GraphEntity),
		relations:     make(map[string]*GraphRelation),
		nameIndex:     make(map[string]string),
		relationIndex: make(map[string][]string),
	}
}

// AddEntity 添加/更新实体节点
func (s *MemoryGraphStore) AddEntity(ctx context.Context, entity *GraphEntity) error {
	if entity == nil || entity.ID == "" {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在同名实体
	nameLower := strings.ToLower(entity.Name)
	if existingID, ok := s.nameIndex[nameLower]; ok && existingID != entity.ID {
		// 更新现有实体的频率
		if existing, ok := s.entities[existingID]; ok {
			existing.Frequency++
			existing.UpdatedAt = time.Now()
			return nil
		}
	}

	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = time.Now()
	}
	entity.UpdatedAt = time.Now()

	s.entities[entity.ID] = entity
	s.nameIndex[nameLower] = entity.ID

	return nil
}

// GetEntity 获取实体
func (s *MemoryGraphStore) GetEntity(ctx context.Context, id string) (*GraphEntity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entity, ok := s.entities[id]
	if !ok {
		return nil, ErrNotFound
	}

	return entity, nil
}

// SearchEntities 按名称模式搜索实体
func (s *MemoryGraphStore) SearchEntities(ctx context.Context, pattern string, entityType string, limit int) ([]*GraphEntity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	patternLower := strings.ToLower(pattern)
	var results []*GraphEntity

	for _, entity := range s.entities {
		// 名称匹配
		if !strings.Contains(strings.ToLower(entity.Name), patternLower) {
			continue
		}
		// 类型匹配
		if entityType != "" && entity.Type != entityType {
			continue
		}

		results = append(results, entity)

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	// 按频率排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Frequency > results[j].Frequency
	})

	return results, nil
}

// DeleteEntity 删除实体及其关系
func (s *MemoryGraphStore) DeleteEntity(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entity, ok := s.entities[id]
	if !ok {
		return ErrNotFound
	}

	// 删除相关关系
	if relIDs, ok := s.relationIndex[id]; ok {
		for _, relID := range relIDs {
			delete(s.relations, relID)
		}
		delete(s.relationIndex, id)
	}

	// 清理其他实体的关系索引
	for entityID, relIDs := range s.relationIndex {
		var remaining []string
		for _, relID := range relIDs {
			if rel, ok := s.relations[relID]; ok {
				if rel.FromEntityID != id && rel.ToEntityID != id {
					remaining = append(remaining, relID)
				}
			}
		}
		s.relationIndex[entityID] = remaining
	}

	// 从索引中删除
	delete(s.nameIndex, strings.ToLower(entity.Name))
	delete(s.entities, id)

	return nil
}

// AddRelation 添加/更新关系
func (s *MemoryGraphStore) AddRelation(ctx context.Context, relation *GraphRelation) error {
	if relation == nil || relation.ID == "" || relation.FromEntityID == "" || relation.ToEntityID == "" {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证实体存在
	if _, ok := s.entities[relation.FromEntityID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.entities[relation.ToEntityID]; !ok {
		return ErrNotFound
	}

	if relation.CreatedAt.IsZero() {
		relation.CreatedAt = time.Now()
	}
	relation.UpdatedAt = time.Now()

	s.relations[relation.ID] = relation

	// 更新索引
	s.relationIndex[relation.FromEntityID] = append(s.relationIndex[relation.FromEntityID], relation.ID)
	s.relationIndex[relation.ToEntityID] = append(s.relationIndex[relation.ToEntityID], relation.ID)

	return nil
}

// GetRelations 获取实体的所有关系
func (s *MemoryGraphStore) GetRelations(ctx context.Context, entityID string) ([]*GraphRelation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relIDs := s.relationIndex[entityID]
	var results []*GraphRelation

	for _, relID := range relIDs {
		if rel, ok := s.relations[relID]; ok {
			results = append(results, rel)
		}
	}

	return results, nil
}

// FindRelatedEntities 图遍历查找相关实体
func (s *MemoryGraphStore) FindRelatedEntities(ctx context.Context, entityID string, relationType string, maxDepth int) ([]*GraphTraversalResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.entities[entityID]; !ok {
		return nil, ErrNotFound
	}

	if maxDepth <= 0 {
		maxDepth = 2
	}

	results := make([]*GraphTraversalResult, 0)
	visited := make(map[string]struct{})
	visited[entityID] = struct{}{}

	// BFS 遍历
	type queueItem struct {
		id    string
		depth int
		path  []*GraphRelation
	}

	queue := []queueItem{{id: entityID, depth: 0, path: nil}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// 查找所有相关关系
		for _, rel := range s.relations {
			// 类型过滤
			if relationType != "" && rel.Type != relationType {
				continue
			}

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

			entity := s.entities[nextID]
			if entity == nil {
				continue
			}

			newPath := make([]*GraphRelation, len(current.path)+1)
			copy(newPath, current.path)
			newPath[len(current.path)] = rel

			score := rel.Strength / float32(current.depth+1)
			results = append(results, &GraphTraversalResult{
				Entity:   entity,
				Depth:    current.depth + 1,
				Path:     newPath,
				Score:    score,
				Relation: rel,
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

// GetShortestPath 最短路径查询
func (s *MemoryGraphStore) GetShortestPath(ctx context.Context, fromID, toID string) ([]*GraphEntity, []*GraphRelation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.entities[fromID]; !ok {
		return nil, nil, ErrNotFound
	}
	if _, ok := s.entities[toID]; !ok {
		return nil, nil, ErrNotFound
	}

	if fromID == toID {
		return []*GraphEntity{s.entities[fromID]}, nil, nil
	}

	// BFS 查找最短路径
	type pathItem struct {
		entityID  string
		entities  []*GraphEntity
		relations []*GraphRelation
	}

	visited := make(map[string]struct{})
	visited[fromID] = struct{}{}

	queue := []pathItem{{
		entityID:  fromID,
		entities:  []*GraphEntity{s.entities[fromID]},
		relations: nil,
	}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, rel := range s.relations {
			var nextID string
			if rel.FromEntityID == current.entityID {
				nextID = rel.ToEntityID
			} else if rel.ToEntityID == current.entityID {
				nextID = rel.FromEntityID
			} else {
				continue
			}

			if _, seen := visited[nextID]; seen {
				continue
			}
			visited[nextID] = struct{}{}

			entity := s.entities[nextID]
			if entity == nil {
				continue
			}

			newEntities := make([]*GraphEntity, len(current.entities)+1)
			copy(newEntities, current.entities)
			newEntities[len(current.entities)] = entity

			newRelations := make([]*GraphRelation, len(current.relations)+1)
			copy(newRelations, current.relations)
			newRelations[len(current.relations)] = rel

			if nextID == toID {
				return newEntities, newRelations, nil
			}

			queue = append(queue, pathItem{
				entityID:  nextID,
				entities:  newEntities,
				relations: newRelations,
			})
		}
	}

	return nil, nil, ErrNotFound
}

// DeleteRelation 删除关系
func (s *MemoryGraphStore) DeleteRelation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rel, ok := s.relations[id]
	if !ok {
		return ErrNotFound
	}

	// 更新索引
	s.removeFromRelationIndex(rel.FromEntityID, id)
	s.removeFromRelationIndex(rel.ToEntityID, id)

	delete(s.relations, id)
	return nil
}

// removeFromRelationIndex 从关系索引中移除
func (s *MemoryGraphStore) removeFromRelationIndex(entityID, relID string) {
	relIDs := s.relationIndex[entityID]
	var remaining []string
	for _, rid := range relIDs {
		if rid != relID {
			remaining = append(remaining, rid)
		}
	}
	s.relationIndex[entityID] = remaining
}

// Clear 清空所有数据
func (s *MemoryGraphStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entities = make(map[string]*GraphEntity)
	s.relations = make(map[string]*GraphRelation)
	s.nameIndex = make(map[string]string)
	s.relationIndex = make(map[string][]string)

	return nil
}

// GetStats 获取统计信息
func (s *MemoryGraphStore) GetStats(ctx context.Context) (*GraphStoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entityTypes := make(map[string]int)
	for _, e := range s.entities {
		entityTypes[e.Type]++
	}

	relationTypes := make(map[string]int)
	for _, r := range s.relations {
		relationTypes[r.Type]++
	}

	return &GraphStoreStats{
		EntityCount:   len(s.entities),
		RelationCount: len(s.relations),
		EntityTypes:   entityTypes,
		RelationTypes: relationTypes,
	}, nil
}

// Close 关闭连接
func (s *MemoryGraphStore) Close() error {
	return nil
}

// Compile-time interface check
var _ GraphStore = (*MemoryGraphStore)(nil)
