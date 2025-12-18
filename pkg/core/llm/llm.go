// Package llm 提供 LLM 服务的统一接口
package llm

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// Provider 定义 LLM 提供商接口
//
// 统一不同 LLM 服务的调用方式，支持 OpenAI、DeepSeek、通义千问、Ollama、vLLM 等。
type Provider interface {
	// Generate 生成响应（非流式）
	//
	// 参数:
	//   - ctx: 上下文
	//   - req: 请求参数
	//
	// 返回:
	//   - Response: 响应结果
	//   - error: 调用错误
	Generate(ctx context.Context, req Request) (Response, error)

	// GenerateStream 生成响应（流式）
	//
	// 返回两个 channel：
	//   - <-chan StreamChunk: 流式响应块
	//   - <-chan error: 错误通道（最多一个错误）
	GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error)

	// Embed 生成文本嵌入向量
	//
	// 参数:
	//   - ctx: 上下文
	//   - texts: 待嵌入的文本列表
	//
	// 返回:
	//   - [][]float32: 嵌入向量列表
	//   - error: 调用错误
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Name 返回提供商名称
	Name() string

	// Model 返回当前模型名称
	Model() string

	// Close 关闭客户端连接
	Close() error
}

// ToolDefinition 工具定义（用于 Function Calling）
type ToolDefinition struct {
	// Name 工具名称
	Name string `json:"name"`
	// Description 工具描述
	Description string `json:"description"`
	// Parameters 参数 Schema (JSON Schema 格式)
	Parameters map[string]interface{} `json:"parameters"`
}

// Request LLM 请求
type Request struct {
	// Messages 消息历史
	Messages []message.Message
	// Tools 可用工具列表（可选）
	Tools []ToolDefinition
	// ToolChoice 工具选择策略（可选）
	// 值: "auto", "none", 或具体工具名
	ToolChoice interface{}
	// Temperature 温度参数（可选）
	Temperature *float64
	// MaxTokens 最大输出 token（可选）
	MaxTokens *int
	// TopP 核采样参数（可选）
	TopP *float64
	// Stop 停止序列（可选）
	Stop []string
}

// Response LLM 响应
type Response struct {
	// ID 响应标识
	ID string `json:"id"`
	// Content 响应文本内容
	Content string `json:"content"`
	// ToolCalls 工具调用请求（如有）
	ToolCalls []message.ToolCall `json:"tool_calls,omitempty"`
	// TokenUsage Token 使用统计
	TokenUsage message.TokenUsage `json:"token_usage"`
	// FinishReason 结束原因
	// 值: "stop", "tool_calls", "length", "content_filter"
	FinishReason string `json:"finish_reason"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	// Content 内容片段
	Content string `json:"content"`
	// ToolCalls 工具调用片段（如有）
	ToolCalls []message.ToolCall `json:"tool_calls,omitempty"`
	// Done 是否完成
	Done bool `json:"done"`
	// FinishReason 结束原因（当 Done=true 时）
	FinishReason string `json:"finish_reason,omitempty"`
	// TokenUsage Token 使用统计（当 Done=true 时）
	TokenUsage *message.TokenUsage `json:"token_usage,omitempty"`
}
