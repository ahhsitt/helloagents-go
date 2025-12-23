package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ExporterType 导出器类型
type ExporterType string

const (
	// ExporterOTLPGRPC OTLP gRPC 导出器
	ExporterOTLPGRPC ExporterType = "otlp-grpc"
	// ExporterOTLPHTTP OTLP HTTP 导出器
	ExporterOTLPHTTP ExporterType = "otlp-http"
	// ExporterStdout 标准输出导出器（用于调试）
	ExporterStdout ExporterType = "stdout"
	// ExporterNone 无导出器
	ExporterNone ExporterType = "none"

	// compressionGzip gzip 压缩类型
	compressionGzip = "gzip"
)

// ExporterConfig 导出器配置
type ExporterConfig struct {
	// Type 导出器类型
	Type ExporterType `json:"type" yaml:"type"`
	// Endpoint OTLP 端点（如 "localhost:4317"）
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	// Insecure 是否使用不安全连接
	Insecure bool `json:"insecure" yaml:"insecure"`
	// Headers 请求头
	Headers map[string]string `json:"headers" yaml:"headers"`
	// Timeout 连接超时
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
	// Compression 压缩类型（"gzip" 或 ""）
	Compression string `json:"compression" yaml:"compression"`
}

// DefaultExporterConfig 返回默认导出器配置
func DefaultExporterConfig() ExporterConfig {
	return ExporterConfig{
		Type:     ExporterOTLPGRPC,
		Endpoint: "localhost:4317",
		Insecure: true,
		Timeout:  10 * time.Second,
	}
}

// CreateTraceExporter 创建追踪导出器
func CreateTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Type {
	case ExporterOTLPGRPC:
		return CreateOTLPGRPCTraceExporter(ctx, cfg)
	case ExporterOTLPHTTP:
		return CreateOTLPHTTPTraceExporter(ctx, cfg)
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case ExporterNone:
		return &NoopSpanExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported trace exporter type: %s", cfg.Type)
	}
}

// CreateOTLPGRPCTraceExporter 创建 OTLP gRPC 追踪导出器
func CreateOTLPGRPCTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if cfg.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(cfg.Timeout))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == compressionGzip {
		opts = append(opts, otlptracegrpc.WithCompressor(compressionGzip))
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// CreateOTLPHTTPTraceExporter 创建 OTLP HTTP 追踪导出器
func CreateOTLPHTTPTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if cfg.Timeout > 0 {
		opts = append(opts, otlptracehttp.WithTimeout(cfg.Timeout))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == compressionGzip {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}

	return otlptracehttp.New(ctx, opts...)
}

// CreateMetricExporter 创建指标导出器
func CreateMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	switch cfg.Type {
	case ExporterOTLPGRPC:
		return CreateOTLPGRPCMetricExporter(ctx, cfg)
	case ExporterOTLPHTTP:
		return CreateOTLPHTTPMetricExporter(ctx, cfg)
	case ExporterStdout:
		return stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	case ExporterNone:
		return &NoopMetricExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported metric exporter type: %s", cfg.Type)
	}
}

// CreateOTLPGRPCMetricExporter 创建 OTLP gRPC 指标导出器
func CreateOTLPGRPCMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	if cfg.Timeout > 0 {
		opts = append(opts, otlpmetricgrpc.WithTimeout(cfg.Timeout))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == compressionGzip {
		opts = append(opts, otlpmetricgrpc.WithCompressor(compressionGzip))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

// CreateOTLPHTTPMetricExporter 创建 OTLP HTTP 指标导出器
func CreateOTLPHTTPMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if cfg.Timeout > 0 {
		opts = append(opts, otlpmetrichttp.WithTimeout(cfg.Timeout))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == compressionGzip {
		opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// NoopSpanExporter 空实现追踪导出器
type NoopSpanExporter struct{}

// ExportSpans 导出 spans（空实现）
func (e *NoopSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

// Shutdown 关闭导出器（空实现）
func (e *NoopSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}

// NoopMetricExporter 空实现指标导出器
type NoopMetricExporter struct{}

// Temporality 返回时间性
func (e *NoopMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// Aggregation 返回聚合方式
func (e *NoopMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return nil
}

// Export 导出指标（空实现）
func (e *NoopMetricExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	return nil
}

// ForceFlush 强制刷新（空实现）
func (e *NoopMetricExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// Shutdown 关闭导出器（空实现）
func (e *NoopMetricExporter) Shutdown(ctx context.Context) error {
	return nil
}
