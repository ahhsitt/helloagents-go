package context

import (
	"time"
)

// PacketType 表示上下文包的类型/优先级。
type PacketType string

const (
	// PacketTypeInstructions 表示系统指令（P0 - 最高优先级）。
	PacketTypeInstructions PacketType = "instructions"

	// PacketTypeTaskState 表示当前任务状态和结论（P1）。
	PacketTypeTaskState PacketType = "task_state"

	// PacketTypeTask 表示当前用户任务/查询（P1）。
	PacketTypeTask PacketType = "task"

	// PacketTypeEvidence 表示来自 Memory/RAG 的事实证据（P2）。
	PacketTypeEvidence PacketType = "evidence"

	// PacketTypeHistory 表示对话历史（P3 - 最低优先级）。
	PacketTypeHistory PacketType = "history"

	// PacketTypeCustom 表示自定义/用户定义的上下文。
	PacketTypeCustom PacketType = "custom"
)

// Priority 返回包类型的优先级（0 = 最高）。
func (t PacketType) Priority() int {
	switch t {
	case PacketTypeInstructions:
		return 0
	case PacketTypeTask, PacketTypeTaskState:
		return 1
	case PacketTypeEvidence:
		return 2
	case PacketTypeHistory:
		return 3
	default:
		return 4
	}
}

// Packet 表示带有元数据的上下文信息单元。
type Packet struct {
	// Content 是包的实际文本内容。
	Content string

	// Type 表示此包的类别/优先级。
	Type PacketType

	// Timestamp 是此包创建或内容生成的时间。
	Timestamp time.Time

	// TokenCount 是内容的 Token 数量。
	TokenCount int

	// RelevanceScore 表示此包与当前查询的相关程度（0.0-1.0）。
	RelevanceScore float64

	// RecencyScore 表示此包的新近程度（0.0-1.0）。
	RecencyScore float64

	// CompositeScore 是用于选择的综合评分。
	CompositeScore float64

	// Metadata 包含关于此包的额外键值数据。
	Metadata map[string]interface{}

	// Source 表示此包的来源（例如 "memory"、"rag"、"history"）。
	Source string
}

// PacketOption 配置 Packet。
type PacketOption func(*Packet)

// WithPacketType 设置包类型。
func WithPacketType(t PacketType) PacketOption {
	return func(p *Packet) {
		p.Type = t
	}
}

// WithTimestamp 设置包的时间戳。
func WithTimestamp(ts time.Time) PacketOption {
	return func(p *Packet) {
		p.Timestamp = ts
	}
}

// WithRelevanceScore 设置相关性评分。
func WithRelevanceScore(score float64) PacketOption {
	return func(p *Packet) {
		p.RelevanceScore = score
	}
}

// WithMetadata 设置包的元数据。
func WithMetadata(metadata map[string]interface{}) PacketOption {
	return func(p *Packet) {
		p.Metadata = metadata
	}
}

// WithSource 设置包的来源。
func WithSource(source string) PacketOption {
	return func(p *Packet) {
		p.Source = source
	}
}

// WithTokenCount 设置 Token 数量（跳过自动计算）。
func WithTokenCount(count int) PacketOption {
	return func(p *Packet) {
		p.TokenCount = count
	}
}

// NewPacket 使用给定的内容和选项创建新的 Packet。
// 如果未提供，Token 数量会自动计算。
func NewPacket(content string, opts ...PacketOption) *Packet {
	p := &Packet{
		Content:   content,
		Type:      PacketTypeCustom,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	// 如果未设置则自动计算 Token 数量
	if p.TokenCount == 0 {
		counter := DefaultTokenCounter()
		p.TokenCount = counter.Count(content)
	}

	return p
}

// NewInstructionsPacket 创建系统指令包。
func NewInstructionsPacket(content string) *Packet {
	return NewPacket(content,
		WithPacketType(PacketTypeInstructions),
		WithSource("system"),
		WithRelevanceScore(1.0), // 始终相关
	)
}

// NewTaskPacket 创建当前任务/查询包。
func NewTaskPacket(query string) *Packet {
	return NewPacket(query,
		WithPacketType(PacketTypeTask),
		WithSource("user"),
		WithRelevanceScore(1.0), // 始终相关
	)
}

// NewHistoryPacket 创建对话历史包。
func NewHistoryPacket(content string, timestamp time.Time) *Packet {
	return NewPacket(content,
		WithPacketType(PacketTypeHistory),
		WithTimestamp(timestamp),
		WithSource("history"),
	)
}

// NewEvidencePacket 创建来自 Memory 或 RAG 的证据包。
func NewEvidencePacket(content string, source string, relevance float64) *Packet {
	return NewPacket(content,
		WithPacketType(PacketTypeEvidence),
		WithSource(source),
		WithRelevanceScore(relevance),
	)
}

// NewTaskStatePacket 创建任务状态信息包。
func NewTaskStatePacket(content string) *Packet {
	return NewPacket(content,
		WithPacketType(PacketTypeTaskState),
		WithSource("memory"),
	)
}

// Clone 创建包的深拷贝。
func (p *Packet) Clone() *Packet {
	clone := &Packet{
		Content:        p.Content,
		Type:           p.Type,
		Timestamp:      p.Timestamp,
		TokenCount:     p.TokenCount,
		RelevanceScore: p.RelevanceScore,
		RecencyScore:   p.RecencyScore,
		CompositeScore: p.CompositeScore,
		Source:         p.Source,
		Metadata:       make(map[string]interface{}, len(p.Metadata)),
	}

	for k, v := range p.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// SetMetadata 设置元数据值。
func (p *Packet) SetMetadata(key string, value interface{}) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}
	p.Metadata[key] = value
}

// GetMetadata 获取元数据值。
func (p *Packet) GetMetadata(key string) (interface{}, bool) {
	if p.Metadata == nil {
		return nil, false
	}
	v, ok := p.Metadata[key]
	return v, ok
}
