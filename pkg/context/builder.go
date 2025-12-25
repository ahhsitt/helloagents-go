package context

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// Builder 定义构建上下文的接口。
type Builder interface {
	// Build 从给定输入构建上下文。
	Build(ctx context.Context, input *BuildInput) (string, error)

	// BuildMessages 从给定输入构建消息列表。
	// 这对于与 LLM 提供商的直接集成很有用。
	BuildMessages(ctx context.Context, input *BuildInput) ([]message.Message, error)
}

// BuildInput 包含上下文构建的所有输入数据。
type BuildInput struct {
	// Query 是当前用户查询。
	Query string

	// SystemInstructions 是系统提示/指令。
	SystemInstructions string

	// History 是对话历史。
	History []message.Message

	// AdditionalPackets 是要包含的额外上下文包。
	AdditionalPackets []*Packet
}

// GSSCBuilder 实现 GSSC（收集-筛选-结构化-压缩）流水线。
type GSSCBuilder struct {
	config     *Config
	gatherer   Gatherer
	selector   Selector
	structurer Structurer
	compressor Compressor
}

// BuilderOption 配置 GSSCBuilder。
type BuilderOption func(*GSSCBuilder)

// WithConfig 设置配置。
func WithConfig(config *Config) BuilderOption {
	return func(b *GSSCBuilder) {
		b.config = config
	}
}

// WithGatherer 设置收集器。
func WithGatherer(gatherer Gatherer) BuilderOption {
	return func(b *GSSCBuilder) {
		b.gatherer = gatherer
	}
}

// WithSelector 设置筛选器。
func WithSelector(selector Selector) BuilderOption {
	return func(b *GSSCBuilder) {
		b.selector = selector
	}
}

// WithStructurer 设置结构化器。
func WithStructurer(structurer Structurer) BuilderOption {
	return func(b *GSSCBuilder) {
		b.structurer = structurer
	}
}

// WithCompressor 设置压缩器。
func WithCompressor(compressor Compressor) BuilderOption {
	return func(b *GSSCBuilder) {
		b.compressor = compressor
	}
}

// NewGSSCBuilder 使用给定选项创建新的 GSSCBuilder。
func NewGSSCBuilder(opts ...BuilderOption) *GSSCBuilder {
	b := &GSSCBuilder{
		config: DefaultConfig(),
	}

	for _, opt := range opts {
		opt(b)
	}

	// 如果未配置则设置默认值
	if b.gatherer == nil {
		b.gatherer = NewCompositeGatherer([]Gatherer{
			NewInstructionsGatherer(),
			NewTaskGatherer(),
			NewHistoryGatherer(b.config.MaxHistoryMessages),
		}, false)
	}

	if b.selector == nil {
		b.selector = NewDefaultSelector(b.config)
	}

	if b.structurer == nil {
		b.structurer = NewDefaultStructurer()
	}

	if b.compressor == nil {
		b.compressor = NewTruncateCompressor()
	}

	return b
}

// Build 使用 GSSC 流水线构建上下文。
func (b *GSSCBuilder) Build(ctx context.Context, input *BuildInput) (string, error) {
	// 1. 收集：收集候选包
	gatherInput := &GatherInput{
		Query:              input.Query,
		SystemInstructions: input.SystemInstructions,
		History:            input.History,
		Config:             b.config,
	}

	packets, err := b.gatherer.Gather(ctx, gatherInput)
	if err != nil {
		return "", err
	}

	// 添加额外的包
	if len(input.AdditionalPackets) > 0 {
		packets = append(packets, input.AdditionalPackets...)
	}

	// 2. 筛选：对包进行评分和过滤
	selected := b.selector.Select(packets, input.Query, b.config)

	// 3. 结构化：组织成模板
	structured := b.structurer.Structure(selected, input.Query, b.config)

	// 4. 压缩：适应预算
	compressed := b.compressor.Compress(structured, b.config)

	return compressed, nil
}

// BuildMessages 从上下文构建消息列表。
func (b *GSSCBuilder) BuildMessages(ctx context.Context, input *BuildInput) ([]message.Message, error) {
	contextStr, err := b.Build(ctx, input)
	if err != nil {
		return nil, err
	}

	var messages []message.Message

	// 添加带有结构化上下文的系统消息
	if contextStr != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleSystem,
			Content: contextStr,
		})
	}

	// 添加用户查询
	if input.Query != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleUser,
			Content: input.Query,
		})
	}

	return messages, nil
}

// Config 返回构建器的配置。
func (b *GSSCBuilder) Config() *Config {
	return b.config
}

// SimpleBuilder 提供不使用完整 GSSC 流水线的简单构建器。
// 它只是连接系统提示 + 历史 + 查询。
type SimpleBuilder struct {
	config *Config
}

// NewSimpleBuilder 创建新的 SimpleBuilder。
func NewSimpleBuilder(config *Config) *SimpleBuilder {
	if config == nil {
		config = DefaultConfig()
	}
	return &SimpleBuilder{config: config}
}

// Build 构建简单的上下文字符串。
func (b *SimpleBuilder) Build(_ context.Context, input *BuildInput) (string, error) {
	// 估算容量：指令 + 历史 + 查询
	capacity := 1 + len(input.History) + 1
	parts := make([]string, 0, capacity)

	if input.SystemInstructions != "" {
		parts = append(parts, input.SystemInstructions)
	}

	// 添加有限的历史
	maxHistory := b.config.MaxHistoryMessages
	history := input.History
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	for _, msg := range history {
		parts = append(parts, string(msg.Role)+": "+msg.Content)
	}

	if input.Query != "" {
		parts = append(parts, "user: "+input.Query)
	}

	counter := b.config.GetTokenCounter()
	result := ""
	for _, part := range parts {
		result += part + "\n"
	}

	// 如果超出预算则简单截断
	for counter.Count(result) > b.config.GetAvailableTokens() && len(parts) > 2 {
		// 删除最旧的历史条目
		parts = append(parts[:1], parts[2:]...)
		result = ""
		for _, part := range parts {
			result += part + "\n"
		}
	}

	return result, nil
}

// BuildMessages 构建消息列表。
func (b *SimpleBuilder) BuildMessages(_ context.Context, input *BuildInput) ([]message.Message, error) {
	var messages []message.Message

	if input.SystemInstructions != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleSystem,
			Content: input.SystemInstructions,
		})
	}

	// 添加有限的历史
	maxHistory := b.config.MaxHistoryMessages
	history := input.History
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	messages = append(messages, history...)

	if input.Query != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleUser,
			Content: input.Query,
		})
	}

	return messages, nil
}

// 编译时接口检查
var _ Builder = (*GSSCBuilder)(nil)
var _ Builder = (*SimpleBuilder)(nil)
