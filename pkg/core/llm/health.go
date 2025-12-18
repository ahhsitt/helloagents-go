package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// HealthStatus 健康状态
type HealthStatus struct {
	// Provider 提供商名称
	Provider string `json:"provider"`
	// Model 模型名称
	Model string `json:"model"`
	// Healthy 是否健康
	Healthy bool `json:"healthy"`
	// Latency 响应延迟
	Latency time.Duration `json:"latency"`
	// LastCheck 最后检查时间
	LastCheck time.Time `json:"last_check"`
	// Error 错误信息（如果不健康）
	Error string `json:"error,omitempty"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	providers     []Provider
	checkInterval time.Duration
	timeout       time.Duration
	mu            sync.RWMutex
	status        map[string]HealthStatus
	stopCh        chan struct{}
	running       bool
}

// HealthCheckerOption 健康检查器选项
type HealthCheckerOption func(*HealthChecker)

// WithHealthCheckInterval 设置检查间隔
func WithHealthCheckInterval(interval time.Duration) HealthCheckerOption {
	return func(h *HealthChecker) {
		h.checkInterval = interval
	}
}

// WithHealthCheckTimeout 设置检查超时
func WithHealthCheckTimeout(timeout time.Duration) HealthCheckerOption {
	return func(h *HealthChecker) {
		h.timeout = timeout
	}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(providers []Provider, opts ...HealthCheckerOption) *HealthChecker {
	h := &HealthChecker{
		providers:     providers,
		checkInterval: 30 * time.Second,
		timeout:       10 * time.Second,
		status:        make(map[string]HealthStatus),
		stopCh:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Start 启动后台健康检查
func (h *HealthChecker) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	// 立即执行一次检查
	h.CheckAll()

	go func() {
		ticker := time.NewTicker(h.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.CheckAll()
			case <-h.stopCh:
				return
			}
		}
	}()
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		close(h.stopCh)
		h.running = false
	}
}

// CheckAll 检查所有提供商
func (h *HealthChecker) CheckAll() {
	var wg sync.WaitGroup

	for _, provider := range h.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			h.Check(p)
		}(provider)
	}

	wg.Wait()
}

// Check 检查单个提供商
func (h *HealthChecker) Check(provider Provider) HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	start := time.Now()

	// 发送简单的健康检查请求
	req := Request{
		Messages: []message.Message{
			{
				Role:    message.RoleUser,
				Content: "Hi",
			},
		},
		MaxTokens: intPtr(5),
	}

	status := HealthStatus{
		Provider:  provider.Name(),
		Model:     provider.Model(),
		LastCheck: time.Now(),
	}

	_, err := provider.Generate(ctx, req)
	status.Latency = time.Since(start)

	if err != nil {
		status.Healthy = false
		status.Error = err.Error()
		slog.Warn("health check failed",
			"provider", provider.Name(),
			"model", provider.Model(),
			"error", err,
			"latency", status.Latency,
		)
	} else {
		status.Healthy = true
		slog.Debug("health check passed",
			"provider", provider.Name(),
			"model", provider.Model(),
			"latency", status.Latency,
		)
	}

	h.mu.Lock()
	h.status[provider.Name()] = status
	h.mu.Unlock()

	return status
}

// GetStatus 获取提供商状态
func (h *HealthChecker) GetStatus(providerName string) (HealthStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status, ok := h.status[providerName]
	return status, ok
}

// GetAllStatus 获取所有提供商状态
func (h *HealthChecker) GetAllStatus() []HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make([]HealthStatus, 0, len(h.status))
	for _, status := range h.status {
		statuses = append(statuses, status)
	}

	return statuses
}

// IsHealthy 检查提供商是否健康
func (h *HealthChecker) IsHealthy(providerName string) bool {
	status, ok := h.GetStatus(providerName)
	if !ok {
		return true // 未知状态默认健康
	}
	return status.Healthy
}

// GetHealthyProviders 获取健康的提供商
func (h *HealthChecker) GetHealthyProviders() []Provider {
	h.mu.RLock()
	defer h.mu.RUnlock()

	healthy := make([]Provider, 0, len(h.providers))
	for _, provider := range h.providers {
		status, ok := h.status[provider.Name()]
		if !ok || status.Healthy {
			healthy = append(healthy, provider)
		}
	}

	return healthy
}

// String 返回状态摘要
func (h *HealthChecker) String() string {
	statuses := h.GetAllStatus()
	healthy := 0
	for _, s := range statuses {
		if s.Healthy {
			healthy++
		}
	}
	return fmt.Sprintf("HealthChecker: %d/%d providers healthy", healthy, len(statuses))
}

// intPtr 返回 int 指针
func intPtr(i int) *int {
	return &i
}
