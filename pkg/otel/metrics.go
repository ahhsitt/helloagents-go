package otel

import (
	"context"
	"sync"
	"time"
)

// Metrics 定义指标接口
type Metrics interface {
	// Counter 返回或创建计数器
	Counter(name string) Counter
	// Histogram 返回或创建直方图
	Histogram(name string) Histogram
	// Gauge 返回或创建仪表
	Gauge(name string) Gauge
}

// Counter 计数器接口
type Counter interface {
	// Add 增加计数
	Add(ctx context.Context, value int64, attrs ...Attr)
}

// Histogram 直方图接口
type Histogram interface {
	// Record 记录值
	Record(ctx context.Context, value float64, attrs ...Attr)
}

// Gauge 仪表接口
type Gauge interface {
	// Set 设置值
	Set(ctx context.Context, value float64, attrs ...Attr)
}

// Attr 指标属性
type Attr struct {
	Key   string
	Value interface{}
}

// NewAttr 创建指标属性
func NewAttr(key string, value interface{}) Attr {
	return Attr{Key: key, Value: value}
}

// InMemoryMetrics 内存指标实现（用于测试和简单场景）
type InMemoryMetrics struct {
	counters   map[string]*InMemoryCounter
	histograms map[string]*InMemoryHistogram
	gauges     map[string]*InMemoryGauge
	mu         sync.RWMutex
}

// NewInMemoryMetrics 创建内存指标
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:   make(map[string]*InMemoryCounter),
		histograms: make(map[string]*InMemoryHistogram),
		gauges:     make(map[string]*InMemoryGauge),
	}
}

// Counter 返回或创建计数器
func (m *InMemoryMetrics) Counter(name string) Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.counters[name]; ok {
		return c
	}

	c := &InMemoryCounter{name: name}
	m.counters[name] = c
	return c
}

// Histogram 返回或创建直方图
func (m *InMemoryMetrics) Histogram(name string) Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.histograms[name]; ok {
		return h
	}

	h := &InMemoryHistogram{name: name}
	m.histograms[name] = h
	return h
}

// Gauge 返回或创建仪表
func (m *InMemoryMetrics) Gauge(name string) Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	if g, ok := m.gauges[name]; ok {
		return g
	}

	g := &InMemoryGauge{name: name}
	m.gauges[name] = g
	return g
}

// GetCounterValue 获取计数器当前值
func (m *InMemoryMetrics) GetCounterValue(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if c, ok := m.counters[name]; ok {
		return c.Value()
	}
	return 0
}

// GetGaugeValue 获取仪表当前值
func (m *InMemoryMetrics) GetGaugeValue(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if g, ok := m.gauges[name]; ok {
		return g.Value()
	}
	return 0
}

// InMemoryCounter 内存计数器
type InMemoryCounter struct {
	name  string
	value int64
	mu    sync.RWMutex
}

// Add 增加计数
func (c *InMemoryCounter) Add(ctx context.Context, value int64, attrs ...Attr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += value
}

// Value 获取当前值
func (c *InMemoryCounter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// InMemoryHistogram 内存直方图
type InMemoryHistogram struct {
	name   string
	values []float64
	mu     sync.RWMutex
}

// Record 记录值
func (h *InMemoryHistogram) Record(ctx context.Context, value float64, attrs ...Attr) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)
}

// Values 获取所有记录的值
func (h *InMemoryHistogram) Values() []float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]float64, len(h.values))
	copy(result, h.values)
	return result
}

// InMemoryGauge 内存仪表
type InMemoryGauge struct {
	name      string
	value     float64
	timestamp time.Time
	mu        sync.RWMutex
}

// Set 设置值
func (g *InMemoryGauge) Set(ctx context.Context, value float64, attrs ...Attr) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
	g.timestamp = time.Now()
}

// Value 获取当前值
func (g *InMemoryGauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// NoopMetrics 空实现指标
type NoopMetrics struct{}

// NewNoopMetrics 创建空实现指标
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

func (m *NoopMetrics) Counter(name string) Counter     { return &NoopCounter{} }
func (m *NoopMetrics) Histogram(name string) Histogram { return &NoopHistogram{} }
func (m *NoopMetrics) Gauge(name string) Gauge         { return &NoopGauge{} }

type NoopCounter struct{}

func (c *NoopCounter) Add(ctx context.Context, value int64, attrs ...Attr) {}

type NoopHistogram struct{}

func (h *NoopHistogram) Record(ctx context.Context, value float64, attrs ...Attr) {}

type NoopGauge struct{}

func (g *NoopGauge) Set(ctx context.Context, value float64, attrs ...Attr) {}

// compile-time interface check
var _ Metrics = (*InMemoryMetrics)(nil)
var _ Metrics = (*NoopMetrics)(nil)
var _ Counter = (*InMemoryCounter)(nil)
var _ Histogram = (*InMemoryHistogram)(nil)
var _ Gauge = (*InMemoryGauge)(nil)
