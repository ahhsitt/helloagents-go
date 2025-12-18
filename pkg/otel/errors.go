package otel

import "errors"

// 可观测性相关错误
var (
	// ErrNotInitialized 未初始化
	ErrNotInitialized = errors.New("observability not initialized")
	// ErrAlreadyInitialized 已初始化
	ErrAlreadyInitialized = errors.New("observability already initialized")
	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid observability config")
	// ErrInvalidSampleRate 采样率无效
	ErrInvalidSampleRate = errors.New("sample rate must be between 0 and 1")
	// ErrExportFailed 导出失败
	ErrExportFailed = errors.New("failed to export telemetry data")
)
