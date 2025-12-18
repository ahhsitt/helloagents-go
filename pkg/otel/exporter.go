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
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

// createTraceExporter 创建追踪导出器
func createTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Type {
	case ExporterOTLPGRPC:
		return createOTLPGRPCTraceExporter(ctx, cfg)
	case ExporterOTLPHTTP:
		return createOTLPHTTPTraceExporter(ctx, cfg)
	case ExporterStdout:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case ExporterNone:
		return &noopSpanExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported trace exporter type: %s", cfg.Type)
	}
}

// createOTLPGRPCTraceExporter 创建 OTLP gRPC 追踪导出器
func createOTLPGRPCTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
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

	if cfg.Compression == "gzip" {
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
	}

	client := otlptracegrpc.NewClient(opts...)
	return otlptrace.New(ctx, client)
}

// createOTLPHTTPTraceExporter 创建 OTLP HTTP 追踪导出器
func createOTLPHTTPTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
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

	if cfg.Compression == "gzip" {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}

	return otlptracehttp.New(ctx, opts...)
}

// createMetricExporter 创建指标导出器
func createMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	switch cfg.Type {
	case ExporterOTLPGRPC:
		return createOTLPGRPCMetricExporter(ctx, cfg)
	case ExporterOTLPHTTP:
		return createOTLPHTTPMetricExporter(ctx, cfg)
	case ExporterStdout:
		return stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	case ExporterNone:
		return &noopMetricExporter{}, nil
	default:
		return nil, fmt.Errorf("unsupported metric exporter type: %s", cfg.Type)
	}
}

// createOTLPGRPCMetricExporter 创建 OTLP gRPC 指标导出器
func createOTLPGRPCMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
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

	if cfg.Compression == "gzip" {
		opts = append(opts, otlpmetricgrpc.WithCompressor("gzip"))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

// createOTLPHTTPMetricExporter 创建 OTLP HTTP 指标导出器
func createOTLPHTTPMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
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

	if cfg.Compression == "gzip" {
		opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// noopSpanExporter 空实现追踪导出器
type noopSpanExporter struct{}

func (e *noopSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}

// noopMetricExporter 空实现指标导出器
type noopMetricExporter struct{}

func (e *noopMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *noopMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return nil
}

func (e *noopMetricExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	return nil
}

func (e *noopMetricExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func (e *noopMetricExporter) Shutdown(ctx context.Context) error {
	return nil
}
