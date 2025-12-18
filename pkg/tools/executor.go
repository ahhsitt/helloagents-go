package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/errors"
)

// Executor 工具执行器
//
// 负责执行工具调用，支持超时控制和重试机制。
type Executor struct {
	registry   *Registry
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration
}

// ExecutorOption 执行器配置选项
type ExecutorOption func(*Executor)

// NewExecutor 创建工具执行器
func NewExecutor(registry *Registry, opts ...ExecutorOption) *Executor {
	e := &Executor{
		registry:   registry,
		timeout:    30 * time.Second,
		maxRetries: 0,
		retryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// WithExecutorTimeout 设置执行超时时间
func WithExecutorTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = d
	}
}

// WithExecutorRetries 设置重试次数和间隔
func WithExecutorRetries(maxRetries int, delay time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.maxRetries = maxRetries
		e.retryDelay = delay
	}
}

// Execute 执行工具
//
// 参数:
//   - ctx: 上下文
//   - name: 工具名称
//   - args: 工具参数
//
// 返回:
//   - ToolResult: 执行结果
func (e *Executor) Execute(ctx context.Context, name string, args map[string]interface{}) ToolResult {
	tool, err := e.registry.Get(name)
	if err != nil {
		return NewToolError(name, err)
	}

	// 应用超时
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// 执行（带重试）
	var result string
	var lastErr error

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		// 检查上下文
		select {
		case <-ctx.Done():
			return NewToolError(name, errors.ErrContextCanceled)
		default:
		}

		// 参数验证
		if validator, ok := tool.(ToolWithValidation); ok {
			if err := validator.Validate(args); err != nil {
				return NewToolError(name, fmt.Errorf("%w: %v", errors.ErrInvalidToolArgs, err))
			}
		}

		// 执行工具
		result, lastErr = tool.Execute(ctx, args)
		if lastErr == nil {
			return NewToolResult(name, result)
		}

		// 检查是否需要重试
		if attempt < e.maxRetries {
			select {
			case <-ctx.Done():
				return NewToolError(name, errors.ErrContextCanceled)
			case <-time.After(e.retryDelay):
			}
		}
	}

	return NewToolError(name, fmt.Errorf("%w: %v", errors.ErrToolExecutionFailed, lastErr))
}

// ExecuteBatch 批量执行工具
//
// 按顺序执行多个工具调用。
func (e *Executor) ExecuteBatch(ctx context.Context, calls []ToolCall) []ToolResult {
	results := make([]ToolResult, len(calls))

	for i, call := range calls {
		select {
		case <-ctx.Done():
			// 填充剩余结果为取消错误
			for j := i; j < len(calls); j++ {
				results[j] = NewToolError(calls[j].Name, errors.ErrContextCanceled)
			}
			return results
		default:
			results[i] = e.Execute(ctx, call.Name, call.Args)
		}
	}

	return results
}

// ToolCall 工具调用请求
type ToolCall struct {
	// ID 调用唯一标识
	ID string
	// Name 工具名称
	Name string
	// Args 参数
	Args map[string]interface{}
}
