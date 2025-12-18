// Package tools 提供工具系统的接口定义和实现
package tools

import (
	"context"
)

// Tool 定义工具的核心接口
//
// 工具是 Agent 与外部世界交互的方式。通过实现此接口，
// 开发者可以让 Agent 执行各种操作，如调用 API、执行计算等。
type Tool interface {
	// Name 返回工具唯一名称
	// 名称用于 LLM Function Calling 识别和调用
	Name() string

	// Description 返回工具描述
	// 描述应清晰说明工具的功能，帮助 LLM 理解何时使用此工具
	Description() string

	// Parameters 返回参数 Schema
	// 遵循 JSON Schema 格式，用于 Function Calling 参数验证
	Parameters() ParameterSchema

	// Execute 执行工具
	//
	// 参数:
	//   - ctx: 上下文，用于超时和取消控制
	//   - args: 工具参数（由 LLM 提供）
	//
	// 返回:
	//   - string: 工具执行结果（将返回给 LLM）
	//   - error: 执行错误
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolWithValidation 支持参数验证的工具接口
type ToolWithValidation interface {
	Tool
	// Validate 验证参数
	Validate(args map[string]interface{}) error
}

// AsyncTool 支持异步执行的工具接口
type AsyncTool interface {
	Tool
	// ExecuteAsync 异步执行工具
	// 返回结果 channel 和错误 channel
	ExecuteAsync(ctx context.Context, args map[string]interface{}) (<-chan string, <-chan error)
}
