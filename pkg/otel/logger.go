package otel

import (
	"context"
	"log/slog"
)

// Logger 定义日志接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, args ...any)
	// Info 信息日志
	Info(msg string, args ...any)
	// Warn 警告日志
	Warn(msg string, args ...any)
	// Error 错误日志
	Error(msg string, args ...any)
	// WithContext 返回带上下文的 Logger（用于关联 Trace ID）
	WithContext(ctx context.Context) Logger
	// WithFields 返回带额外字段的 Logger
	WithFields(fields map[string]any) Logger
}

// SlogLogger slog 适配器
type SlogLogger struct {
	logger *slog.Logger
	attrs  []any
}

// NewSlogLogger 创建 slog 适配器
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogLogger{logger: logger}
}

// Debug 调试日志
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, append(l.attrs, args...)...)
}

// Info 信息日志
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, append(l.attrs, args...)...)
}

// Warn 警告日志
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, append(l.attrs, args...)...)
}

// Error 错误日志
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, append(l.attrs, args...)...)
}

// WithContext 返回带上下文的 Logger
func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	// 从上下文提取 trace ID 和 span ID
	span := SpanFromContext(ctx)
	if span == nil {
		return l
	}

	sc := span.SpanContext()
	if sc.TraceID == "" {
		return l
	}

	return &SlogLogger{
		logger: l.logger,
		attrs: append(l.attrs,
			"trace_id", sc.TraceID,
			"span_id", sc.SpanID,
		),
	}
}

// WithFields 返回带额外字段的 Logger
func (l *SlogLogger) WithFields(fields map[string]any) Logger {
	newAttrs := make([]any, len(l.attrs), len(l.attrs)+len(fields)*2)
	copy(newAttrs, l.attrs)

	for k, v := range fields {
		newAttrs = append(newAttrs, k, v)
	}

	return &SlogLogger{
		logger: l.logger,
		attrs:  newAttrs,
	}
}

// SpanFromContext 从上下文获取 Span（辅助函数）
func SpanFromContext(ctx context.Context) Span {
	if ctx == nil {
		return nil
	}
	// 尝试从全局追踪器获取
	if globalTracer != nil {
		return globalTracer.SpanFromContext(ctx)
	}
	return nil
}

// NoopLogger 空实现日志
type NoopLogger struct{}

// NewNoopLogger 创建空实现日志
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (l *NoopLogger) Debug(msg string, args ...any)                {}
func (l *NoopLogger) Info(msg string, args ...any)                 {}
func (l *NoopLogger) Warn(msg string, args ...any)                 {}
func (l *NoopLogger) Error(msg string, args ...any)                {}
func (l *NoopLogger) WithContext(ctx context.Context) Logger       { return l }
func (l *NoopLogger) WithFields(fields map[string]any) Logger      { return l }

// compile-time interface check
var _ Logger = (*SlogLogger)(nil)
var _ Logger = (*NoopLogger)(nil)
