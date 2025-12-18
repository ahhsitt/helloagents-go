package otel

import "time"

// Config 可观测性配置
type Config struct {
	// Enabled 是否启用可观测性
	Enabled bool `koanf:"enabled"`

	// ServiceName 服务名称
	ServiceName string `koanf:"service_name"`
	// ServiceVersion 服务版本
	ServiceVersion string `koanf:"service_version"`
	// Environment 环境（dev, staging, prod）
	Environment string `koanf:"environment"`

	// Tracing 追踪配置
	Tracing TracingConfig `koanf:"tracing"`
	// Metrics 指标配置
	Metrics MetricsConfig `koanf:"metrics"`
	// Logging 日志配置
	Logging LoggingConfig `koanf:"logging"`
}

// TracingConfig 追踪配置
type TracingConfig struct {
	// Enabled 是否启用追踪
	Enabled bool `koanf:"enabled"`
	// Endpoint OTLP 端点
	Endpoint string `koanf:"endpoint"`
	// Insecure 是否使用不安全连接
	Insecure bool `koanf:"insecure"`
	// SampleRate 采样率 (0.0-1.0)
	SampleRate float64 `koanf:"sample_rate"`
	// Timeout 导出超时
	Timeout time.Duration `koanf:"timeout"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	// Enabled 是否启用指标
	Enabled bool `koanf:"enabled"`
	// Endpoint OTLP 端点
	Endpoint string `koanf:"endpoint"`
	// Insecure 是否使用不安全连接
	Insecure bool `koanf:"insecure"`
	// Interval 导出间隔
	Interval time.Duration `koanf:"interval"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	// Level 日志级别 (debug, info, warn, error)
	Level string `koanf:"level"`
	// Format 日志格式 (text, json)
	Format string `koanf:"format"`
	// IncludeTraceID 是否包含 Trace ID
	IncludeTraceID bool `koanf:"include_trace_id"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		ServiceName:    "helloagents",
		ServiceVersion: "0.1.0",
		Environment:    "development",
		Tracing: TracingConfig{
			Enabled:    false,
			Endpoint:   "localhost:4317",
			Insecure:   true,
			SampleRate: 1.0,
			Timeout:    30 * time.Second,
		},
		Metrics: MetricsConfig{
			Enabled:  false,
			Endpoint: "localhost:4317",
			Insecure: true,
			Interval: 60 * time.Second,
		},
		Logging: LoggingConfig{
			Level:          "info",
			Format:         "text",
			IncludeTraceID: true,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Tracing.SampleRate < 0 || c.Tracing.SampleRate > 1 {
		return ErrInvalidSampleRate
	}
	return nil
}

// WithDefaults 返回带默认值的配置
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()

	if c.ServiceName == "" {
		c.ServiceName = defaults.ServiceName
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = defaults.ServiceVersion
	}
	if c.Environment == "" {
		c.Environment = defaults.Environment
	}
	if c.Tracing.Endpoint == "" {
		c.Tracing.Endpoint = defaults.Tracing.Endpoint
	}
	if c.Tracing.SampleRate == 0 {
		c.Tracing.SampleRate = defaults.Tracing.SampleRate
	}
	if c.Tracing.Timeout == 0 {
		c.Tracing.Timeout = defaults.Tracing.Timeout
	}
	if c.Metrics.Endpoint == "" {
		c.Metrics.Endpoint = defaults.Metrics.Endpoint
	}
	if c.Metrics.Interval == 0 {
		c.Metrics.Interval = defaults.Metrics.Interval
	}
	if c.Logging.Level == "" {
		c.Logging.Level = defaults.Logging.Level
	}
	if c.Logging.Format == "" {
		c.Logging.Format = defaults.Logging.Format
	}

	return c
}
