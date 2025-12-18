package agents

import "time"

// Option Agent 配置选项函数
type Option func(*AgentOptions)

// AgentOptions Agent 配置选项
type AgentOptions struct {
	Name          string
	SystemPrompt  string
	MaxIterations int
	Temperature   float64
	MaxTokens     int
	Timeout       time.Duration
}

// DefaultAgentOptions 返回默认选项
func DefaultAgentOptions() *AgentOptions {
	return &AgentOptions{
		Name:          "Agent",
		MaxIterations: 10,
		Temperature:   0.7,
		MaxTokens:     4096,
		Timeout:       5 * time.Minute,
	}
}

// WithName 设置 Agent 名称
func WithName(name string) Option {
	return func(o *AgentOptions) {
		o.Name = name
	}
}

// WithSystemPrompt 设置系统提示词
func WithSystemPrompt(prompt string) Option {
	return func(o *AgentOptions) {
		o.SystemPrompt = prompt
	}
}

// WithMaxIterations 设置最大迭代次数
func WithMaxIterations(n int) Option {
	return func(o *AgentOptions) {
		o.MaxIterations = n
	}
}

// WithAgentTemperature 设置温度参数
func WithAgentTemperature(t float64) Option {
	return func(o *AgentOptions) {
		o.Temperature = t
	}
}

// WithAgentMaxTokens 设置最大 token 数
func WithAgentMaxTokens(n int) Option {
	return func(o *AgentOptions) {
		o.MaxTokens = n
	}
}

// WithAgentTimeout 设置超时时间
func WithAgentTimeout(d time.Duration) Option {
	return func(o *AgentOptions) {
		o.Timeout = d
	}
}
