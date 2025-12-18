package llm

import (
	"fmt"
	"os"
	"strings"
)

// ProviderType 提���商类型
type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderQwen     ProviderType = "qwen"
	ProviderOllama   ProviderType = "ollama"
	ProviderVLLM     ProviderType = "vllm"
)

// ProviderConfig 提供商配置
type ProviderConfig struct {
	// Type 提供商类型
	Type ProviderType `json:"type" yaml:"type"`
	// APIKey API 密钥
	APIKey string `json:"api_key" yaml:"api_key"`
	// BaseURL 基础 URL（可选）
	BaseURL string `json:"base_url" yaml:"base_url"`
	// Model 模型名称
	Model string `json:"model" yaml:"model"`
	// Enabled 是否启用
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// Factory LLM 提供商工厂
type Factory struct {
	configs map[ProviderType]ProviderConfig
}

// NewFactory 创建 LLM 工厂
func NewFactory() *Factory {
	return &Factory{
		configs: make(map[ProviderType]ProviderConfig),
	}
}

// RegisterProvider 注册提供商配置
func (f *Factory) RegisterProvider(config ProviderConfig) {
	f.configs[config.Type] = config
}

// Create 创建指定类型的提供商
func (f *Factory) Create(providerType ProviderType) (Provider, error) {
	config, ok := f.configs[providerType]
	if !ok {
		// 尝试从环境变量获取配置
		config = f.detectFromEnv(providerType)
	}

	return f.createFromConfig(config)
}

// CreateDefault 创建默认提供商
//
// 按以下顺序尝试创建：OpenAI -> DeepSeek -> Qwen -> Ollama -> vLLM
func (f *Factory) CreateDefault() (Provider, error) {
	// 优先使用已注册的配置
	for _, providerType := range []ProviderType{
		ProviderOpenAI,
		ProviderDeepSeek,
		ProviderQwen,
		ProviderOllama,
		ProviderVLLM,
	} {
		if config, ok := f.configs[providerType]; ok && config.Enabled {
			provider, err := f.createFromConfig(config)
			if err == nil {
				return provider, nil
			}
		}
	}

	// 从环境变量自动检测
	return f.AutoDetect()
}

// AutoDetect 自动检测并创建提供商
func (f *Factory) AutoDetect() (Provider, error) {
	// OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return NewOpenAI(
			WithAPIKey(apiKey),
			WithModel(getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini")),
		)
	}

	// DeepSeek
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		return NewDeepSeek(
			WithAPIKey(apiKey),
			WithModel(getEnvOrDefault("DEEPSEEK_MODEL", "deepseek-chat")),
		)
	}

	// Qwen
	if apiKey := os.Getenv("QWEN_API_KEY"); apiKey != "" {
		return NewQwenClient(
			WithQwenAPIKey(apiKey),
			WithQwenModel(getEnvOrDefault("QWEN_MODEL", "qwen-turbo")),
		), nil
	}

	// Ollama（无需 API Key）
	if baseURL := os.Getenv("OLLAMA_BASE_URL"); baseURL != "" {
		return NewOllamaClient(
			WithOllamaBaseURL(baseURL),
			WithOllamaModel(getEnvOrDefault("OLLAMA_MODEL", "llama3.2")),
		), nil
	}

	// 默认尝试本地 Ollama
	if isOllamaAvailable() {
		return NewOllamaClient(), nil
	}

	// vLLM
	if baseURL := os.Getenv("VLLM_BASE_URL"); baseURL != "" {
		return NewVLLMClient(
			WithVLLMBaseURL(baseURL),
			WithVLLMModel(getEnvOrDefault("VLLM_MODEL", "default")),
		), nil
	}

	return nil, fmt.Errorf("no LLM provider configured, set OPENAI_API_KEY, DEEPSEEK_API_KEY, QWEN_API_KEY, OLLAMA_BASE_URL, or VLLM_BASE_URL")
}

// detectFromEnv 从环境变量检测配置
func (f *Factory) detectFromEnv(providerType ProviderType) ProviderConfig {
	config := ProviderConfig{
		Type:    providerType,
		Enabled: true,
	}

	prefix := strings.ToUpper(string(providerType))

	config.APIKey = os.Getenv(prefix + "_API_KEY")
	config.BaseURL = os.Getenv(prefix + "_BASE_URL")
	config.Model = os.Getenv(prefix + "_MODEL")

	return config
}

// createFromConfig 从配置创建提供商
func (f *Factory) createFromConfig(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderOpenAI:
		opts := []Option{}
		if config.APIKey != "" {
			opts = append(opts, WithAPIKey(config.APIKey))
		}
		if config.BaseURL != "" {
			opts = append(opts, WithBaseURL(config.BaseURL))
		}
		if config.Model != "" {
			opts = append(opts, WithModel(config.Model))
		}
		return NewOpenAI(opts...)

	case ProviderDeepSeek:
		opts := []Option{}
		if config.APIKey != "" {
			opts = append(opts, WithAPIKey(config.APIKey))
		}
		if config.Model != "" {
			opts = append(opts, WithModel(config.Model))
		}
		return NewDeepSeek(opts...)

	case ProviderQwen:
		opts := []QwenOption{}
		if config.APIKey != "" {
			opts = append(opts, WithQwenAPIKey(config.APIKey))
		}
		if config.BaseURL != "" {
			opts = append(opts, WithQwenBaseURL(config.BaseURL))
		}
		if config.Model != "" {
			opts = append(opts, WithQwenModel(config.Model))
		}
		return NewQwenClient(opts...), nil

	case ProviderOllama:
		opts := []OllamaOption{}
		if config.BaseURL != "" {
			opts = append(opts, WithOllamaBaseURL(config.BaseURL))
		}
		if config.Model != "" {
			opts = append(opts, WithOllamaModel(config.Model))
		}
		return NewOllamaClient(opts...), nil

	case ProviderVLLM:
		opts := []VLLMOption{}
		if config.APIKey != "" {
			opts = append(opts, WithVLLMAPIKey(config.APIKey))
		}
		if config.BaseURL != "" {
			opts = append(opts, WithVLLMBaseURL(config.BaseURL))
		}
		if config.Model != "" {
			opts = append(opts, WithVLLMModel(config.Model))
		}
		return NewVLLMClient(opts...), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.Type)
	}
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isOllamaAvailable 检查 Ollama 是否可用
func isOllamaAvailable() bool {
	// 简单检查默认端口是否可连接
	// 实际实现中可以做更完整的健康检查
	return false // 默认不自动启用
}
