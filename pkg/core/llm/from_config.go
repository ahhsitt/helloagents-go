package llm

import (
	"fmt"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/config"
)

// FromConfig 从配置创建 LLM Provider
func FromConfig(cfg config.LLMConfig) (Provider, error) {
	cfg = cfg.WithDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建主提供商
	primary, err := createProviderFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	// 如果有备用配置，创建 FallbackProvider
	if cfg.Fallback != nil {
		fallback, err := FromConfig(*cfg.Fallback)
		if err != nil {
			return nil, fmt.Errorf("failed to create fallback provider: %w", err)
		}
		return NewFallbackProvider(primary, []Provider{fallback}), nil
	}

	return primary, nil
}

// createProviderFromConfig 根据配置创建特定提供商
func createProviderFromConfig(cfg config.LLMConfig) (Provider, error) {
	switch cfg.Provider {
	case config.ProviderOpenAI:
		return createOpenAIFromConfig(cfg)
	case config.ProviderDeepSeek:
		return createDeepSeekFromConfig(cfg)
	case config.ProviderQwen:
		return createQwenFromConfig(cfg)
	case config.ProviderOllama:
		return createOllamaFromConfig(cfg)
	case config.ProviderVLLM:
		return createVLLMFromConfig(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// createOpenAIFromConfig 从配置创建 OpenAI 客户端
func createOpenAIFromConfig(cfg config.LLMConfig) (*OpenAIClient, error) {
	opts := []Option{
		WithModel(cfg.Model),
		WithTimeout(cfg.Timeout),
		WithMaxRetries(cfg.MaxRetries),
		WithRetryDelay(cfg.RetryDelay),
	}

	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}

	return NewOpenAI(opts...)
}

// createDeepSeekFromConfig 从配置创建 DeepSeek 客户端
func createDeepSeekFromConfig(cfg config.LLMConfig) (*DeepSeekClient, error) {
	opts := []Option{
		WithTimeout(cfg.Timeout),
		WithMaxRetries(cfg.MaxRetries),
		WithRetryDelay(cfg.RetryDelay),
	}

	if cfg.APIKey != "" {
		opts = append(opts, WithAPIKey(cfg.APIKey))
	}
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}

	return NewDeepSeek(opts...)
}

// createQwenFromConfig 从配置创建通义千问客户端
func createQwenFromConfig(cfg config.LLMConfig) (*QwenClient, error) {
	opts := []QwenOption{}

	if cfg.APIKey != "" {
		opts = append(opts, WithQwenAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, WithQwenBaseURL(cfg.BaseURL))
	}
	if cfg.Model != "" {
		opts = append(opts, WithQwenModel(cfg.Model))
	}

	return NewQwenClient(opts...), nil
}

// createOllamaFromConfig 从配置创建 Ollama 客户端
func createOllamaFromConfig(cfg config.LLMConfig) (*OllamaClient, error) {
	opts := []OllamaOption{}

	if cfg.BaseURL != "" {
		opts = append(opts, WithOllamaBaseURL(cfg.BaseURL))
	}
	if cfg.Model != "" {
		opts = append(opts, WithOllamaModel(cfg.Model))
	}

	return NewOllamaClient(opts...), nil
}

// createVLLMFromConfig 从配置创建 vLLM 客户端
func createVLLMFromConfig(cfg config.LLMConfig) (*VLLMClient, error) {
	opts := []VLLMOption{}

	if cfg.APIKey != "" {
		opts = append(opts, WithVLLMAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, WithVLLMBaseURL(cfg.BaseURL))
	}
	if cfg.Model != "" {
		opts = append(opts, WithVLLMModel(cfg.Model))
	}

	return NewVLLMClient(opts...), nil
}

// MustFromConfig 从配置创建 Provider，失败时 panic
func MustFromConfig(cfg config.LLMConfig) Provider {
	provider, err := FromConfig(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create provider from config: %v", err))
	}
	return provider
}

// DefaultConfig 返回默认配置
func DefaultConfig() config.LLMConfig {
	return config.LLMConfig{
		Provider:   config.ProviderOpenAI,
		Model:      "gpt-4o-mini",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// OllamaDefaultConfig 返回 Ollama 默认配置
func OllamaDefaultConfig() config.LLMConfig {
	return config.LLMConfig{
		Provider:   config.ProviderOllama,
		Model:      "llama3.2",
		BaseURL:    "http://localhost:11434",
		Timeout:    5 * time.Minute,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// DeepSeekDefaultConfig 返回 DeepSeek 默认配置
func DeepSeekDefaultConfig() config.LLMConfig {
	return config.LLMConfig{
		Provider:   config.ProviderDeepSeek,
		Model:      "deepseek-chat",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// QwenDefaultConfig 返回通义千问默认配置
func QwenDefaultConfig() config.LLMConfig {
	return config.LLMConfig{
		Provider:   config.ProviderQwen,
		Model:      "qwen-turbo",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}
