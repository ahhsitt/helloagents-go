package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jGraphStore Neo4j 图存储
//
// 基于 Neo4j 的图存储实现，支持实体和关系管理。
type Neo4jGraphStore struct {
	driver neo4j.DriverWithContext
}

// Neo4jConfig Neo4j 配置
type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}

// NewNeo4jGraphStore 创建 Neo4j 图存储
func NewNeo4jGraphStore(config Neo4jConfig) (*Neo4jGraphStore, error) {
	if config.URI == "" {
		config.URI = "bolt://localhost:7687"
	}

	auth := neo4j.NoAuth()
	if config.Username != "" && config.Password != "" {
		auth = neo4j.BasicAuth(config.Username, config.Password, "")
	}

	driver, err := neo4j.NewDriverWithContext(config.URI, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// 验证连接
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify connectivity: %w", err)
	}

	store := &Neo4jGraphStore{driver: driver}

	// 创建索引
	if err := store.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return store, nil
}

// createIndexes 创建索引
func (s *Neo4jGraphStore) createIndexes(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	indexes := []string{
		"CREATE INDEX entity_id IF NOT EXISTS FOR (e:Entity) ON (e.id)",
		"CREATE INDEX entity_name IF NOT EXISTS FOR (e:Entity) ON (e.name)",
		"CREATE INDEX entity_type IF NOT EXISTS FOR (e:Entity) ON (e.type)",
	}

	for _, idx := range indexes {
		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			// 忽略索引已存在的错误
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	return nil
}

// AddEntity 添加/更新实体节点
func (s *Neo4jGraphStore) AddEntity(ctx context.Context, entity *GraphEntity) error {
	if entity == nil || entity.ID == "" {
		return ErrInvalidInput
	}

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	now := time.Now().UnixMilli()

	query := `
	MERGE (e:Entity {id: $id})
	ON CREATE SET
		e.name = $name,
		e.type = $type,
		e.description = $description,
		e.frequency = $frequency,
		e.created_at = $now,
		e.updated_at = $now
	ON MATCH SET
		e.name = $name,
		e.type = $type,
		e.description = $description,
		e.frequency = e.frequency + 1,
		e.updated_at = $now
	`

	params := map[string]interface{}{
		"id":          entity.ID,
		"name":        entity.Name,
		"type":        entity.Type,
		"description": entity.Description,
		"frequency":   entity.Frequency,
		"now":         now,
	}

	_, err := session.Run(ctx, query, params)
	return err
}

// GetEntity 获取实体
func (s *Neo4jGraphStore) GetEntity(ctx context.Context, id string) (*GraphEntity, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `MATCH (e:Entity {id: $id}) RETURN e`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}

	if result.Next(ctx) {
		record := result.Record()
		nodeVal, _ := record.Get("e")
		node := nodeVal.(neo4j.Node)
		return s.nodeToEntity(node), nil
	}

	return nil, ErrNotFound
}

// SearchEntities 按名称模式搜索实体
func (s *Neo4jGraphStore) SearchEntities(ctx context.Context, pattern string, entityType string, limit int) ([]*GraphEntity, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	var query string
	params := map[string]interface{}{
		"pattern": "(?i).*" + pattern + ".*",
		"limit":   limit,
	}

	if entityType != "" {
		query = `MATCH (e:Entity) WHERE e.name =~ $pattern AND e.type = $type RETURN e ORDER BY e.frequency DESC LIMIT $limit`
		params["type"] = entityType
	} else {
		query = `MATCH (e:Entity) WHERE e.name =~ $pattern RETURN e ORDER BY e.frequency DESC LIMIT $limit`
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var entities []*GraphEntity
	for result.Next(ctx) {
		record := result.Record()
		nodeVal, _ := record.Get("e")
		node := nodeVal.(neo4j.Node)
		entities = append(entities, s.nodeToEntity(node))
	}

	return entities, nil
}

// DeleteEntity 删除实体及其关系
func (s *Neo4jGraphStore) DeleteEntity(ctx context.Context, id string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `MATCH (e:Entity {id: $id}) DETACH DELETE e`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return err
	}

	summary, err := result.Consume(ctx)
	if err != nil {
		return err
	}

	if summary.Counters().NodesDeleted() == 0 {
		return ErrNotFound
	}

	return nil
}

