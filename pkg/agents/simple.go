// Package agents 提供 Agent 的接口定义和实现
package agents

import (
	"context"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/config"
	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

// SimpleAgent 简单对话代理
//
// SimpleAgent 提供最基础的对话能力，适用于:
// - 简单问答场景
// - 无需工具调用的对话
// - 单轮或多轮对话
//
// 使用示例:
//
//	agent, err := agents.NewSimple(llmProvider, agents.WithName("MyAgent"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	output, err := agent.Run(ctx, agents.Input{Query: "你好"})
type SimpleAgent struct {
	provider llm.Provider
	options  *AgentOptions
	config   config.AgentConfig

	// 多轮对话支持
	history []message.Message
	mu      sync.RWMutex
}

// NewSimple 创建 SimpleAgent 实例
//
// 参数:
//   - provider: LLM 提供商实例
//   - opts: 可选配置项
//
// 返回:
//   - *SimpleAgent: Agent 实例
//   - error: 如果 provider 为 nil 返回错误
func NewSimple(provider llm.Provider, opts ...Option) (*SimpleAgent, error) {
	if provider == nil {
		return nil, errors.ErrProviderUnavailable
	}

	options := DefaultAgentOptions()
	for _, opt := range opts {
		opt(options)
	}

	cfg := config.AgentConfig{
		Name:          options.Name,
		SystemPrompt:  options.SystemPrompt,
		MaxIterations: options.MaxIterations,
		Temperature:   options.Temperature,
		MaxTokens:     options.MaxTokens,
		Timeout:       options.Timeout,
	}

	return &SimpleAgent{
		provider: provider,
		options:  options,
		config:   cfg.WithDefaults(),
		history:  make([]message.Message, 0),
	}, nil
}

// Name 返回 Agent 名称
func (a *SimpleAgent) Name() string {
	return a.config.Name
}

// Config 返回 Agent 配置（只读）
func (a *SimpleAgent) Config() config.AgentConfig {
	return a.config
}

// Run 执行 Agent 的主要逻辑
//
// 参数:
//   - ctx: 上下文，用于取消、超时控制
//   - input: Agent 输入，包含用户查询
//
// 返回:
//   - Output: 包含响应和 token 使用量
//   - error: 执行错误
func (a *SimpleAgent) Run(ctx context.Context, input Input) (Output, error) {
	startTime := time.Now()

	// 应用超时
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	// 构建消息列表
	messages := a.buildMessages(input)

	// 构建 LLM 请求
	temp := a.config.Temperature
	maxTokens := a.config.MaxTokens
	req := llm.Request{
		Messages:    messages,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	// 调用 LLM
	resp, err := a.provider.Generate(ctx, req)
	if err != nil {
		return Output{
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}, err
	}

	// 保存对话历史
	a.addToHistory(input.Query, resp.Content)

	return Output{
		Response:   resp.Content,
		TokenUsage: resp.TokenUsage,
		Duration:   time.Since(startTime),
	}, nil
}

// RunStream 以流式方式执行 Agent
//
// 返回两个 channel：
//   - <-chan StreamChunk: 流式输出块
//   - <-chan error: 错误通道（最多一个错误）
func (a *SimpleAgent) RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 应用超时
		if a.config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
			defer cancel()
		}

		// 构建消息列表
		messages := a.buildMessages(input)

		// 构建 LLM 请求
		temp := a.config.Temperature
		maxTokens := a.config.MaxTokens
		req := llm.Request{
			Messages:    messages,
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		}

		// 调用 LLM 流式接口
		llmChunks, llmErrs := a.provider.GenerateStream(ctx, req)

		var fullContent string

		// 转发 LLM 流式响应
		for {
			select {
			case <-ctx.Done():
				errChan <- errors.ErrContextCanceled
				return
			case err, ok := <-llmErrs:
				if ok && err != nil {
					errChan <- err
					return
				}
			case chunk, ok := <-llmChunks:
				if !ok {
					// 流结束
					return
				}

				// 累积内容
				if chunk.Content != "" {
					fullContent += chunk.Content
					chunkChan <- StreamChunk{
						Type:    ChunkTypeText,
						Content: chunk.Content,
					}
				}

				if chunk.Done {
					// 保存对话历史
					a.addToHistory(input.Query, fullContent)

					// 发送完成信号
					chunkChan <- StreamChunk{
						Type: ChunkTypeDone,
						Done: true,
					}
					return
				}
			}
		}
	}()

	return chunkChan, errChan
}

// buildMessages 构建发送给 LLM 的消息列表
func (a *SimpleAgent) buildMessages(input Input) []message.Message {
	messages := make([]message.Message, 0)

	// 添加系统提示词
	if a.config.SystemPrompt != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleSystem,
			Content: a.config.SystemPrompt,
		})
	}

	// 添加历史消息
	a.mu.RLock()
	messages = append(messages, a.history...)
	a.mu.RUnlock()

	// 添加当前用户消息
	messages = append(messages, message.Message{
		Role:    message.RoleUser,
		Content: input.Query,
	})

	return messages
}

// addToHistory 将一轮对话添加到历史记录
func (a *SimpleAgent) addToHistory(query, response string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.history = append(a.history,
		message.Message{
			Role:      message.RoleUser,
			Content:   query,
			Timestamp: time.Now(),
		},
		message.Message{
			Role:      message.RoleAssistant,
			Content:   response,
			Timestamp: time.Now(),
		},
	)
}

// ClearHistory 清除对话历史
func (a *SimpleAgent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = make([]message.Message, 0)
}

// GetHistory 获取对话历史
func (a *SimpleAgent) GetHistory() []message.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]message.Message, len(a.history))
	copy(result, a.history)
	return result
}

// SetSystemPrompt 动态设置系统提示词
func (a *SimpleAgent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config.SystemPrompt = prompt
}

// compile-time interface check
var _ Agent = (*SimpleAgent)(nil)
