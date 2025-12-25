package context

// Config 保存上下文构建的配置。
type Config struct {
	// MaxTokens 是上下文的总 Token 预算。
	MaxTokens int

	// ReserveRatio 是为 LLM 响应预留的 Token 百分比（0.0-1.0）。
	// 默认值为 0.15（预留 15% 用于生成）。
	ReserveRatio float64

	// MinRelevance 是包被纳入上下文的最低相关性分数。
	// 分数低于此阈值的包将被过滤掉。
	MinRelevance float64

	// EnableMMR 启用最大边际相关性（MMR）以增加多样性。
	EnableMMR bool

	// MMRLambda 平衡相关性与多样性（0=纯多样性，1=纯相关性）。
	MMRLambda float64

	// EnableCompression 在超出预算时启用上下文压缩。
	EnableCompression bool

	// RelevanceWeight 是复合评分中相关性分数的权重。
	RelevanceWeight float64

	// RecencyWeight 是复合评分中新近性分数的权重。
	RecencyWeight float64

	// RecencyTau 是新近性衰减的时间常数（秒）。
	// 默认值为 3600（1 小时）。
	RecencyTau float64

	// TokenCounter 是要使用的 Token 计数器。
	TokenCounter TokenCounter

	// MaxHistoryMessages 限制要包含的历史消息数量。
	MaxHistoryMessages int

	// OutputTemplate 是可选的输出格式指令模板。
	OutputTemplate string
}

// ConfigOption 配置 Config。
type ConfigOption func(*Config)

// WithMaxTokens 设置最大 Token 预算。
func WithMaxTokens(tokens int) ConfigOption {
	return func(c *Config) {
		c.MaxTokens = tokens
	}
}

// WithReserveRatio 设置生成预留比例。
func WithReserveRatio(ratio float64) ConfigOption {
	return func(c *Config) {
		c.ReserveRatio = ratio
	}
}

// WithMinRelevance 设置最低相关性阈值。
func WithMinRelevance(threshold float64) ConfigOption {
	return func(c *Config) {
		c.MinRelevance = threshold
	}
}

// WithMMR 启用指定 lambda 值的 MMR。
func WithMMR(lambda float64) ConfigOption {
	return func(c *Config) {
		c.EnableMMR = true
		c.MMRLambda = lambda
	}
}

// WithCompression 启用或禁用压缩。
func WithCompression(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableCompression = enabled
	}
}

// WithScoringWeights 设置复合评分的权重。
func WithScoringWeights(relevance, recency float64) ConfigOption {
	return func(c *Config) {
		c.RelevanceWeight = relevance
		c.RecencyWeight = recency
	}
}

// WithRecencyTau 设置新近性衰减的时间常数。
func WithRecencyTau(tau float64) ConfigOption {
	return func(c *Config) {
		c.RecencyTau = tau
	}
}

// WithTokenCounter 设置 Token 计数器。
func WithTokenCounter(counter TokenCounter) ConfigOption {
	return func(c *Config) {
		c.TokenCounter = counter
	}
}

// WithMaxHistoryMessages 设置最大历史消息数量。
func WithMaxHistoryMessages(n int) ConfigOption {
	return func(c *Config) {
		c.MaxHistoryMessages = n
	}
}

// WithOutputTemplate 设置输出格式模板。
func WithOutputTemplate(template string) ConfigOption {
	return func(c *Config) {
		c.OutputTemplate = template
	}
}

// DefaultConfig 返回具有合理默认值的 Config。
func DefaultConfig() *Config {
	return &Config{
		MaxTokens:          8000,
		ReserveRatio:       0.15,
		MinRelevance:       0.3,
		EnableMMR:          false,
		MMRLambda:          0.7,
		EnableCompression:  true,
		RelevanceWeight:    0.7,
		RecencyWeight:      0.3,
		RecencyTau:         3600, // 1 小时
		TokenCounter:       nil,  // 需要时使用 DefaultTokenCounter()
		MaxHistoryMessages: 10,
		OutputTemplate:     defaultOutputTemplate,
	}
}

// NewConfig 使用给定的选项创建新的 Config。
func NewConfig(opts ...ConfigOption) *Config {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetAvailableTokens 返回 Token 预算减去预留量。
func (c *Config) GetAvailableTokens() int {
	return int(float64(c.MaxTokens) * (1 - c.ReserveRatio))
}

// GetTokenCounter 返回配置的 Token 计数器或默认计数器。
func (c *Config) GetTokenCounter() TokenCounter {
	if c.TokenCounter != nil {
		return c.TokenCounter
	}
	return DefaultTokenCounter()
}

// defaultOutputTemplate 是默认的输出格式指令。
const defaultOutputTemplate = `请按以下格式回答：
1. 结论（简洁明确）
2. 依据（列出支撑证据及来源）
3. 风险与假设（如有）
4. 下一步行动建议（如适用）`