// AddRelation 添加/更新关系
func (s *Neo4jGraphStore) AddRelation(ctx context.Context, relation *GraphRelation) error {
	if relation == nil || relation.ID == "" || relation.FromEntityID == "" || relation.ToEntityID == "" {
		return ErrInvalidInput
	}

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	now := time.Now().UnixMilli()

	// 先检查实体是否存在
	checkQuery := `
	MATCH (from:Entity {id: $fromId}), (to:Entity {id: $toId})
	RETURN from, to
	`

	result, err := session.Run(ctx, checkQuery, map[string]interface{}{
		"fromId": relation.FromEntityID,
		"toId":   relation.ToEntityID,
	})
	if err != nil {
		return err
	}

	if !result.Next(ctx) {
		return ErrNotFound
	}

	// 创建关系
	query := fmt.Sprintf(`
	MATCH (from:Entity {id: $fromId}), (to:Entity {id: $toId})
	MERGE (from)-[r:%s {id: $id}]->(to)
	ON CREATE SET
		r.strength = $strength,
		r.created_at = $now,
		r.updated_at = $now
	ON MATCH SET
		r.strength = $strength,
		r.updated_at = $now
	`, s.sanitizeRelationType(relation.Type))

	params := map[string]interface{}{
		"id":       relation.ID,
		"fromId":   relation.FromEntityID,
		"toId":     relation.ToEntityID,
		"strength": relation.Strength,
		"now":      now,
	}

	_, err = session.Run(ctx, query, params)
	return err
}

// GetRelations 获取实体的所有关系
func (s *Neo4jGraphStore) GetRelations(ctx context.Context, entityID string) ([]*GraphRelation, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
	MATCH (e:Entity {id: $id})-[r]-(other:Entity)
	RETURN r, startNode(r).id as fromId, endNode(r).id as toId, type(r) as relType
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": entityID})
	if err != nil {
		return nil, err
	}

	var relations []*GraphRelation
	for result.Next(ctx) {
		record := result.Record()
		relVal, _ := record.Get("r")
		rel := relVal.(neo4j.Relationship)

		fromID, _ := record.Get("fromId")
		toID, _ := record.Get("toId")
		relType, _ := record.Get("relType")

		relations = append(relations, &GraphRelation{
			ID:           s.getStringProp(rel.Props, "id"),
			FromEntityID: fromID.(string),
			ToEntityID:   toID.(string),
			Type:         relType.(string),
			Strength:     s.getFloat32Prop(rel.Props, "strength"),
			CreatedAt:    time.UnixMilli(s.getInt64Prop(rel.Props, "created_at")),
			UpdatedAt:    time.UnixMilli(s.getInt64Prop(rel.Props, "updated_at")),
		})
	}

	return relations, nil
}

// FindRelatedEntities 图遍历查找相关实体
func (s *Neo4jGraphStore) FindRelatedEntities(ctx context.Context, entityID string, relationType string, maxDepth int) ([]*GraphTraversalResult, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	if maxDepth <= 0 {
		maxDepth = 2
	}

	var query string
	params := map[string]interface{}{
		"id":       entityID,
		"maxDepth": maxDepth,
	}

	if relationType != "" {
		query = fmt.Sprintf(`
		MATCH path = (start:Entity {id: $id})-[r:%s*1..%d]-(end:Entity)
		WHERE start <> end
		WITH end, length(path) as depth, relationships(path) as rels
		RETURN DISTINCT end, depth, rels[size(rels)-1] as lastRel
		ORDER BY depth, end.frequency DESC
		`, s.sanitizeRelationType(relationType), maxDepth)
	} else {
		query = fmt.Sprintf(`
		MATCH path = (start:Entity {id: $id})-[r*1..%d]-(end:Entity)
		WHERE start <> end
		WITH end, length(path) as depth, relationships(path) as rels
		RETURN DISTINCT end, depth, rels[size(rels)-1] as lastRel
		ORDER BY depth, end.frequency DESC
		`, maxDepth)
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var results []*GraphTraversalResult
	seen := make(map[string]bool)

	for result.Next(ctx) {
		record := result.Record()
		nodeVal, _ := record.Get("end")
		node := nodeVal.(neo4j.Node)
		entity := s.nodeToEntity(node)

		// 避免重复
		if seen[entity.ID] {
			continue
		}
		seen[entity.ID] = true

		depthVal, _ := record.Get("depth")
		depth := int(depthVal.(int64))

		relVal, _ := record.Get("lastRel")
		var relation *GraphRelation
		if relVal != nil {
			rel := relVal.(neo4j.Relationship)
			relation = &GraphRelation{
				ID:       s.getStringProp(rel.Props, "id"),
				Type:     rel.Type,
				Strength: s.getFloat32Prop(rel.Props, "strength"),
			}
		}

		score := float32(1.0) / float32(depth)
		if relation != nil {
			score *= relation.Strength
		}

		results = append(results, &GraphTraversalResult{
			Entity:   entity,
			Depth:    depth,
			Score:    score,
			Relation: relation,
		})
	}

	return results, nil
}

// GetShortestPath 最短路径查询
func (s *Neo4jGraphStore) GetShortestPath(ctx context.Context, fromID, toID string) ([]*GraphEntity, []*GraphRelation, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
	MATCH path = shortestPath((start:Entity {id: $fromId})-[*]-(end:Entity {id: $toId}))
	RETURN nodes(path) as nodes, relationships(path) as rels
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"fromId": fromID,
		"toId":   toID,
	})
	if err != nil {
		return nil, nil, err
	}

	if !result.Next(ctx) {
		return nil, nil, ErrNotFound
	}

	record := result.Record()

	// 提取节点
	nodesVal, _ := record.Get("nodes")
	nodes := nodesVal.([]interface{})
	entities := make([]*GraphEntity, len(nodes))
	for i, n := range nodes {
		entities[i] = s.nodeToEntity(n.(neo4j.Node))
	}

	// 提取关系
	relsVal, _ := record.Get("rels")
	rels := relsVal.([]interface{})
	relations := make([]*GraphRelation, len(rels))
	for i, r := range rels {
		rel := r.(neo4j.Relationship)
		relations[i] = &GraphRelation{
			ID:       s.getStringProp(rel.Props, "id"),
			Type:     rel.Type,
			Strength: s.getFloat32Prop(rel.Props, "strength"),
		}
	}

	return entities, relations, nil
}

