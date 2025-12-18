package config

import "time"

// AgentConfig Agent 配置
type AgentConfig struct {
	// Name Agent 名称
	Name string `koanf:"name"`
	// SystemPrompt 系统提示词
	SystemPrompt string `koanf:"system_prompt"`
	// MaxIterations 最大迭代次数（ReAct 等模式）
	// 默认: 10, 范围: [1, 100]
	MaxIterations int `koanf:"max_iterations"`
	// Temperature LLM 温度参数
	// 默认: 0.7, 范围: [0, 2]
	Temperature float64 `koanf:"temperature"`
	// MaxTokens 最大输出 token 数
	// 默认: 4096
	MaxTokens int `koanf:"max_tokens"`
	// Timeout 执行超时时间
	// 默认: 5m
	Timeout time.Duration `koanf:"timeout"`
}

// Validate 验证 Agent 配置
func (c *AgentConfig) Validate() error {
	if c.Name == "" {
		return ErrNameRequired
	}
	if c.MaxIterations < 1 || c.MaxIterations > 100 {
		return ErrInvalidMaxIterations
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return ErrInvalidTemperature
	}
	if c.MaxTokens < 1 {
		return ErrInvalidMaxTokens
	}
	return nil
}

// WithDefaults 返回带默认值的配置
func (c AgentConfig) WithDefaults() AgentConfig {
	if c.Name == "" {
		c.Name = "Agent"
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 10
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 4096
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}
	return c
}
