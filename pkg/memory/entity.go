package memory

import (
	"time"

	"github.com/google/uuid"
)

// EntityType 实体类型
type EntityType string

const (
	// EntityTypePerson 人物
	EntityTypePerson EntityType = "person"
	// EntityTypeOrganization 组织
	EntityTypeOrganization EntityType = "organization"
	// EntityTypeLocation 地点
	EntityTypeLocation EntityType = "location"
	// EntityTypeConcept 概念
	EntityTypeConcept EntityType = "concept"
	// EntityTypeEvent 事件
	EntityTypeEvent EntityType = "event"
	// EntityTypeProduct 产品
	EntityTypeProduct EntityType = "product"
	// EntityTypeOther 其他
	EntityTypeOther EntityType = "other"
)

// Entity 实体结构
//
// 表示知识图谱中的节点，可以是人物、组织、地点、概念等。
type Entity struct {
	// ID 唯一标识
	ID string `json:"id"`
	// Name 实体名称
	Name string `json:"name"`
	// Type 实体类型
	Type EntityType `json:"type"`
	// Description 实体描述
	Description string `json:"description,omitempty"`
	// Properties 附加属性
	Properties map[string]interface{} `json:"properties,omitempty"`
	// Frequency 出现频率
	Frequency int `json:"frequency"`
	// Vector 嵌入向量（用于语义检索）
	Vector []float32 `json:"-"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// NewEntity 创建新实体
func NewEntity(name string, entityType EntityType) *Entity {
	now := time.Now()
	return &Entity{
		ID:         uuid.New().String(),
		Name:       name,
		Type:       entityType,
		Properties: make(map[string]interface{}),
		Frequency:  1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// SetProperty 设置属性
func (e *Entity) SetProperty(key string, value interface{}) {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	e.Properties[key] = value
	e.UpdatedAt = time.Now()
}

// GetProperty 获取属性
func (e *Entity) GetProperty(key string) (interface{}, bool) {
	if e.Properties == nil {
		return nil, false
	}
	v, ok := e.Properties[key]
	return v, ok
}

// IncrementFrequency 增加出现频率
func (e *Entity) IncrementFrequency() {
	e.Frequency++
	e.UpdatedAt = time.Now()
}

// RelationType 关系类型
type RelationType string

const (
	// RelationTypeRelatedTo 相关
	RelationTypeRelatedTo RelationType = "related_to"
	// RelationTypePartOf 属于
	RelationTypePartOf RelationType = "part_of"
	// RelationTypeHasA 拥有
	RelationTypeHasA RelationType = "has_a"
	// RelationTypeIsA 是一种
	RelationTypeIsA RelationType = "is_a"
	// RelationTypeLocatedIn 位于
	RelationTypeLocatedIn RelationType = "located_in"
	// RelationTypeWorksAt 工作于
	RelationTypeWorksAt RelationType = "works_at"
	// RelationTypeKnows 认识
	RelationTypeKnows RelationType = "knows"
	// RelationTypeCreatedBy 创建者
	RelationTypeCreatedBy RelationType = "created_by"
	// RelationTypeDependsOn 依赖
	RelationTypeDependsOn RelationType = "depends_on"
	// RelationTypeSimilarTo 相似
	RelationTypeSimilarTo RelationType = "similar_to"
)

// Relation 关系结构
//
// 表示知识图谱中两个实体之间的关系。
type Relation struct {
	// ID 唯一标识
	ID string `json:"id"`
	// FromEntityID 源实体 ID
	FromEntityID string `json:"from_entity_id"`
	// ToEntityID 目标实体 ID
	ToEntityID string `json:"to_entity_id"`
	// RelationType 关系类型
	RelationType RelationType `json:"relation_type"`
	// Strength 关系强度 (0-1)
	Strength float32 `json:"strength"`
	// Evidence 证据列表（相关记忆 ID）
	Evidence []string `json:"evidence,omitempty"`
	// Properties 附加属性
	Properties map[string]interface{} `json:"properties,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// NewRelation 创建新关系
func NewRelation(fromID, toID string, relType RelationType) *Relation {
	now := time.Now()
	return &Relation{
		ID:           uuid.New().String(),
		FromEntityID: fromID,
		ToEntityID:   toID,
		RelationType: relType,
		Strength:     1.0,
		Evidence:     make([]string, 0),
		Properties:   make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddEvidence 添加证据
func (r *Relation) AddEvidence(memoryID string) {
	for _, e := range r.Evidence {
		if e == memoryID {
			return // 已存在
		}
	}
	r.Evidence = append(r.Evidence, memoryID)
	r.UpdatedAt = time.Now()
}

// SetProperty 设置属性
func (r *Relation) SetProperty(key string, value interface{}) {
	if r.Properties == nil {
		r.Properties = make(map[string]interface{})
	}
	r.Properties[key] = value
	r.UpdatedAt = time.Now()
}

// UpdateStrength 更新关系强度
func (r *Relation) UpdateStrength(strength float32) {
	if strength < 0 {
		strength = 0
	}
	if strength > 1 {
		strength = 1
	}
	r.Strength = strength
	r.UpdatedAt = time.Now()
}

// EntityOption 实体选项
type EntityOption func(*entityOptions)

type entityOptions struct {
	description string
	properties  map[string]interface{}
	frequency   int
}

// WithEntityDescription 设置实体描述
func WithEntityDescription(desc string) EntityOption {
	return func(o *entityOptions) {
		o.description = desc
	}
}

// WithEntityProperties 设置实体属性
func WithEntityProperties(props map[string]interface{}) EntityOption {
	return func(o *entityOptions) {
		o.properties = props
	}
}

// WithEntityFrequency 设置初始频率
func WithEntityFrequency(freq int) EntityOption {
	return func(o *entityOptions) {
		o.frequency = freq
	}
}

// NewEntityWithOptions 使用选项创建实体
func NewEntityWithOptions(name string, entityType EntityType, opts ...EntityOption) *Entity {
	options := &entityOptions{
		frequency: 1,
	}
	for _, opt := range opts {
		opt(options)
	}

	entity := NewEntity(name, entityType)
	entity.Description = options.description
	if options.properties != nil {
		entity.Properties = options.properties
	}
	entity.Frequency = options.frequency
	return entity
}

// RelationOption 关系选项
type RelationOption func(*relationOptions)

type relationOptions struct {
	strength   float32
	evidence   []string
	properties map[string]interface{}
}

// WithRelationStrength 设置关系强度
func WithRelationStrength(strength float32) RelationOption {
	return func(o *relationOptions) {
		o.strength = strength
	}
}

// WithRelationEvidence 设置证据
func WithRelationEvidence(evidence []string) RelationOption {
	return func(o *relationOptions) {
		o.evidence = evidence
	}
}

// WithRelationProperties 设置关系属性
func WithRelationProperties(props map[string]interface{}) RelationOption {
	return func(o *relationOptions) {
		o.properties = props
	}
}

// NewRelationWithOptions 使用选项创建关系
func NewRelationWithOptions(fromID, toID string, relType RelationType, opts ...RelationOption) *Relation {
	options := &relationOptions{
		strength: 1.0,
	}
	for _, opt := range opts {
		opt(options)
	}

	rel := NewRelation(fromID, toID, relType)
	rel.Strength = options.strength
	if options.evidence != nil {
		rel.Evidence = options.evidence
	}
	if options.properties != nil {
		rel.Properties = options.properties
	}
	return rel
}

// GraphSearchResult 图搜索结果
type GraphSearchResult struct {
	// Entity 实体
	Entity *Entity `json:"entity"`
	// Depth 搜索深度
	Depth int `json:"depth"`
	// Path 路径（从起点到该实体的关系列表）
	Path []*Relation `json:"path,omitempty"`
	// Score 得分
	Score float32 `json:"score"`
}

// ExtractedEntity 提取的实体
type ExtractedEntity struct {
	// Name 实体名称
	Name string `json:"name"`
	// Type 实体类型
	Type EntityType `json:"type"`
	// StartPos 开始位置
	StartPos int `json:"start_pos"`
	// EndPos 结束位置
	EndPos int `json:"end_pos"`
	// Confidence 置信度
	Confidence float32 `json:"confidence"`
}

// ExtractedRelation 提取的关系
type ExtractedRelation struct {
	// FromEntity 源实体名称
	FromEntity string `json:"from_entity"`
	// ToEntity 目标实体名称
	ToEntity string `json:"to_entity"`
	// RelationType 关系类型
	RelationType RelationType `json:"relation_type"`
	// Confidence 置信度
	Confidence float32 `json:"confidence"`
	// Context 上下文（包含关系的句子）
	Context string `json:"context"`
}
