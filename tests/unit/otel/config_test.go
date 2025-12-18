package otel_test

import (
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/otel"
)

func TestDefaultConfig(t *testing.T) {
	cfg := otel.DefaultConfig()

	if cfg.Enabled {
		t.Fatal("expected Enabled to be false by default")
	}
	if cfg.ServiceName != "helloagents" {
		t.Fatalf("expected ServiceName 'helloagents', got %s", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "0.1.0" {
		t.Fatalf("expected ServiceVersion '0.1.0', got %s", cfg.ServiceVersion)
	}
	if cfg.Environment != "development" {
		t.Fatalf("expected Environment 'development', got %s", cfg.Environment)
	}
}

func TestDefaultConfig_Tracing(t *testing.T) {
	cfg := otel.DefaultConfig()

	if cfg.Tracing.Enabled {
		t.Fatal("expected Tracing.Enabled to be false by default")
	}
	if cfg.Tracing.Endpoint != "localhost:4317" {
		t.Fatalf("expected Tracing.Endpoint 'localhost:4317', got %s", cfg.Tracing.Endpoint)
	}
	if !cfg.Tracing.Insecure {
		t.Fatal("expected Tracing.Insecure to be true by default")
	}
	if cfg.Tracing.SampleRate != 1.0 {
		t.Fatalf("expected Tracing.SampleRate 1.0, got %f", cfg.Tracing.SampleRate)
	}
	if cfg.Tracing.Timeout != 30*time.Second {
		t.Fatalf("expected Tracing.Timeout 30s, got %s", cfg.Tracing.Timeout)
	}
}

func TestDefaultConfig_Metrics(t *testing.T) {
	cfg := otel.DefaultConfig()

	if cfg.Metrics.Enabled {
		t.Fatal("expected Metrics.Enabled to be false by default")
	}
	if cfg.Metrics.Endpoint != "localhost:4317" {
		t.Fatalf("expected Metrics.Endpoint 'localhost:4317', got %s", cfg.Metrics.Endpoint)
	}
	if !cfg.Metrics.Insecure {
		t.Fatal("expected Metrics.Insecure to be true by default")
	}
	if cfg.Metrics.Interval != 60*time.Second {
		t.Fatalf("expected Metrics.Interval 60s, got %s", cfg.Metrics.Interval)
	}
}

func TestDefaultConfig_Logging(t *testing.T) {
	cfg := otel.DefaultConfig()

	if cfg.Logging.Level != "info" {
		t.Fatalf("expected Logging.Level 'info', got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Fatalf("expected Logging.Format 'text', got %s", cfg.Logging.Format)
	}
	if !cfg.Logging.IncludeTraceID {
		t.Fatal("expected Logging.IncludeTraceID to be true by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    otel.Config
		expectErr bool
	}{
		{
			name:      "valid config",
			config:    otel.DefaultConfig(),
			expectErr: false,
		},
		{
			name: "invalid sample rate - negative",
			config: otel.Config{
				Tracing: otel.TracingConfig{
					SampleRate: -0.1,
				},
			},
			expectErr: true,
		},
		{
			name: "invalid sample rate - too high",
			config: otel.Config{
				Tracing: otel.TracingConfig{
					SampleRate: 1.5,
				},
			},
			expectErr: true,
		},
		{
			name: "valid sample rate - zero",
			config: otel.Config{
				Tracing: otel.TracingConfig{
					SampleRate: 0.0,
				},
			},
			expectErr: false,
		},
		{
			name: "valid sample rate - one",
			config: otel.Config{
				Tracing: otel.TracingConfig{
					SampleRate: 1.0,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	emptyConfig := otel.Config{}
	cfg := emptyConfig.WithDefaults()

	if cfg.ServiceName != "helloagents" {
		t.Fatalf("expected ServiceName 'helloagents', got %s", cfg.ServiceName)
	}
	if cfg.ServiceVersion != "0.1.0" {
		t.Fatalf("expected ServiceVersion '0.1.0', got %s", cfg.ServiceVersion)
	}
	if cfg.Environment != "development" {
		t.Fatalf("expected Environment 'development', got %s", cfg.Environment)
	}
	if cfg.Tracing.Endpoint != "localhost:4317" {
		t.Fatalf("expected Tracing.Endpoint 'localhost:4317', got %s", cfg.Tracing.Endpoint)
	}
}

func TestConfig_WithDefaults_PreservesSetValues(t *testing.T) {
	cfg := otel.Config{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		Tracing: otel.TracingConfig{
			Endpoint: "custom:4317",
		},
	}

	result := cfg.WithDefaults()

	if result.ServiceName != "my-service" {
		t.Fatalf("expected ServiceName 'my-service', got %s", result.ServiceName)
	}
	if result.ServiceVersion != "1.0.0" {
		t.Fatalf("expected ServiceVersion '1.0.0', got %s", result.ServiceVersion)
	}
	if result.Tracing.Endpoint != "custom:4317" {
		t.Fatalf("expected Tracing.Endpoint 'custom:4317', got %s", result.Tracing.Endpoint)
	}
	// Should fill in unset environment
	if result.Environment != "development" {
		t.Fatalf("expected Environment 'development', got %s", result.Environment)
	}
}
