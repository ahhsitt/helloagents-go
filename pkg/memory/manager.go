package memory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// 管理器相关错误
var (
	// ErrMemoryTypeNotFound 记忆类型未注册
	ErrMemoryTypeNotFound = errors.New("memory type not found")
	// ErrMemoryTypeExists 记忆类型已存在
	ErrMemoryTypeExists = errors.New("memory type already exists")
)

// Memory 统一记忆接口
//
// 所有记忆类型都需要实现此接口，便于 MemoryManager 统一管理。
type Memory interface {
	// Add 添加记忆项
	Add(ctx context.Context, item *MemoryItem) (string, error)

	// Retrieve 检索记忆
	Retrieve(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error)

	// Update 更新记忆
	Update(ctx context.Context, id string, opts ...UpdateOption) error

	// Remove 删除记忆
	Remove(ctx context.Context, id string) error

	// Has 检查记忆是否存在
	Has(ctx context.Context, id string) bool

	// Clear 清空记忆
	Clear(ctx context.Context) error

	// GetStats 获取统计信息
	GetStats(ctx context.Context) (*MemoryStats, error)
}

// MemoryStats 记忆统计信息
type MemoryStats struct {
	// Count 记忆数量
	Count int `json:"count"`
	// TotalTokens 总 token 数（如果适用）
	TotalTokens int `json:"total_tokens,omitempty"`
	// OldestTimestamp 最早记忆时间戳
	OldestTimestamp int64 `json:"oldest_timestamp,omitempty"`
	// NewestTimestamp 最新记忆时间戳
	NewestTimestamp int64 `json:"newest_timestamp,omitempty"`
	// AvgImportance 平均重要性
	AvgImportance float32 `json:"avg_importance,omitempty"`
}

// RetrieveOption 检索选项
type RetrieveOption func(*retrieveOptions)

type retrieveOptions struct {
	limit      int
	minScore   float32
	memoryType MemoryType
	userID     string
}

// WithLimit 设置返回数量限制
func WithLimit(limit int) RetrieveOption {
	return func(o *retrieveOptions) {
		o.limit = limit
	}
}

// WithMinScore 设置最小分数阈值
func WithMinScore(score float32) RetrieveOption {
	return func(o *retrieveOptions) {
		o.minScore = score
	}
}

// WithMemoryTypeFilter 按记忆类型过滤
func WithMemoryTypeFilter(t MemoryType) RetrieveOption {
	return func(o *retrieveOptions) {
		o.memoryType = t
	}
}

// WithUserIDFilter 按用户 ID 过滤
func WithUserIDFilter(userID string) RetrieveOption {
	return func(o *retrieveOptions) {
		o.userID = userID
	}
}

// UpdateOption 更新选项
type UpdateOption func(*updateOptions)

type updateOptions struct {
	content    *string
	importance *float32
	metadata   map[string]interface{}
}

// WithContentUpdate 更新内容
func WithContentUpdate(content string) UpdateOption {
	return func(o *updateOptions) {
		o.content = &content
	}
}

// WithImportanceUpdate 更新重要性
func WithImportanceUpdate(importance float32) UpdateOption {
	return func(o *updateOptions) {
		o.importance = &importance
	}
}

// WithMetadataUpdate 更新元数据
func WithMetadataUpdate(metadata map[string]interface{}) UpdateOption {
	return func(o *updateOptions) {
		o.metadata = metadata
	}
}

// ForgetStrategy 遗忘策略
type ForgetStrategy string

const (
	// ForgetByImportance 基于重要性遗忘
	ForgetByImportance ForgetStrategy = "importance_based"
	// ForgetByTime 基于时间遗忘
	ForgetByTime ForgetStrategy = "time_based"
	// ForgetByCapacity 基于容量遗忘
	ForgetByCapacity ForgetStrategy = "capacity_based"
)

// ForgetOption 遗忘选项
type ForgetOption func(*forgetOptions)

type forgetOptions struct {
	threshold      float32
	maxAgeDays     int
	targetCapacity int
}

// WithThreshold 设置重要性阈值（低于此值的记忆将被遗忘）
func WithThreshold(t float32) ForgetOption {
	return func(o *forgetOptions) {
		o.threshold = t
	}
}

// WithMaxAgeDays 设置最大保存天数
func WithMaxAgeDays(days int) ForgetOption {
	return func(o *forgetOptions) {
		o.maxAgeDays = days
	}
}

