// Package context 为 HelloAgents 框架提供上下文工程能力。
//
// 本包实现了 GSSC (Gather-Select-Structure-Compress) 流水线，
// 用于构建优化的 LLM 交互上下文。
package context

import (
	"strings"

	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter 定义 Token 计数接口。
type TokenCounter interface {
	// Count 返回给定文本的 Token 数量。
	Count(text string) int

	// CountMessages 返回消息列表的总 Token 数量，
	// 包括角色前缀和分隔符。
	CountMessages(messages []message.Message) int
}

// TiktokenCounter 使用 tiktoken 实现精确的 Token 计数。
type TiktokenCounter struct {
	encoding *tiktoken.Tiktoken
	model    string
}

// TiktokenOption 配置 TiktokenCounter。
type TiktokenOption func(*TiktokenCounter)

// WithModel 设置 Token 编码使用的模型。
// 支持的模型：gpt-4、gpt-4o、gpt-3.5-turbo 等。
func WithModel(model string) TiktokenOption {
	return func(c *TiktokenCounter) {
		c.model = model
	}
}

// NewTiktokenCounter 创建新的 TiktokenCounter。
// 默认使用 cl100k_base 编码（GPT-4、GPT-4o 等使用）。
func NewTiktokenCounter(opts ...TiktokenOption) (*TiktokenCounter, error) {
	c := &TiktokenCounter{
		model: "gpt-4o", // 默认使用 GPT-4o
	}

	for _, opt := range opts {
		opt(c)
	}

	// 尝试获取模型对应的编码
	encoding, err := tiktoken.EncodingForModel(c.model)
	if err != nil {
		// 降级到 cl100k_base 编码
		encoding, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, err
		}
	}

	c.encoding = encoding
	return c, nil
}

// Count 返回给定文本的 Token 数量。
func (c *TiktokenCounter) Count(text string) int {
	if c.encoding == nil {
		return estimateTokens(text)
	}
	return len(c.encoding.Encode(text, nil, nil))
}

// CountMessages 返回消息列表的总 Token 数量。
// 这会考虑 OpenAI API 中消息格式化的开销。
func (c *TiktokenCounter) CountMessages(messages []message.Message) int {
	// 基于 OpenAI 的 Token 计数指南：
	// https://cookbook.openai.com/examples/how_to_count_tokens_with_tiktoken
	tokensPerMessage := 3 // <|start|>{role/name}\n{content}<|end|>\n
	tokensPerName := 1

	total := 0
	for _, msg := range messages {
		total += tokensPerMessage
		total += c.Count(string(msg.Role))
		total += c.Count(msg.Content)
		if msg.Name != "" {
			total += c.Count(msg.Name) + tokensPerName
		}
	}
	total += 3 // 每个回复都以 <|start|>assistant<|message|> 开头

	return total
}

// EstimatedCounter 使用字符估算实现 Token 计数。
// 这是当 tiktoken 不可用时的降级方案。
type EstimatedCounter struct {
	// CharsPerToken 是每个 Token 的平均字符数。
	// 默认值为 4，这是英文文本的合理估计。
	CharsPerToken float64
}

// NewEstimatedCounter 创建新的 EstimatedCounter。
func NewEstimatedCounter() *EstimatedCounter {
	return &EstimatedCounter{
		CharsPerToken: 4.0,
	}
}

// Count 返回估算的 Token 数量。
func (c *EstimatedCounter) Count(text string) int {
	if c.CharsPerToken <= 0 {
		c.CharsPerToken = 4.0
	}
	return int(float64(len(text)) / c.CharsPerToken)
}

// CountMessages 返回消息列表的估算 Token 数量。
func (c *EstimatedCounter) CountMessages(messages []message.Message) int {
	tokensPerMessage := 4 // 每条消息的开销

	total := 0
	for _, msg := range messages {
		total += tokensPerMessage
		total += c.Count(string(msg.Role))
		total += c.Count(msg.Content)
		if msg.Name != "" {
			total += c.Count(msg.Name) + 1
		}
	}
	total += 3 // 回复引导

	return total
}

// estimateTokens 提供简单的 Token 估算降级方案。
func estimateTokens(text string) int {
	// 粗略估算：英文 1 token ≈ 4 字符，
	// 但中文/日文字符通常每个 1-2 个 token
	charCount := len(text)
	wordCount := len(strings.Fields(text))

	// 使用混合方法：同时计算词数和字符数
	// 这对混合内容效果更好
	if wordCount == 0 {
		return charCount / 4
	}

	// 取字符估算和词估算的平均值
	charBasedTokens := charCount / 4
	wordBasedTokens := int(float64(wordCount) * 1.3) // 平均每词约 1.3 个 token

	return (charBasedTokens + wordBasedTokens) / 2
}

// DefaultTokenCounter 返回一个 TokenCounter，
// 优先使用 TiktokenCounter，如果不可用则降级到 EstimatedCounter。
func DefaultTokenCounter() TokenCounter {
	counter, err := NewTiktokenCounter()
	if err != nil {
		return NewEstimatedCounter()
	}
	return counter
}

// 编译时接口检查
var _ TokenCounter = (*TiktokenCounter)(nil)
var _ TokenCounter = (*EstimatedCounter)(nil)
