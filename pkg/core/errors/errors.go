// Package errors 定义框架的通用错误类型
package errors

import (
	"errors"
	"fmt"
)

// 通用错误
var (
	// ErrNotImplemented 功能未实现
	ErrNotImplemented = errors.New("not implemented")
	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrContextCanceled 上下文被取消
	ErrContextCanceled = errors.New("context canceled")
)

// LLM 相关错误
var (
	// ErrRateLimited 请求被限速
	ErrRateLimited = errors.New("rate limited")
	// ErrTimeout 请求超时
	ErrTimeout = errors.New("request timeout")
	// ErrTokenLimitExceeded Token 限制超出
	ErrTokenLimitExceeded = errors.New("token limit exceeded")
	// ErrInvalidAPIKey API 密钥无效
	ErrInvalidAPIKey = errors.New("invalid API key")
	// ErrModelNotFound 模型未找到
	ErrModelNotFound = errors.New("model not found")
	// ErrProviderUnavailable 提供商不可用
	ErrProviderUnavailable = errors.New("provider unavailable")
	// ErrInvalidResponse LLM 响应无效
	ErrInvalidResponse = errors.New("invalid LLM response")
)

// Agent 相关错误
var (
	// ErrMaxIterationsExceeded 超出最大迭代次数
	ErrMaxIterationsExceeded = errors.New("max iterations exceeded")
	// ErrAgentNotReady Agent 未就绪
	ErrAgentNotReady = errors.New("agent not ready")
	// ErrNoToolsAvailable 没有可用工具
	ErrNoToolsAvailable = errors.New("no tools available")
)

// Tool 相关错误
var (
	// ErrToolNotFound 工具未找到
	ErrToolNotFound = errors.New("tool not found")
	// ErrToolExecutionFailed 工具执行失败
	ErrToolExecutionFailed = errors.New("tool execution failed")
	// ErrInvalidToolArgs 工具参数无效
	ErrInvalidToolArgs = errors.New("invalid tool arguments")
	// ErrToolAlreadyRegistered 工具已注册
	ErrToolAlreadyRegistered = errors.New("tool already registered")
	// ErrInvalidTool 无效的工具
	ErrInvalidTool = errors.New("invalid tool")
	// ErrToolTimeout 工具执行超时
	ErrToolTimeout = errors.New("tool execution timeout")
)

// Memory 相关错误
var (
	// ErrMemoryFull 记忆已满
	ErrMemoryFull = errors.New("memory full")
	// ErrMemoryNotFound 记忆未找到
	ErrMemoryNotFound = errors.New("memory not found")
	// ErrMemoryExpired 记忆已过期
	ErrMemoryExpired = errors.New("memory expired")
)

// RAG 相关错误
var (
	// ErrDocumentNotFound 文档未找到
	ErrDocumentNotFound = errors.New("document not found")
	// ErrEmbeddingFailed 嵌入失败
	ErrEmbeddingFailed = errors.New("embedding failed")
	// ErrVectorStoreFailed 向量存储失败
	ErrVectorStoreFailed = errors.New("vector store operation failed")
)

// WrapError 包装错误并添加上下文信息
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrProviderUnavailable)
}

// IsFatal 判断错误是否为致命错误（不可恢复）
func IsFatal(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrInvalidAPIKey) ||
		errors.Is(err, ErrModelNotFound) ||
		errors.Is(err, ErrInvalidConfig)
}