// WithTargetCapacity 设置目标容量
func WithTargetCapacity(n int) ForgetOption {
	return func(o *forgetOptions) {
		o.targetCapacity = n
	}
}

// MemoryManager 统一记忆管理器
//
// 管理多种类型的记忆，提供统一的添加、检索、遗忘接口。
type MemoryManager struct {
	config      *MemoryConfig
	userID      string
	memoryTypes map[MemoryType]Memory
	mu          sync.RWMutex
}

// ManagerOption 管理器配置选项
type ManagerOption func(*MemoryManager)

// WithManagerUserID 设置用户 ID
func WithManagerUserID(userID string) ManagerOption {
	return func(m *MemoryManager) {
		m.userID = userID
	}
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager(config *MemoryConfig, opts ...ManagerOption) *MemoryManager {
	if config == nil {
		config = DefaultMemoryConfig()
	}

	m := &MemoryManager{
		config:      config,
		memoryTypes: make(map[MemoryType]Memory),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// RegisterMemory 注册记忆类型
func (m *MemoryManager) RegisterMemory(memType MemoryType, memory Memory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.memoryTypes[memType]; exists {
		return ErrMemoryTypeExists
	}

	m.memoryTypes[memType] = memory
	return nil
}

// UnregisterMemory 取消注册记忆类型
func (m *MemoryManager) UnregisterMemory(memType MemoryType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.memoryTypes[memType]; !exists {
		return ErrMemoryTypeNotFound
	}

	delete(m.memoryTypes, memType)
	return nil
}

// GetMemory 获取指定类型的记忆存储
func (m *MemoryManager) GetMemory(memType MemoryType) (Memory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	memory, exists := m.memoryTypes[memType]
	if !exists {
		return nil, ErrMemoryTypeNotFound
	}

	return memory, nil
}

// AddMemoryOption 添加记忆选项
type AddMemoryOption func(*addMemoryOptions)

type addMemoryOptions struct {
	memoryType MemoryType
	importance float32
	metadata   map[string]interface{}
}

// WithAddMemoryType 指定记忆类型
func WithAddMemoryType(t MemoryType) AddMemoryOption {
	return func(o *addMemoryOptions) {
		o.memoryType = t
	}
}

// WithAddImportance 指定重要性
func WithAddImportance(importance float32) AddMemoryOption {
	return func(o *addMemoryOptions) {
		o.importance = importance
	}
}

// WithAddMetadata 指定元数据
func WithAddMetadata(metadata map[string]interface{}) AddMemoryOption {
	return func(o *addMemoryOptions) {
		o.metadata = metadata
	}
}

// AddMemory 添加记忆
//
// 如果未指定记忆类型，将自动分类。
func (m *MemoryManager) AddMemory(ctx context.Context, content string, opts ...AddMemoryOption) (string, error) {
	options := &addMemoryOptions{
		importance: 0.5,
	}
	for _, opt := range opts {
		opt(options)
	}

	// 如果未指定记忆类型，自动分类
	memType := options.memoryType
	if memType == "" {
		memType = m.classifyMemoryType(content)
	}

	// 如果未指定重要性，自动计算
	importance := options.importance
	if importance == 0.5 {
		importance = m.calculateImportance(content, options.metadata)
	}

	// 创建记忆项
	item := NewMemoryItem(content, memType,
		WithUserID(m.userID),
		WithImportance(importance),
		WithMetadata(options.metadata),
	)

	// 获取对应的记忆存储
	m.mu.RLock()
	memory, exists := m.memoryTypes[memType]
	m.mu.RUnlock()

	if !exists {
		return "", ErrMemoryTypeNotFound
	}

	return memory.Add(ctx, item)
}

// RetrieveMemories 从所有记忆类型检索
//
// 返回按相关性排序的结果。
func (m *MemoryManager) RetrieveMemories(ctx context.Context, query string, opts ...RetrieveOption) ([]*MemoryItem, error) {
	options := &retrieveOptions{
		limit: 10,
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	memories := make(map[MemoryType]Memory, len(m.memoryTypes))
	for k, v := range m.memoryTypes {
		memories[k] = v
	}
	m.mu.RUnlock()

	// 如果指定了记忆类型，只从该类型检索
	if options.memoryType != "" {
		memory, exists := memories[options.memoryType]
		if !exists {
			return nil, ErrMemoryTypeNotFound
		}
		return memory.Retrieve(ctx, query, opts...)
	}

	// 并行从所有记忆类型检索
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []*MemoryItem
		errs    []error
	)

	for _, memory := range memories {
		wg.Add(1)
		go func(mem Memory) {
			defer wg.Done()
			items, err := mem.Retrieve(ctx, query, opts...)
			mu.Lock()
			if err != nil {
				errs = append(errs, err)
			} else {
				results = append(results, items...)
			}
			mu.Unlock()
		}(memory)
	}

	wg.Wait()

	if len(errs) > 0 && len(results) == 0 {
		return nil, errs[0]
	}

	// 按重要性排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Importance > results[j].Importance
	})

	// 限制返回数量
	if options.limit > 0 && len(results) > options.limit {
		results = results[:options.limit]
	}

	return results, nil
}

// UpdateMemory 更新记忆
func (m *MemoryManager) UpdateMemory(ctx context.Context, memType MemoryType, id string, opts ...UpdateOption) error {
	m.mu.RLock()
	memory, exists := m.memoryTypes[memType]
	m.mu.RUnlock()

	if !exists {
		return ErrMemoryTypeNotFound
	}

	return memory.Update(ctx, id, opts...)
}

// RemoveMemory 删除记忆
func (m *MemoryManager) RemoveMemory(ctx context.Context, memType MemoryType, id string) error {
	m.mu.RLock()
	memory, exists := m.memoryTypes[memType]
	m.mu.RUnlock()

	if !exists {
		return ErrMemoryTypeNotFound
	}

	return memory.Remove(ctx, id)
}

// ForgetMemories 执行遗忘
//
// 根据策略从所有记忆类型中删除记忆。
func (m *MemoryManager) ForgetMemories(ctx context.Context, strategy ForgetStrategy, opts ...ForgetOption) (int, error) {
	m.mu.RLock()
	memories := make(map[MemoryType]Memory, len(m.memoryTypes))
	for k, v := range m.memoryTypes {
		memories[k] = v
	}
	m.mu.RUnlock()

	totalForgotten := 0

	for _, memory := range memories {
		// 检查是否实现了 Forgetter 接口
		if forgetter, ok := memory.(Forgetter); ok {
			count, err := forgetter.Forget(ctx, strategy, opts...)
			if err != nil {
				continue // 忽略单个记忆类型的错误
			}
			totalForgotten += count
		}
	}

	return totalForgotten, nil
}

// Forgetter 支持遗忘的记忆接口
type Forgetter interface {
	Forget(ctx context.Context, strategy ForgetStrategy, opts ...ForgetOption) (int, error)
}

// ConsolidateOption 整合选项
type ConsolidateOption func(*consolidateOptions)

type consolidateOptions struct {
	minImportance float32
	maxAge        int
	targetType    MemoryType
}

// WithConsolidateMinImportance 设置最小重要性阈值
func WithConsolidateMinImportance(importance float32) ConsolidateOption {
	return func(o *consolidateOptions) {
		o.minImportance = importance
	}
}

// WithConsolidateMaxAge 设置最大保存天数（仅整合超过此天数的记忆）
func WithConsolidateMaxAge(days int) ConsolidateOption {
	return func(o *consolidateOptions) {
		o.maxAge = days
	}
}

// WithConsolidateTargetType 设置目标记忆类型
func WithConsolidateTargetType(t MemoryType) ConsolidateOption {
	return func(o *consolidateOptions) {
		o.targetType = t
	}
}

// ConsolidateMemories 整合记忆
//
// 将高重要性的工作记忆转移到情景/语义记忆。
func (m *MemoryManager) ConsolidateMemories(ctx context.Context, opts ...ConsolidateOption) (int, error) {
	options := &consolidateOptions{
		minImportance: 0.7,   // 默认只整合重要性 >= 0.7 的记忆
		targetType:    MemoryTypeEpisodic, // 默认整合到情景记忆
	}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.RLock()
	workingMem, hasWorking := m.memoryTypes[MemoryTypeWorking]
	targetMem, hasTarget := m.memoryTypes[options.targetType]
	m.mu.RUnlock()

	if !hasWorking {
		return 0, nil // 没有工作记忆，无需整合
	}
	if !hasTarget {
		return 0, ErrMemoryTypeNotFound
	}

	// 从工作记忆中检索高重要性的记忆
	items, err := workingMem.Retrieve(ctx, "", WithLimit(1000))
	if err != nil {
		return 0, err
	}

	consolidated := 0
	for _, item := range items {
		// 检查重要性阈值
		if item.Importance < options.minImportance {
			continue
		}

		// 检查年龄（如果设置了）
		if options.maxAge > 0 && item.AgeDays() < float64(options.maxAge) {
			continue
		}

		// 创建新的记忆项（转换类型）
		newItem := item.Clone()
		newItem.MemoryType = options.targetType

		// 添加到目标记忆
		_, err := targetMem.Add(ctx, newItem)
		if err != nil {
			continue
		}

		// 从工作记忆中删除
		if err := workingMem.Remove(ctx, item.ID); err != nil {
			continue
		}

		consolidated++
	}

	return consolidated, nil
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	// TotalCount 总记忆数
	TotalCount int `json:"total_count"`
	// ByType 按类型统计
	ByType map[MemoryType]*MemoryStats `json:"by_type"`
}

// GetStats 获取统计信息
func (m *MemoryManager) GetStats(ctx context.Context) (*ManagerStats, error) {
	m.mu.RLock()
	memories := make(map[MemoryType]Memory, len(m.memoryTypes))
	for k, v := range m.memoryTypes {
		memories[k] = v
	}
	m.mu.RUnlock()

	stats := &ManagerStats{
		ByType: make(map[MemoryType]*MemoryStats),
	}

	for memType, memory := range memories {
		memStats, err := memory.GetStats(ctx)
		if err != nil {
			continue
		}
		stats.ByType[memType] = memStats
		stats.TotalCount += memStats.Count
	}

	return stats, nil
}

// classifyMemoryType 自动分类记忆类型
//
// 基于内容关键词判断记忆类型。
func (m *MemoryManager) classifyMemoryType(content string) MemoryType {
	contentLower := strings.ToLower(content)

	// 情景记忆关键词：事件、经历、时间相关
	episodicKeywords := []string{
		"happened", "occurred", "yesterday", "today", "last time",
		"remember when", "event", "meeting", "conversation",
		"发生", "昨天", "今天", "上次", "事件", "会议", "对话",
		"记得", "当时", "那时候",
	}
	for _, kw := range episodicKeywords {
		if strings.Contains(contentLower, kw) {
			return MemoryTypeEpisodic
		}
	}

	// 语义记忆关键词：概念、知识、定义
	semanticKeywords := []string{
		"definition", "concept", "theory", "principle", "fact",
		"means", "is a", "refers to", "known as",
		"定义", "概念", "理论", "原理", "事实",
		"是指", "是一种", "称为", "叫做",
	}
	for _, kw := range semanticKeywords {
		if strings.Contains(contentLower, kw) {
			return MemoryTypeSemantic
		}
	}

	// 默认返回工作记忆
	return MemoryTypeWorking
}

// calculateImportance 计算重要性
//
// 基于内容长度、关键词、元数据计算。
func (m *MemoryManager) calculateImportance(content string, metadata map[string]interface{}) float32 {
	var importance float32 = 0.5

	// 基于内容长度
	length := len(content)
	if length < 20 {
		importance -= 0.1
	} else if length > 500 {
		importance += 0.2
	} else if length > 200 {
		importance += 0.1
	}

	// 基于关键词
	contentLower := strings.ToLower(content)
	highImportanceKeywords := []string{
		"important", "critical", "urgent", "remember", "don't forget",
		"key", "essential", "must", "should",
		"重要", "关键", "紧急", "记住", "别忘了",
		"必须", "应该", "一定",
	}
	for _, kw := range highImportanceKeywords {
		if strings.Contains(contentLower, kw) {
			importance += 0.15
			break
		}
	}

	lowImportanceKeywords := []string{
		"maybe", "perhaps", "might", "possibly", "trivial",
		"也许", "可能", "或许", "无关紧要",
	}
	for _, kw := range lowImportanceKeywords {
		if strings.Contains(contentLower, kw) {
			importance -= 0.1
			break
		}
	}

	// 基于元数据
	if metadata != nil {
		// 如果明确指定了重要性
		if imp, ok := metadata["importance"].(float32); ok {
			return imp
		}
		if imp, ok := metadata["importance"].(float64); ok {
			return float32(imp)
		}

		// 如果有特殊标记
		if priority, ok := metadata["priority"].(string); ok {
			switch strings.ToLower(priority) {
			case "high":
				importance += 0.2
			case "low":
				importance -= 0.2
			}
		}
	}

	// 确保在有效范围内
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}

	return importance
}

// Config 返回配置
func (m *MemoryManager) Config() *MemoryConfig {
	return m.config
}

// UserID 返回用户 ID
func (m *MemoryManager) UserID() string {
	return m.userID
}
