package otel_test

import (
	"context"
	"sync"
	"testing"

	"github.com/easyops/helloagents-go/pkg/otel"
)

func TestNewInMemoryMetrics(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
}

func TestInMemoryMetrics_Counter(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	counter := metrics.Counter("test_counter")

	if counter == nil {
		t.Fatal("expected non-nil counter")
	}

	ctx := context.Background()
	counter.Add(ctx, 5)
	counter.Add(ctx, 3)

	value := metrics.GetCounterValue("test_counter")
	if value != 8 {
		t.Fatalf("expected counter value 8, got %d", value)
	}
}

func TestInMemoryMetrics_CounterWithAttrs(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	counter := metrics.Counter("test_counter")
	ctx := context.Background()

	// Should not panic with attributes
	counter.Add(ctx, 1, otel.NewAttr("key", "value"))

	value := metrics.GetCounterValue("test_counter")
	if value != 1 {
		t.Fatalf("expected counter value 1, got %d", value)
	}
}

func TestInMemoryMetrics_SameCounterReturned(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()

	counter1 := metrics.Counter("same_counter")
	counter2 := metrics.Counter("same_counter")

	ctx := context.Background()
	counter1.Add(ctx, 5)
	counter2.Add(ctx, 3)

	value := metrics.GetCounterValue("same_counter")
	if value != 8 {
		t.Fatalf("expected counter value 8, got %d", value)
	}
}

func TestInMemoryMetrics_Histogram(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	histogram := metrics.Histogram("test_histogram")

	if histogram == nil {
		t.Fatal("expected non-nil histogram")
	}

	ctx := context.Background()
	histogram.Record(ctx, 1.5)
	histogram.Record(ctx, 2.5)
	histogram.Record(ctx, 3.5)

	// Verify we can record without panic
}

func TestInMemoryMetrics_Gauge(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	gauge := metrics.Gauge("test_gauge")

	if gauge == nil {
		t.Fatal("expected non-nil gauge")
	}

	ctx := context.Background()
	gauge.Set(ctx, 42.5)

	value := metrics.GetGaugeValue("test_gauge")
	if value != 42.5 {
		t.Fatalf("expected gauge value 42.5, got %f", value)
	}

	// Update value
	gauge.Set(ctx, 100.0)
	value = metrics.GetGaugeValue("test_gauge")
	if value != 100.0 {
		t.Fatalf("expected gauge value 100.0, got %f", value)
	}
}

func TestInMemoryMetrics_GetNonExistent(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()

	counterValue := metrics.GetCounterValue("non_existent")
	if counterValue != 0 {
		t.Fatalf("expected 0 for non-existent counter, got %d", counterValue)
	}

	gaugeValue := metrics.GetGaugeValue("non_existent")
	if gaugeValue != 0 {
		t.Fatalf("expected 0 for non-existent gauge, got %f", gaugeValue)
	}
}

func TestInMemoryMetrics_ConcurrentAccess(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent counter increments
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter := metrics.Counter("concurrent_counter")
			counter.Add(ctx, 1)
		}()
	}

	wg.Wait()

	value := metrics.GetCounterValue("concurrent_counter")
	if value != 100 {
		t.Fatalf("expected counter value 100, got %d", value)
	}
}

func TestNoopMetrics(t *testing.T) {
	metrics := otel.NewNoopMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}

	ctx := context.Background()

	// All operations should not panic
	counter := metrics.Counter("test")
	counter.Add(ctx, 100)

	histogram := metrics.Histogram("test")
	histogram.Record(ctx, 1.5)

	gauge := metrics.Gauge("test")
	gauge.Set(ctx, 42.0)
}

func TestNewAttr(t *testing.T) {
	attr := otel.NewAttr("key", "value")

	if attr.Key != "key" {
		t.Fatalf("expected key 'key', got %s", attr.Key)
	}
	if attr.Value != "value" {
		t.Fatalf("expected value 'value', got %v", attr.Value)
	}
}

func TestInMemoryMetrics_ImplementsMetrics(t *testing.T) {
	metrics := otel.NewInMemoryMetrics()
	var _ otel.Metrics = metrics
}

func TestNoopMetrics_ImplementsMetrics(t *testing.T) {
	metrics := otel.NewNoopMetrics()
	var _ otel.Metrics = metrics
}
