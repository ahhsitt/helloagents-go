package memory

import (
	"time"

	"github.com/google/uuid"
)

// MemoryType 记忆类型常量
type MemoryType string

const (
	// MemoryTypeWorking 工作记忆
	MemoryTypeWorking MemoryType = "working"
	// MemoryTypeEpisodic 情景记忆
	MemoryTypeEpisodic MemoryType = "episodic"
	// MemoryTypeSemantic 语义记忆
	MemoryTypeSemantic MemoryType = "semantic"
)

// MemoryItem 统一记忆项数据结构
//
// 作为所有记忆类型的通用结构，便于统一管理和跨类型操作。
type MemoryItem struct {
	// ID 唯一标识
	ID string `json:"id"`
	// Content 记忆内容
	Content string `json:"content"`
	// MemoryType 记忆类型
	MemoryType MemoryType `json:"memory_type"`
	// UserID 用户标识
	UserID string `json:"user_id"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Importance 重要性评分 (0-1)
	Importance float32 `json:"importance"`
	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryItemOption 配置选项
type MemoryItemOption func(*MemoryItem)

// NewMemoryItem 创建新的记忆项
func NewMemoryItem(content string, memoryType MemoryType, opts ...MemoryItemOption) *MemoryItem {
	item := &MemoryItem{
		ID:         uuid.New().String(),
		Content:    content,
		MemoryType: memoryType,
		Timestamp:  time.Now(),
		Importance: 0.5, // 默认中等重要性
		Metadata:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(item)
	}

	return item
}

// WithID 设置 ID
func WithID(id string) MemoryItemOption {
	return func(item *MemoryItem) {
		item.ID = id
	}
}

// WithUserID 设置用户 ID
func WithUserID(userID string) MemoryItemOption {
	return func(item *MemoryItem) {
		item.UserID = userID
	}
}

// WithTimestamp 设置时间戳
func WithTimestamp(ts time.Time) MemoryItemOption {
	return func(item *MemoryItem) {
		item.Timestamp = ts
	}
}

// WithImportance 设置重要性
func WithImportance(importance float32) MemoryItemOption {
	return func(item *MemoryItem) {
		if importance < 0 {
			importance = 0
		}
		if importance > 1 {
			importance = 1
		}
		item.Importance = importance
	}
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]interface{}) MemoryItemOption {
	return func(item *MemoryItem) {
		item.Metadata = metadata
	}
}

// WithMetadataKV 添加单个元数据键值对
func WithMetadataKV(key string, value interface{}) MemoryItemOption {
	return func(item *MemoryItem) {
		if item.Metadata == nil {
			item.Metadata = make(map[string]interface{})
		}
		item.Metadata[key] = value
	}
}

// Validate 验证记忆项的有效性
func (item *MemoryItem) Validate() error {
	if item.Content == "" {
		return ErrInvalidInput
	}
	if item.MemoryType == "" {
		return ErrInvalidInput
	}
	if item.Importance < 0 || item.Importance > 1 {
		return ErrInvalidInput
	}
	return nil
}

// Clone 克隆记忆项
func (item *MemoryItem) Clone() *MemoryItem {
	cloned := &MemoryItem{
		ID:         item.ID,
		Content:    item.Content,
		MemoryType: item.MemoryType,
		UserID:     item.UserID,
		Timestamp:  item.Timestamp,
		Importance: item.Importance,
	}

	if item.Metadata != nil {
		cloned.Metadata = make(map[string]interface{}, len(item.Metadata))
		for k, v := range item.Metadata {
			cloned.Metadata[k] = v
		}
	}

	return cloned
}

// GetMetadataString 获取字符串类型的元数据
func (item *MemoryItem) GetMetadataString(key string) string {
	if item.Metadata == nil {
		return ""
	}
	if v, ok := item.Metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetMetadataFloat 获取浮点数类型的元数据
func (item *MemoryItem) GetMetadataFloat(key string) float64 {
	if item.Metadata == nil {
		return 0
	}
	if v, ok := item.Metadata[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

// AgeHours 返回记忆的年龄（小时）
func (item *MemoryItem) AgeHours() float64 {
	return time.Since(item.Timestamp).Hours()
}

// AgeDays 返回记忆的年龄（天）
func (item *MemoryItem) AgeDays() float64 {
	return time.Since(item.Timestamp).Hours() / 24.0
}
