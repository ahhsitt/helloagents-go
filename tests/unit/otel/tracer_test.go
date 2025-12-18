package otel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/easyops/helloagents-go/pkg/otel"
)

func TestNoopTracer(t *testing.T) {
	tracer := otel.NewNoopTracer()
	if tracer == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestNoopTracer_Start(t *testing.T) {
	tracer := otel.NewNoopTracer()
	ctx := context.Background()

	newCtx, span := tracer.Start(ctx, "test-span")

	if newCtx == nil {
		t.Fatal("expected non-nil context")
	}
	if span == nil {
		t.Fatal("expected non-nil span")
	}

	// Should not panic
	span.End()
}

func TestNoopSpan_Methods(t *testing.T) {
	tracer := otel.NewNoopTracer()
	_, span := tracer.Start(context.Background(), "test")

	// All methods should not panic
	span.SetStatus(otel.StatusOK, "ok")
	span.AddEvent("event-name")
	span.RecordError(errors.New("test error"))
	span.End()

	sc := span.SpanContext()
	if sc.TraceID != "" {
		t.Fatal("expected empty trace ID for noop span")
	}
}

func TestNoopTracer_SpanFromContext(t *testing.T) {
	tracer := otel.NewNoopTracer()
	ctx := context.Background()

	span := tracer.SpanFromContext(ctx)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
}

func TestSpanKindConstants(t *testing.T) {
	// Verify span kinds are defined
	kinds := []otel.SpanKind{
		otel.SpanKindInternal,
		otel.SpanKindServer,
		otel.SpanKindClient,
		otel.SpanKindProducer,
		otel.SpanKindConsumer,
	}

	for i, kind := range kinds {
		if int(kind) != i {
			t.Fatalf("expected kind %d to have value %d, got %d", i, i, kind)
		}
	}
}

func TestStatusCodeConstants(t *testing.T) {
	if otel.StatusUnset != 0 {
		t.Fatalf("expected StatusUnset=0, got %d", otel.StatusUnset)
	}
	if otel.StatusOK != 1 {
		t.Fatalf("expected StatusOK=1, got %d", otel.StatusOK)
	}
	if otel.StatusError != 2 {
		t.Fatalf("expected StatusError=2, got %d", otel.StatusError)
	}
}

func TestWithSpanKind(t *testing.T) {
	cfg := &otel.SpanConfig{}
	opt := otel.WithSpanKind(otel.SpanKindClient)
	opt(cfg)

	if cfg.Kind != otel.SpanKindClient {
		t.Fatalf("expected SpanKindClient, got %d", cfg.Kind)
	}
}

func TestNoopTracer_ImplementsTracer(t *testing.T) {
	tracer := otel.NewNoopTracer()
	var _ otel.Tracer = tracer
}
