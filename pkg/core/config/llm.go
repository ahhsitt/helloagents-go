package config

import "time"

// Provider LLM 提供商类型
type Provider string

const (
	// ProviderOpenAI OpenAI 提供商
	ProviderOpenAI Provider = "openai"
	// ProviderDeepSeek DeepSeek 提供商
	ProviderDeepSeek Provider = "deepseek"
	// ProviderQwen 通义千问提供商
	ProviderQwen Provider = "qwen"
	// ProviderOllama Ollama 提供商
	ProviderOllama Provider = "ollama"
	// ProviderVLLM vLLM 提供商
	ProviderVLLM Provider = "vllm"
)

// IsValid 检查提供商是否有效
func (p Provider) IsValid() bool {
	switch p {
	case ProviderOpenAI, ProviderDeepSeek, ProviderQwen, ProviderOllama, ProviderVLLM:
		return true
	default:
		return false
	}
}

// LLMConfig LLM 配置
type LLMConfig struct {
	// Provider 提供商
	Provider Provider `koanf:"provider"`
	// Model 模型名称
	Model string `koanf:"model"`
	// APIKey API 密钥
	APIKey string `koanf:"api_key"`
	// BaseURL 自定义 API 端点
	BaseURL string `koanf:"base_url"`
	// Timeout 请求超时时间
	// 默认: 30s, 最大: 5m
	Timeout time.Duration `koanf:"timeout"`
	// MaxRetries 最大重试次数
	// 默认: 3, 最大: 10
	MaxRetries int `koanf:"max_retries"`
	// RetryDelay 重试间隔基数
	// 默认: 1s
	RetryDelay time.Duration `koanf:"retry_delay"`
	// EmbeddingModel 嵌入模型名称
	EmbeddingModel string `koanf:"embedding_model"`
	// Fallback 备用提供商配置
	Fallback *LLMConfig `koanf:"fallback"`
}

// Validate 验证 LLM 配置
func (c *LLMConfig) Validate() error {
	if c.Model == "" {
		return ErrModelRequired
	}
	if c.Timeout < 0 {
		return ErrInvalidTimeout
	}
	if c.Timeout > 5*time.Minute {
		c.Timeout = 5 * time.Minute
	}
	if c.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}
	if c.MaxRetries > 10 {
		c.MaxRetries = 10
	}
	return nil
}

// WithDefaults 返回带默认值的配置
func (c LLMConfig) WithDefaults() LLMConfig {
	if c.Provider == "" {
		c.Provider = ProviderOpenAI
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = time.Second
	}
	return c
}
