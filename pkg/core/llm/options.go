package llm

import "time"

// Option LLM 配置选项函数
type Option func(*Options)

// Options LLM 配置选项
type Options struct {
	// APIKey API 密钥
	APIKey string
	// BaseURL 自定义 API 端点
	BaseURL string
	// Model 模型名称
	Model string
	// Timeout 请求超时
	Timeout time.Duration
	// MaxRetries 最大重试次数
	MaxRetries int
	// RetryDelay 重试间隔基数
	RetryDelay time.Duration
	// Temperature 默认温度
	Temperature float64
	// MaxTokens 默认最大 token
	MaxTokens int
	// EmbeddingModel 嵌入模型
	EmbeddingModel string
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  time.Second,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}

// WithAPIKey 设置 API 密钥
func WithAPIKey(key string) Option {
	return func(o *Options) {
		o.APIKey = key
	}
}

// WithBaseURL 设置自定义端点
func WithBaseURL(url string) Option {
	return func(o *Options) {
		o.BaseURL = url
	}
}

// WithModel 设置模型
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithTimeout 设置超时时间
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) Option {
	return func(o *Options) {
		o.MaxRetries = n
	}
}

// WithRetryDelay 设置重试间隔
func WithRetryDelay(d time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = d
	}
}

// WithTemperature 设置默认温度
func WithTemperature(t float64) Option {
	return func(o *Options) {
		o.Temperature = t
	}
}

// WithMaxTokens 设置默认最大 token
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.MaxTokens = n
	}
}

// WithEmbeddingModel 设置嵌入模型
func WithEmbeddingModel(model string) Option {
	return func(o *Options) {
		o.EmbeddingModel = model
	}
}

// RequestOption 请求选项函数
type RequestOption func(*Request)

// WithRequestTemperature 设置请求温度
func WithRequestTemperature(t float64) RequestOption {
	return func(r *Request) {
		r.Temperature = &t
	}
}

// WithRequestMaxTokens 设置请求最大 token
func WithRequestMaxTokens(n int) RequestOption {
	return func(r *Request) {
		r.MaxTokens = &n
	}
}

// WithTools 设置可用工具
func WithTools(tools []ToolDefinition) RequestOption {
	return func(r *Request) {
		r.Tools = tools
	}
}

// WithToolChoice 设置工具选择策略
func WithToolChoice(choice interface{}) RequestOption {
	return func(r *Request) {
		r.ToolChoice = choice
	}
}

// WithStop 设置停止序列
func WithStop(stop []string) RequestOption {
	return func(r *Request) {
		r.Stop = stop
	}
}