// DeleteRelation 删除关系
func (s *Neo4jGraphStore) DeleteRelation(ctx context.Context, id string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `MATCH ()-[r {id: $id}]-() DELETE r`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return err
	}

	summary, err := result.Consume(ctx)
	if err != nil {
		return err
	}

	if summary.Counters().RelationshipsDeleted() == 0 {
		return ErrNotFound
	}

	return nil
}

// Clear 清空所有数据
func (s *Neo4jGraphStore) Clear(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `MATCH (n:Entity) DETACH DELETE n`

	_, err := session.Run(ctx, query, nil)
	return err
}

// GetStats 获取统计信息
func (s *Neo4jGraphStore) GetStats(ctx context.Context) (*GraphStoreStats, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// 统计实体数量和类型分布
	entityQuery := `MATCH (e:Entity) RETURN e.type as type, count(*) as count`

	result, err := session.Run(ctx, entityQuery, nil)
	if err != nil {
		return nil, err
	}

	entityTypes := make(map[string]int)
	totalEntities := 0
	for result.Next(ctx) {
		record := result.Record()
		typeVal, _ := record.Get("type")
		countVal, _ := record.Get("count")
		typeStr := typeVal.(string)
		count := int(countVal.(int64))
		entityTypes[typeStr] = count
		totalEntities += count
	}

	// 统计关系数量和类型分布
	relQuery := `MATCH ()-[r]->() RETURN type(r) as type, count(*) as count`

	result, err = session.Run(ctx, relQuery, nil)
	if err != nil {
		return nil, err
	}

	relationTypes := make(map[string]int)
	totalRelations := 0
	for result.Next(ctx) {
		record := result.Record()
		typeVal, _ := record.Get("type")
		countVal, _ := record.Get("count")
		typeStr := typeVal.(string)
		count := int(countVal.(int64))
		relationTypes[typeStr] = count
		totalRelations += count
	}

	return &GraphStoreStats{
		EntityCount:   totalEntities,
		RelationCount: totalRelations,
		EntityTypes:   entityTypes,
		RelationTypes: relationTypes,
	}, nil
}

// Close 关闭连接
func (s *Neo4jGraphStore) Close() error {
	return s.driver.Close(context.Background())
}

// nodeToEntity 将 Neo4j 节点转换为 Entity
func (s *Neo4jGraphStore) nodeToEntity(node neo4j.Node) *GraphEntity {
	return &GraphEntity{
		ID:          s.getStringProp(node.Props, "id"),
		Name:        s.getStringProp(node.Props, "name"),
		Type:        s.getStringProp(node.Props, "type"),
		Description: s.getStringProp(node.Props, "description"),
		Frequency:   s.getIntProp(node.Props, "frequency"),
		CreatedAt:   time.UnixMilli(s.getInt64Prop(node.Props, "created_at")),
		UpdatedAt:   time.UnixMilli(s.getInt64Prop(node.Props, "updated_at")),
	}
}

// getStringProp 获取字符串属性
func (s *Neo4jGraphStore) getStringProp(props map[string]interface{}, key string) string {
	if v, ok := props[key].(string); ok {
		return v
	}
	return ""
}

// getIntProp 获取整数属性
func (s *Neo4jGraphStore) getIntProp(props map[string]interface{}, key string) int {
	if v, ok := props[key].(int64); ok {
		return int(v)
	}
	return 0
}

// getInt64Prop 获取 int64 属性
func (s *Neo4jGraphStore) getInt64Prop(props map[string]interface{}, key string) int64 {
	if v, ok := props[key].(int64); ok {
		return v
	}
	return 0
}

// getFloat32Prop 获取 float32 属性
func (s *Neo4jGraphStore) getFloat32Prop(props map[string]interface{}, key string) float32 {
	if v, ok := props[key].(float64); ok {
		return float32(v)
	}
	return 0
}

// sanitizeRelationType 清理关系类型名称
func (s *Neo4jGraphStore) sanitizeRelationType(relType string) string {
	// Neo4j 关系类型只能包含字母、数字和下划线
	relType = strings.ToUpper(relType)
	relType = strings.ReplaceAll(relType, " ", "_")
	relType = strings.ReplaceAll(relType, "-", "_")
	return relType
}

// Compile-time interface check
var _ GraphStore = (*Neo4jGraphStore)(nil)
