package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// FallbackProvider 带备用降级的提供商
type FallbackProvider struct {
	primary   Provider
	fallbacks []Provider
	mu        sync.RWMutex
	// 健康状态跟踪
	healthStatus map[Provider]bool
	lastCheck    map[Provider]time.Time
	checkInterval time.Duration
}

// FallbackOption 备用提供商选项
type FallbackOption func(*FallbackProvider)

// WithFallbackCheckInterval 设置健康检查间隔
func WithFallbackCheckInterval(interval time.Duration) FallbackOption {
	return func(f *FallbackProvider) {
		f.checkInterval = interval
	}
}

// NewFallbackProvider 创建带备用的提供商
func NewFallbackProvider(primary Provider, fallbacks []Provider, opts ...FallbackOption) *FallbackProvider {
	f := &FallbackProvider{
		primary:       primary,
		fallbacks:     fallbacks,
		healthStatus:  make(map[Provider]bool),
		lastCheck:     make(map[Provider]time.Time),
		checkInterval: 30 * time.Second,
	}

	// 初始化健康状态
	f.healthStatus[primary] = true
	for _, fb := range fallbacks {
		f.healthStatus[fb] = true
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Generate 生成响应（非流式）
func (f *FallbackProvider) Generate(ctx context.Context, req Request) (Response, error) {
	providers := f.getAvailableProviders()

	var lastErr error
	for _, provider := range providers {
		resp, err := provider.Generate(ctx, req)
		if err == nil {
			f.markHealthy(provider)
			return resp, nil
		}

		lastErr = err
		f.markUnhealthy(provider)
		slog.Warn("provider failed, trying fallback",
			"provider", provider.Name(),
			"error", err,
		)
	}

	return Response{}, fmt.Errorf("all providers failed, last error: %w", lastErr)
}

// GenerateStream 生成响应（流式）
func (f *FallbackProvider) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	providers := f.getAvailableProviders()

	if len(providers) == 0 {
		errCh := make(chan error, 1)
		chunkCh := make(chan StreamChunk)
		close(chunkCh)
		errCh <- fmt.Errorf("no available providers")
		close(errCh)
		return chunkCh, errCh
	}

	// 尝试第一个可用的提供商
	for i, provider := range providers {
		chunkCh, errCh := provider.GenerateStream(ctx, req)

		// 对于流式请求，我们需要监控第一个块来判断是否成功
		// 这里简化处理，直接返回第一个提供商的结果
		if i == 0 {
			return f.wrapStreamWithFallback(ctx, req, provider, providers[1:], chunkCh, errCh)
		}
	}

	// 使用主提供商
	return f.primary.GenerateStream(ctx, req)
}

// wrapStreamWithFallback 包装流式响应，支持错误时降级
func (f *FallbackProvider) wrapStreamWithFallback(
	ctx context.Context,
	req Request,
	currentProvider Provider,
	remainingProviders []Provider,
	chunkCh <-chan StreamChunk,
	errCh <-chan error,
) (<-chan StreamChunk, <-chan error) {
	outChunkCh := make(chan StreamChunk)
	outErrCh := make(chan error, 1)

	go func() {
		defer close(outChunkCh)
		defer close(outErrCh)

		receivedAny := false

		for {
			select {
			case chunk, ok := <-chunkCh:
				if !ok {
					// 正常结束
					if receivedAny {
						f.markHealthy(currentProvider)
					}
					return
				}
				receivedAny = true
				select {
				case outChunkCh <- chunk:
				case <-ctx.Done():
					outErrCh <- ctx.Err()
					return
				}

			case err, ok := <-errCh:
				if !ok {
					continue
				}
				if err != nil && !receivedAny {
					// 尝试降级到下一个提供商
					f.markUnhealthy(currentProvider)
					slog.Warn("stream provider failed, trying fallback",
						"provider", currentProvider.Name(),
						"error", err,
					)

					for _, fallback := range remainingProviders {
						fbChunkCh, fbErrCh := fallback.GenerateStream(ctx, req)
						// 传递降级后的流
						for chunk := range fbChunkCh {
							select {
							case outChunkCh <- chunk:
							case <-ctx.Done():
								outErrCh <- ctx.Err()
								return
							}
						}
						if fbErr := <-fbErrCh; fbErr != nil {
							f.markUnhealthy(fallback)
							continue
						}
						f.markHealthy(fallback)
						return
					}
					outErrCh <- fmt.Errorf("all providers failed, last error: %w", err)
					return
				}
				if err != nil {
					outErrCh <- err
				}
				return

			case <-ctx.Done():
				outErrCh <- ctx.Err()
				return
			}
		}
	}()

	return outChunkCh, outErrCh
}

// Embed 生成文本嵌入向量
func (f *FallbackProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	providers := f.getAvailableProviders()

	var lastErr error
	for _, provider := range providers {
		embeddings, err := provider.Embed(ctx, texts)
		if err == nil {
			f.markHealthy(provider)
			return embeddings, nil
		}

		lastErr = err
		f.markUnhealthy(provider)
		slog.Warn("embed provider failed, trying fallback",
			"provider", provider.Name(),
			"error", err,
		)
	}

	return nil, fmt.Errorf("all providers failed for embedding, last error: %w", lastErr)
}

// Name 返回提供商名称
func (f *FallbackProvider) Name() string {
	return fmt.Sprintf("fallback(%s)", f.primary.Name())
}

// Model 返回当前模型名称
func (f *FallbackProvider) Model() string {
	return f.primary.Model()
}

// Close 关闭所有客户端连接
func (f *FallbackProvider) Close() error {
	var firstErr error

	if err := f.primary.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	for _, fb := range f.fallbacks {
		if err := fb.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// getAvailableProviders 获取可用的提供商列表
func (f *FallbackProvider) getAvailableProviders() []Provider {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]Provider, 0, 1+len(f.fallbacks))

	// 优先添加健康的提供商
	if f.isHealthy(f.primary) {
		providers = append(providers, f.primary)
	}

	for _, fb := range f.fallbacks {
		if f.isHealthy(fb) {
			providers = append(providers, fb)
		}
	}

	// 如果所有提供商都不健康，仍然尝试所有
	if len(providers) == 0 {
		providers = append(providers, f.primary)
		providers = append(providers, f.fallbacks...)
	}

	return providers
}

// isHealthy 检查提供商是否健康
func (f *FallbackProvider) isHealthy(provider Provider) bool {
	healthy, ok := f.healthStatus[provider]
	if !ok {
		return true
	}

	// 如果不健康，检查是否应该重试
	if !healthy {
		lastCheck, ok := f.lastCheck[provider]
		if ok && time.Since(lastCheck) > f.checkInterval {
			return true // 允许重试
		}
	}

	return healthy
}

// markHealthy 标记提供商为健康
func (f *FallbackProvider) markHealthy(provider Provider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.healthStatus[provider] = true
	f.lastCheck[provider] = time.Now()
}

// markUnhealthy 标记提供商为不健康
func (f *FallbackProvider) markUnhealthy(provider Provider) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.healthStatus[provider] = false
	f.lastCheck[provider] = time.Now()
}

// compile-time interface check
var _ Provider = (*FallbackProvider)(nil)
