package otel

import (
	"context"
	"log/slog"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Provider 可观测性提供者
//
// 管理追踪、指标和日志的生命周期。
type Provider struct {
	config   Config
	tracer   Tracer
	metrics  Metrics
	logger   Logger
	shutdown []func(context.Context) error
	mu       sync.RWMutex
}

var (
	globalProvider *Provider
	globalTracer   Tracer
	globalMu       sync.RWMutex
)

// NewProvider 创建可观测性提供者
func NewProvider(cfg Config) (*Provider, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Provider{
		config:   cfg,
		shutdown: make([]func(context.Context) error, 0),
	}

	if !cfg.Enabled {
		p.tracer = NewNoopTracer()
		p.metrics = NewNoopMetrics()
		p.logger = NewNoopLogger()
		return p, nil
	}

	// 初始化追踪
	if cfg.Tracing.Enabled {
		if err := p.initTracing(); err != nil {
			return nil, err
		}
	} else {
		p.tracer = NewNoopTracer()
	}

	// 初始化指标
	if cfg.Metrics.Enabled {
		p.metrics = NewInMemoryMetrics() // 简化实现，实际应使用 OTLP
	} else {
		p.metrics = NewNoopMetrics()
	}

	// 初始化日志
	p.logger = NewSlogLogger(slog.Default())

	return p, nil
}

// initTracing 初始化追踪
func (p *Provider) initTracing() error {
	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(p.config.ServiceName),
			semconv.ServiceVersionKey.String(p.config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(p.config.Environment),
		),
	)
	if err != nil {
		return err
	}

	// 创建采样器
	var sampler sdktrace.Sampler
	if p.config.Tracing.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if p.config.Tracing.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(p.config.Tracing.SampleRate)
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 注册关闭函数
	p.shutdown = append(p.shutdown, tp.Shutdown)

	// 创建 Tracer
	p.tracer = NewTracer(tp.Tracer(p.config.ServiceName))

	return nil
}

// Tracer 返回追踪器
func (p *Provider) Tracer() Tracer {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tracer
}

// Metrics 返回指标收集器
func (p *Provider) Metrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// Logger 返回日志器
func (p *Provider) Logger() Logger {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.logger
}

// Shutdown 优雅关闭
func (p *Provider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, fn := range p.shutdown {
		if err := fn(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// SetGlobal 设置全局提供者
func SetGlobal(p *Provider) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalProvider = p
	globalTracer = p.Tracer()
}

// Global 获取全局提供者
func Global() *Provider {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalProvider
}

// GetTracer 获取全局追踪器
func GetTracer() Tracer {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalTracer != nil {
		return globalTracer
	}
	return NewNoopTracer()
}

// GetMetrics 获取全局指标收集器
func GetMetrics() Metrics {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalProvider != nil {
		return globalProvider.Metrics()
	}
	return NewNoopMetrics()
}

// GetLogger 获取全局日志器
func GetLogger() Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalProvider != nil {
		return globalProvider.Logger()
	}
	return NewNoopLogger()
}

// MustInit 初始化全局可观测性（失败则 panic）
func MustInit(cfg Config) *Provider {
	p, err := NewProvider(cfg)
	if err != nil {
		panic(err)
	}
	SetGlobal(p)
	return p
}
