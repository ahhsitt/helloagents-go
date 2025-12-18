// Package agents 提供 Agent 的接口定义和实现
package agents

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/config"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

// Agent 定义智能代理的核心接口
//
// Agent 是框架的核心抽象，负责接收用户输入并生成响应。
// 不同的推理模式（Simple、ReAct、Reflection、PlanAndSolve）通过不同实现提供。
type Agent interface {
	// Run 执行 Agent 的主要逻辑
	//
	// 参数:
	//   - ctx: 上下文，用于取消、超时控制和追踪传播
	//   - input: Agent 输入，包含用户查询和上下文信息
	//
	// 返回:
	//   - Output: 包含响应、推理步骤、token 使用量等
	//   - error: 执行错误
	Run(ctx context.Context, input Input) (Output, error)

	// RunStream 以流式方式执行 Agent（用于实时响应）
	//
	// 返回两个 channel：
	//   - <-chan StreamChunk: 流式输出块
	//   - <-chan error: 错误通道（最多一个错误）
	RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error)

	// Name 返回 Agent 名称
	Name() string

	// Config 返回 Agent 配置（只读）
	Config() config.AgentConfig
}

// ToolAware 支持工具的 Agent 接口
type ToolAware interface {
	Agent
	// AddTool 向 Agent 注册工具
	AddTool(tool Tool)
	// AddTools 批量注册工具
	AddTools(tools ...Tool)
	// Tools 返回已注册的工具列表
	Tools() []Tool
}

// MemoryAware 支持记忆的 Agent 接口
type MemoryAware interface {
	Agent
	// SetMemory 设置 Agent 的记忆系统
	SetMemory(memory Memory)
	// Memory 返回当前记忆系统
	Memory() Memory
}

// Tool 工具接口（简化版，完整版在 tools 包）
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// Memory 记忆接口（简化版，完整版在 memory 包）
type Memory interface {
	AddMessage(ctx context.Context, msg message.Message) error
	GetHistory(ctx context.Context, limit int) ([]message.Message, error)
	Clear(ctx context.Context) error
}

// StreamChunk 流式输出块
type StreamChunk struct {
	// Type 块类型
	Type ChunkType `json:"type"`
	// Content 内容片段
	Content string `json:"content"`
	// Step 推理步骤（当 Type=ChunkStep 时）
	Step *ReasoningStep `json:"step,omitempty"`
	// Done 是否完成
	Done bool `json:"done"`
}

// ChunkType 流式块类型
type ChunkType string

const (
	// ChunkTypeText 文本内容
	ChunkTypeText ChunkType = "text"
	// ChunkTypeStep 推理步骤
	ChunkTypeStep ChunkType = "step"
	// ChunkTypeTool 工具调用
	ChunkTypeTool ChunkType = "tool"
	// ChunkTypeError 错误信息
	ChunkTypeError ChunkType = "error"
	// ChunkTypeDone 完成标志
	ChunkTypeDone ChunkType = "done"
)
