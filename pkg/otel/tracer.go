// Package otel 提供 OpenTelemetry 可观测性支持
package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracer 定义追踪器接口
//
// 提供统一的追踪接口，支持创建 Span、记录事件和设置属性。
type Tracer interface {
	// Start 开始一个新的 Span
	//
	// 参数:
	//   - ctx: 父上下文
	//   - name: Span 名称
	//   - opts: Span 选项
	//
	// 返回:
	//   - context.Context: 包含 Span 的新上下文
	//   - Span: 创建的 Span
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)

	// SpanFromContext 从上下文中获取当前 Span
	SpanFromContext(ctx context.Context) Span
}

// Span 定义 Span 接口
type Span interface {
	// End 结束 Span
	End()

	// SetAttributes 设置属性
	SetAttributes(attrs ...attribute.KeyValue)

	// AddEvent 添加事件
	AddEvent(name string, attrs ...attribute.KeyValue)

	// RecordError 记录错误
	RecordError(err error)

	// SetStatus 设置状态
	SetStatus(code StatusCode, description string)

	// SpanContext 返回 Span 上下文
	SpanContext() SpanContext
}

// SpanContext Span 上下文信息
type SpanContext struct {
	TraceID string
	SpanID  string
}

// StatusCode Span 状态码
type StatusCode int

const (
	// StatusUnset 未设置
	StatusUnset StatusCode = iota
	// StatusOK 成功
	StatusOK
	// StatusError 错误
	StatusError
)

// SpanOption Span 配置选项
type SpanOption func(*SpanConfig)

// SpanConfig Span 配置
type SpanConfig struct {
	Kind       SpanKind
	Attributes []attribute.KeyValue
}

// SpanKind Span 类型
type SpanKind int

const (
	// SpanKindInternal 内部调用
	SpanKindInternal SpanKind = iota
	// SpanKindServer 服务端
	SpanKindServer
	// SpanKindClient 客户端
	SpanKindClient
	// SpanKindProducer 生产者
	SpanKindProducer
	// SpanKindConsumer 消费者
	SpanKindConsumer
)

// WithSpanKind 设置 Span 类型
func WithSpanKind(kind SpanKind) SpanOption {
	return func(cfg *SpanConfig) {
		cfg.Kind = kind
	}
}

// WithAttributes 设置 Span 属性
func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(cfg *SpanConfig) {
		cfg.Attributes = append(cfg.Attributes, attrs...)
	}
}

// OTelTracer OpenTelemetry 追踪器实现
type OTelTracer struct {
	tracer trace.Tracer
}

// NewTracer 创建 OpenTelemetry 追踪器
func NewTracer(tracer trace.Tracer) *OTelTracer {
	return &OTelTracer{tracer: tracer}
}

// Start 开始一个新的 Span
func (t *OTelTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	cfg := &SpanConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// 转换选项
	spanOpts := []trace.SpanStartOption{
		trace.WithSpanKind(convertSpanKind(cfg.Kind)),
	}
	if len(cfg.Attributes) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(cfg.Attributes...))
	}

	ctx, span := t.tracer.Start(ctx, name, spanOpts...)
	return ctx, &OTelSpan{span: span}
}

// SpanFromContext 从上下文中获取当前 Span
func (t *OTelTracer) SpanFromContext(ctx context.Context) Span {
	span := trace.SpanFromContext(ctx)
	return &OTelSpan{span: span}
}

// convertSpanKind 转换 SpanKind
func convertSpanKind(kind SpanKind) trace.SpanKind {
	switch kind {
	case SpanKindServer:
		return trace.SpanKindServer
	case SpanKindClient:
		return trace.SpanKindClient
	case SpanKindProducer:
		return trace.SpanKindProducer
	case SpanKindConsumer:
		return trace.SpanKindConsumer
	default:
		return trace.SpanKindInternal
	}
}

// OTelSpan OpenTelemetry Span 实现
type OTelSpan struct {
	span trace.Span
}

// End 结束 Span
func (s *OTelSpan) End() {
	s.span.End()
}

// SetAttributes 设置属性
func (s *OTelSpan) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

// AddEvent 添加事件
func (s *OTelSpan) AddEvent(name string, attrs ...attribute.KeyValue) {
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError 记录错误
func (s *OTelSpan) RecordError(err error) {
	s.span.RecordError(err)
}

// SetStatus 设置状态
func (s *OTelSpan) SetStatus(code StatusCode, description string) {
	// 使用 trace.Span 的 SetStatus 方法
	switch code {
	case StatusOK:
		s.span.SetStatus(2, description) // codes.Ok = 2
	case StatusError:
		s.span.SetStatus(1, description) // codes.Error = 1
	default:
		s.span.SetStatus(0, description) // codes.Unset = 0
	}
}

// SpanContext 返回 Span 上下文
func (s *OTelSpan) SpanContext() SpanContext {
	sc := s.span.SpanContext()
	return SpanContext{
		TraceID: sc.TraceID().String(),
		SpanID:  sc.SpanID().String(),
	}
}

// NoopTracer 空实现追踪器（用于禁用追踪）
type NoopTracer struct{}

// NewNoopTracer 创建空实现追踪器
func NewNoopTracer() *NoopTracer {
	return &NoopTracer{}
}

// Start 开始 Span（空实现）
func (t *NoopTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &NoopSpan{}
}

// SpanFromContext 获取 Span（空实现）
func (t *NoopTracer) SpanFromContext(ctx context.Context) Span {
	return &NoopSpan{}
}

// NoopSpan 空实现 Span
type NoopSpan struct{}

func (s *NoopSpan) End()                                                 {}
func (s *NoopSpan) SetAttributes(attrs ...attribute.KeyValue)            {}
func (s *NoopSpan) AddEvent(name string, attrs ...attribute.KeyValue)    {}
func (s *NoopSpan) RecordError(err error)                                {}
func (s *NoopSpan) SetStatus(code StatusCode, description string)        {}
func (s *NoopSpan) SpanContext() SpanContext                             { return SpanContext{} }

// compile-time interface check
var _ Tracer = (*OTelTracer)(nil)
var _ Tracer = (*NoopTracer)(nil)
var _ Span = (*OTelSpan)(nil)
var _ Span = (*NoopSpan)(nil)
