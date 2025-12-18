package memory

import (
	"context"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// WorkingMemory 工作记忆实现
//
// 基于内存的对话历史存储，支持容量限制和 TTL。
type WorkingMemory struct {
	messages   []message.Message
	maxSize    int
	tokenLimit int
	ttl        time.Duration
	mu         sync.RWMutex
}

// WorkingMemoryOption 配置选项
type WorkingMemoryOption func(*WorkingMemory)

// NewWorkingMemory 创建工作记忆
func NewWorkingMemory(opts ...WorkingMemoryOption) *WorkingMemory {
	m := &WorkingMemory{
		messages:   make([]message.Message, 0),
		maxSize:    100,     // 默认最多 100 条消息
		tokenLimit: 4000,    // 默认 4000 token 限制
		ttl:        0,       // 默认不过期
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithMaxSize 设置最大消息数量
func WithMaxSize(size int) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.maxSize = size
	}
}

// WithTokenLimit 设置 token 限制
func WithTokenLimit(limit int) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.tokenLimit = limit
	}
}

// WithTTL 设置消息过期时间
func WithTTL(ttl time.Duration) WorkingMemoryOption {
	return func(m *WorkingMemory) {
		m.ttl = ttl
	}
}

// AddMessage 添加消息到记忆
func (m *WorkingMemory) AddMessage(ctx context.Context, msg message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 设置时间戳
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	m.messages = append(m.messages, msg)

	// 应用 LRU 清理
	if m.maxSize > 0 && len(m.messages) > m.maxSize {
		m.messages = m.messages[len(m.messages)-m.maxSize:]
	}

	return nil
}

// GetHistory 获取对话历史
func (m *WorkingMemory) GetHistory(ctx context.Context, limit int) ([]message.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 清理过期消息
	messages := m.filterExpired()

	if limit <= 0 || limit >= len(messages) {
		result := make([]message.Message, len(messages))
		copy(result, messages)
		return result, nil
	}

	// 返回最近的 limit 条
	start := len(messages) - limit
	result := make([]message.Message, limit)
	copy(result, messages[start:])
	return result, nil
}

// GetRecentHistory 获取最近 n 条消息
func (m *WorkingMemory) GetRecentHistory(ctx context.Context, n int) ([]message.Message, error) {
	return m.GetHistory(ctx, n)
}

// Clear 清空记忆
func (m *WorkingMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]message.Message, 0)
	return nil
}

// Size 返回当前消息数量
func (m *WorkingMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// filterExpired 过滤过期消息（内部使用，需要持有锁）
func (m *WorkingMemory) filterExpired() []message.Message {
	if m.ttl == 0 {
		return m.messages
	}

	cutoff := time.Now().Add(-m.ttl)
	result := make([]message.Message, 0, len(m.messages))

	for _, msg := range m.messages {
		if msg.Timestamp.After(cutoff) {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesWithinTokenLimit 获取不超过 token 限制的消息
//
// 从最新消息开始，向前累计直到达到 token 限制。
// 注意：此方法使用简化的 token 计算（按字符数估算）。
func (m *WorkingMemory) GetMessagesWithinTokenLimit(ctx context.Context) ([]message.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tokenLimit <= 0 {
		result := make([]message.Message, len(m.messages))
		copy(result, m.messages)
		return result, nil
	}

	messages := m.filterExpired()
	result := make([]message.Message, 0)
	totalTokens := 0

	// 从最新消息向前遍历
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		// 简化的 token 估算：1 token ≈ 4 字符（英文），中文约 1-2 字符
		tokens := len(msg.Content) / 3
		if totalTokens+tokens > m.tokenLimit {
			break
		}
		totalTokens += tokens
		result = append([]message.Message{msg}, result...)
	}

	return result, nil
}

// compile-time interface check
var _ ConversationMemory = (*WorkingMemory)(nil)
