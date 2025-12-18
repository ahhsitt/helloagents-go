package config

import "errors"

// 配置验证相关错误
var (
	// ErrModelRequired 模型名称必填
	ErrModelRequired = errors.New("model name is required")
	// ErrInvalidTimeout 超时时间无效
	ErrInvalidTimeout = errors.New("invalid timeout value")
	// ErrInvalidMaxRetries 重试次数无效
	ErrInvalidMaxRetries = errors.New("invalid max retries value")
	// ErrNameRequired Agent 名称必填
	ErrNameRequired = errors.New("agent name is required")
	// ErrInvalidMaxIterations 迭代次数无效
	ErrInvalidMaxIterations = errors.New("max iterations must be between 1 and 100")
	// ErrInvalidTemperature 温度值无效
	ErrInvalidTemperature = errors.New("temperature must be between 0 and 2")
	// ErrInvalidMaxTokens Token 数无效
	ErrInvalidMaxTokens = errors.New("max tokens must be positive")
)
