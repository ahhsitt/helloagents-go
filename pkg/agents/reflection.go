package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/config"
	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

// ReflectionAgent 实现自我反思推理模式的 Agent
//
// ReflectionAgent 通过 Generate-Reflect-Improve 循环来提升输出质量:
// 1. Generate: 生成初始响应
// 2. Reflect: 自我评估和反思
// 3. Improve: 根据反思改进响应
// 4. 重复直到满意或达到最大迭代次数
//
// 使用示例:
//
//	agent, err := agents.NewReflection(llmProvider)
//	output, err := agent.Run(ctx, agents.Input{Query: "写一个快速排序算法"})
type ReflectionAgent struct {
	provider llm.Provider
	options  *AgentOptions
	config   config.AgentConfig

	// 对话历史
	history []message.Message
	mu      sync.RWMutex
}

// NewReflection 创建 ReflectionAgent 实例
func NewReflection(provider llm.Provider, opts ...Option) (*ReflectionAgent, error) {
	if provider == nil {
		return nil, errors.ErrProviderUnavailable
	}

	options := DefaultAgentOptions()
	options.MaxIterations = 3 // Reflection 默认最大迭代 3 次
	for _, opt := range opts {
		opt(options)
	}

	if options.SystemPrompt == "" {
		options.SystemPrompt = reflectionSystemPrompt
	}

	cfg := config.AgentConfig{
		Name:          options.Name,
		SystemPrompt:  options.SystemPrompt,
		MaxIterations: options.MaxIterations,
		Temperature:   options.Temperature,
		MaxTokens:     options.MaxTokens,
		Timeout:       options.Timeout,
	}

	return &ReflectionAgent{
		provider: provider,
		options:  options,
		config:   cfg.WithDefaults(),
		history:  make([]message.Message, 0),
	}, nil
}

const reflectionSystemPrompt = `You are a thoughtful assistant that produces high-quality responses through self-reflection.

For each request, you will:
1. Generate an initial response
2. Reflect on the quality, accuracy, and completeness of your response
3. Identify areas for improvement
4. Provide an improved response based on your reflection

Be thorough in your reflection and honest about any shortcomings in your initial response.`

const reflectionPromptTemplate = `Please reflect on your previous response and identify:
1. What was done well?
2. What could be improved?
3. Are there any errors or inaccuracies?
4. Is the response complete and comprehensive?

Then provide an improved version of your response that addresses these points.

Previous response:
%s

Provide your reflection and improved response:`

// Name 返回 Agent 名称
func (a *ReflectionAgent) Name() string {
	return a.config.Name
}

// Config 返回 Agent 配置
func (a *ReflectionAgent) Config() config.AgentConfig {
	return a.config
}

// Run 执行 Reflection 推理循环
func (a *ReflectionAgent) Run(ctx context.Context, input Input) (Output, error) {
	startTime := time.Now()

	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	var steps []ReasoningStep
	var totalUsage message.TokenUsage

	// Phase 1: Initial generation
	messages := a.buildMessages(input)
	temp := a.config.Temperature
	maxTokens := a.config.MaxTokens

	req := llm.Request{
		Messages:    messages,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	resp, err := a.provider.Generate(ctx, req)
	if err != nil {
		return Output{
			Steps:      steps,
			TokenUsage: totalUsage,
			Duration:   time.Since(startTime),
			Error:      err.Error(),
		}, err
	}

	totalUsage = addTokenUsage(totalUsage, resp.TokenUsage)
	currentResponse := resp.Content
	steps = append(steps, NewThoughtStep(fmt.Sprintf("Initial response: %s", currentResponse)))

	// Phase 2-3: Reflection and Improvement loop
	for iteration := 1; iteration < a.config.MaxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return Output{
				Response:   currentResponse,
				Steps:      steps,
				TokenUsage: totalUsage,
				Duration:   time.Since(startTime),
				Error:      errors.ErrContextCanceled.Error(),
			}, errors.ErrContextCanceled
		default:
		}

		// Build reflection prompt
		reflectionPrompt := fmt.Sprintf(reflectionPromptTemplate, currentResponse)
		reflectionMessages := append(messages,
			message.Message{Role: message.RoleAssistant, Content: currentResponse},
			message.Message{Role: message.RoleUser, Content: reflectionPrompt},
		)

		req := llm.Request{
			Messages:    reflectionMessages,
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		}

		resp, err := a.provider.Generate(ctx, req)
		if err != nil {
			return Output{
				Response:   currentResponse,
				Steps:      steps,
				TokenUsage: totalUsage,
				Duration:   time.Since(startTime),
				Error:      err.Error(),
			}, err
		}

		totalUsage = addTokenUsage(totalUsage, resp.TokenUsage)

		// Record reflection step
		steps = append(steps, NewReflectionStep(fmt.Sprintf("Iteration %d reflection and improvement", iteration)))
		currentResponse = resp.Content
	}

	a.addToHistory(input.Query, currentResponse)

	return Output{
		Response:   currentResponse,
		Steps:      steps,
		TokenUsage: totalUsage,
		Duration:   time.Since(startTime),
	}, nil
}

// RunStream 流式执行（简化实现，逐步发送结果）
func (a *ReflectionAgent) RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		output, err := a.Run(ctx, input)
		if err != nil {
			errChan <- err
		}

		for _, step := range output.Steps {
			chunkChan <- StreamChunk{Type: ChunkTypeStep, Step: &step}
		}

		if output.Response != "" {
			chunkChan <- StreamChunk{Type: ChunkTypeText, Content: output.Response}
		}

		chunkChan <- StreamChunk{Type: ChunkTypeDone, Done: true}
	}()

	return chunkChan, errChan
}

func (a *ReflectionAgent) buildMessages(input Input) []message.Message {
	messages := make([]message.Message, 0)

	if a.config.SystemPrompt != "" {
		messages = append(messages, message.Message{
			Role:    message.RoleSystem,
			Content: a.config.SystemPrompt,
		})
	}

	a.mu.RLock()
	messages = append(messages, a.history...)
	a.mu.RUnlock()

	messages = append(messages, message.Message{
		Role:    message.RoleUser,
		Content: input.Query,
	})

	return messages
}

func (a *ReflectionAgent) addToHistory(query, response string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.history = append(a.history,
		message.Message{Role: message.RoleUser, Content: query, Timestamp: time.Now()},
		message.Message{Role: message.RoleAssistant, Content: response, Timestamp: time.Now()},
	)
}

// ClearHistory 清除对话历史
func (a *ReflectionAgent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = make([]message.Message, 0)
}

// GetHistory 获取对话历史
func (a *ReflectionAgent) GetHistory() []message.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]message.Message, len(a.history))
	copy(result, a.history)
	return result
}

// addTokenUsage 累加 token 使用量
func addTokenUsage(a, b message.TokenUsage) message.TokenUsage {
	return message.TokenUsage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}

var _ Agent = (*ReflectionAgent)(nil)
