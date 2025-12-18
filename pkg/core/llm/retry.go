package llm

import (
	"context"
	"math"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/errors"
)

// RetryFunc 可重试的函数类型
type RetryFunc func() error

// retry 执行带指数退避的重试
func retry(ctx context.Context, maxRetries int, baseDelay time.Duration, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			return errors.ErrContextCanceled
		default:
		}

		// 执行函数
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if !errors.IsRetryable(err) {
			return err
		}

		// 如果不是最后一次重试，等待后继续
		if attempt < maxRetries {
			delay := calculateBackoff(attempt, baseDelay)
			select {
			case <-ctx.Done():
				return errors.ErrContextCanceled
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

// calculateBackoff 计算指数退避时间
// 使用公式: baseDelay * 2^attempt + jitter
// 最大延迟限制为 30 秒
func calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// 指数增长
	exp := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(baseDelay) * exp)

	// 添加 10% 的随机抖动
	jitter := time.Duration(float64(delay) * 0.1)
	delay += jitter

	// 限制最大延迟
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// RetryWithCallback 带回调的重试
type RetryWithCallback struct {
	MaxRetries int
	BaseDelay  time.Duration
	OnRetry    func(attempt int, err error)
}

// Do 执行带回调的重试
func (r *RetryWithCallback) Do(ctx context.Context, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return errors.ErrContextCanceled
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !errors.IsRetryable(err) {
			return err
		}

		if r.OnRetry != nil {
			r.OnRetry(attempt, err)
		}

		if attempt < r.MaxRetries {
			delay := calculateBackoff(attempt, r.BaseDelay)
			select {
			case <-ctx.Done():
				return errors.ErrContextCanceled
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}
