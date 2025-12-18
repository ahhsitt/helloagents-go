// Package config 提供配置加载和管理功能
package config

import (
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Config 全局配置结构
type Config struct {
	// LLM LLM 配置
	LLM LLMConfig `koanf:"llm"`
	// Agent Agent 配置
	Agent AgentConfig `koanf:"agent"`
	// Observability 可观测性配置
	Observability ObservabilityConfig `koanf:"observability"`
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	// Enabled 是否启用
	Enabled bool `koanf:"enabled"`
	// ServiceName 服务名称
	ServiceName string `koanf:"service_name"`
	// TracerEndpoint 追踪端点
	TracerEndpoint string `koanf:"tracer_endpoint"`
	// MetricsEndpoint 指标端点
	MetricsEndpoint string `koanf:"metrics_endpoint"`
	// SampleRate 采样率 [0, 1]
	SampleRate float64 `koanf:"sample_rate"`
}

// Loader 配置加载器
type Loader struct {
	k *koanf.Koanf
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{
		k: koanf.New("."),
	}
}

// LoadFile 从文件加载配置
func (l *Loader) LoadFile(path string) error {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // 文件不存在不报错，使用默认值
	}

	// 根据文件扩展名选择解析器
	// TODO: 实现具体解析器后启用
	switch {
	case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
		// YAML 解析器暂未实现
		return nil
	case strings.HasSuffix(path, ".json"):
		// JSON 解析器暂未实现
		return nil
	default:
		return nil
	}
}

// LoadEnv 从环境变量加载配置
func (l *Loader) LoadEnv(prefix string) error {
	return l.k.Load(env.Provider(prefix, ".", func(s string) string {
		// 转换环境变量名: HELLOAGENTS_LLM_API_KEY -> llm.api_key
		s = strings.TrimPrefix(s, prefix)
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", ".")
		return s
	}), nil)
}

// Unmarshal 解析配置到结构体
func (l *Loader) Unmarshal(cfg *Config) error {
	return l.k.Unmarshal("", cfg)
}

// Get 获取配置值
func (l *Loader) Get(key string) interface{} {
	return l.k.Get(key)
}

// GetString 获取字符串配置值
func (l *Loader) GetString(key string) string {
	return l.k.String(key)
}

// GetInt 获取整数配置值
func (l *Loader) GetInt(key string) int {
	return l.k.Int(key)
}

// GetBool 获取布尔配置值
func (l *Loader) GetBool(key string) bool {
	return l.k.Bool(key)
}

// GetDuration 获取时间间隔配置值
func (l *Loader) GetDuration(key string) time.Duration {
	return l.k.Duration(key)
}

// Load 加载完整配置（文件 + 环境变量）
func Load(configPath string) (*Config, error) {
	loader := NewLoader()

	// 加载配置文件
	if configPath != "" {
		if err := loader.LoadFile(configPath); err != nil {
			return nil, err
		}
	}

	// 加载环境变量（优先级更高）
	if err := loader.LoadEnv("HELLOAGENTS_"); err != nil {
		return nil, err
	}

	// 解析到结构体
	cfg := &Config{}
	if err := loader.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// 应用默认值
	applyDefaults(cfg)

	return cfg, nil
}

// applyDefaults 应用默认配置值
func applyDefaults(cfg *Config) {
	// LLM 默认值
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 30 * time.Second
	}
	if cfg.LLM.MaxRetries == 0 {
		cfg.LLM.MaxRetries = 3
	}
	if cfg.LLM.RetryDelay == 0 {
		cfg.LLM.RetryDelay = time.Second
	}

	// Agent 默认值
	if cfg.Agent.MaxIterations == 0 {
		cfg.Agent.MaxIterations = 10
	}
	if cfg.Agent.Temperature == 0 {
		cfg.Agent.Temperature = 0.7
	}
	if cfg.Agent.MaxTokens == 0 {
		cfg.Agent.MaxTokens = 4096
	}
	if cfg.Agent.Timeout == 0 {
		cfg.Agent.Timeout = 5 * time.Minute
	}

	// Observability 默认值
	if cfg.Observability.SampleRate == 0 {
		cfg.Observability.SampleRate = 1.0
	}
}
