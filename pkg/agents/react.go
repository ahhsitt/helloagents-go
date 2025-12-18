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
	"github.com/easyops/helloagents-go/pkg/tools"
)

// ReActAgent 实现 ReAct (Reasoning + Acting) 推理模式的 Agent
//
// ReActAgent 通过 Thought-Action-Observation 循环来解决复杂问题:
// 1. Thought: 分析当前状态和下一步行动
// 2. Action: 选择并执行工具
// 3. Observation: 观察工具执行结果
// 4. 重复直到得出最终答案
//
// 使用示例:
//
//	agent, err := agents.NewReAct(llmProvider, registry)
//	agent.AddTool(builtin.NewCalculator())
//	output, err := agent.Run(ctx, agents.Input{Query: "123 乘以 456 等于多少?"})
type ReActAgent struct {
	provider llm.Provider
	options  *AgentOptions
	config   config.AgentConfig

	// 工具系统
	registry *tools.Registry
	executor *tools.Executor

	// 对话历史
	history []message.Message
	mu      sync.RWMutex
}

// NewReAct 创建 ReActAgent 实例
//
// 参数:
//   - provider: LLM 提供商实例
//   - registry: 工具注册表（可为 nil，将创建新的）
//   - opts: 可选配置项
func NewReAct(provider llm.Provider, registry *tools.Registry, opts ...Option) (*ReActAgent, error) {
	if provider == nil {
		return nil, errors.ErrProviderUnavailable
	}

	options := DefaultAgentOptions()
	options.MaxIterations = 10 // ReAct 默认最大迭代次数
	for _, opt := range opts {
		opt(options)
	}

	if registry == nil {
		registry = tools.NewRegistry()
	}

	cfg := config.AgentConfig{
		Name:          options.Name,
		SystemPrompt:  options.SystemPrompt,
		MaxIterations: options.MaxIterations,
		Temperature:   options.Temperature,
		MaxTokens:     options.MaxTokens,
		Timeout:       options.Timeout,
	}

	// 设置默认系统提示词
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = reactSystemPrompt
	}

	return &ReActAgent{
		provider: provider,
		options:  options,
		config:   cfg.WithDefaults(),
		registry: registry,
		executor: tools.NewExecutor(registry),
		history:  make([]message.Message, 0),
	}, nil
}

// reactSystemPrompt ReAct 模式的默认系统提示词
const reactSystemPrompt = `You are a helpful assistant that can use tools to complete tasks.

When you need to use a tool, respond with the tool call. After receiving the tool result, continue reasoning until you can provide the final answer.

Important guidelines:
1. Think step by step before using tools
2. Use tools when necessary to get information or perform actions
3. If a tool returns an error, try to understand why and adjust your approach
4. Provide a clear final answer when you have enough information`

// Name 返回 Agent 名称
func (a *ReActAgent) Name() string {
	return a.config.Name
}

// Config 返回 Agent 配置
func (a *ReActAgent) Config() config.AgentConfig {
	return a.config
}

// AddTool 向 Agent 注册工具
func (a *ReActAgent) AddTool(tool tools.Tool) {
	a.registry.Register(tool)
}

// AddTools 批量注册工具
func (a *ReActAgent) AddTools(toolList ...tools.Tool) {
	for _, tool := range toolList {
		a.registry.Register(tool)
	}
}

// Tools 返回已注册的工具列表
func (a *ReActAgent) Tools() []tools.Tool {
	return a.registry.All()
}

// Run 执行 ReAct 推理循环
func (a *ReActAgent) Run(ctx context.Context, input Input) (Output, error) {
	startTime := time.Now()

	// 应用超时
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	var steps []ReasoningStep
	var totalUsage message.TokenUsage

	// 构建初始消息
	messages := a.buildMessages(input)

	// 获取工具定义
	toolDefs := a.getToolDefinitions()

	// ReAct 循环
	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		// 检查上下文
		select {
		case <-ctx.Done():
			return Output{
				Steps:      steps,
				TokenUsage: totalUsage,
				Duration:   time.Since(startTime),
				Error:      errors.ErrContextCanceled.Error(),
			}, errors.ErrContextCanceled
		default:
		}

		// 构建 LLM 请求
		temp := a.config.Temperature
		maxTokens := a.config.MaxTokens
		req := llm.Request{
			Messages:    messages,
			Tools:       toolDefs,
			ToolChoice:  "auto",
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		}

		// 调用 LLM
		resp, err := a.provider.Generate(ctx, req)
		if err != nil {
			return Output{
				Steps:      steps,
				TokenUsage: totalUsage,
				Duration:   time.Since(startTime),
				Error:      err.Error(),
			}, err
		}

		// 累计 token 使用
		totalUsage.PromptTokens += resp.TokenUsage.PromptTokens
		totalUsage.CompletionTokens += resp.TokenUsage.CompletionTokens
		totalUsage.TotalTokens += resp.TokenUsage.TotalTokens

		// 处理响应
		if len(resp.ToolCalls) == 0 {
			// 没有工具调用，返回最终答案
			a.addToHistory(input.Query, resp.Content)

			return Output{
				Response:   resp.Content,
				Steps:      steps,
				TokenUsage: totalUsage,
				Duration:   time.Since(startTime),
			}, nil
		}

		// 记录思考步骤
		if resp.Content != "" {
			steps = append(steps, NewThoughtStep(resp.Content))
		}

		// 添加助手消息到对话
		assistantMsg := message.Message{
			Role:      message.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// 执行工具调用
		for _, tc := range resp.ToolCalls {
			// 记录行动步骤
			steps = append(steps, NewActionStep(tc.Name, tc.Arguments))

			// 执行工具
			result := a.executor.Execute(ctx, tc.Name, tc.Arguments)

			// 记录观察步骤
			if result.Success {
				steps = append(steps, NewObservationStep(tc.Name, result.Result))
			} else {
				steps = append(steps, NewObservationStep(tc.Name, fmt.Sprintf("Error: %s", result.Error)))
			}

			// 添加工具结果消息
			toolMsg := message.NewToolMessage(tc.ID, tc.Name, result.Result)
			if !result.Success {
				toolMsg.Content = fmt.Sprintf("Error: %s", result.Error)
			}
			messages = append(messages, toolMsg)
		}
	}

	// 超出最大迭代次数
	return Output{
		Steps:      steps,
		TokenUsage: totalUsage,
		Duration:   time.Since(startTime),
		Error:      errors.ErrMaxIterationsExceeded.Error(),
	}, errors.ErrMaxIterationsExceeded
}

// RunStream 以流式方式执行 ReAct
func (a *ReActAgent) RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 对于 ReAct，我们逐步发送推理过程
		// 由于工具调用的复杂性，这里使用非流式方式执行，然后发送步骤

		output, err := a.Run(ctx, input)
		if err != nil {
			errChan <- err
		}

		// 发送推理步骤
		for _, step := range output.Steps {
			chunkChan <- StreamChunk{
				Type: ChunkTypeStep,
				Step: &step,
			}
		}

		// 发送最终响应
		if output.Response != "" {
			chunkChan <- StreamChunk{
				Type:    ChunkTypeText,
				Content: output.Response,
			}
		}

		// 发送完成信号
		chunkChan <- StreamChunk{
			Type: ChunkTypeDone,
			Done: true,
		}
	}()

	return chunkChan, errChan
}

// buildMessages 构建初始消息列表
func (a *ReActAgent) buildMessages(input Input) []message.Message {
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

// getToolDefinitions 获取工具定义
func (a *ReActAgent) getToolDefinitions() []llm.ToolDefinition {
	toolList := a.registry.All()
	defs := make([]llm.ToolDefinition, len(toolList))

	for i, t := range toolList {
		schema := t.Parameters()
		defs[i] = llm.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type":       schema.Type,
				"properties": schema.Properties,
				"required":   schema.Required,
			},
		}
	}

	return defs
}

// addToHistory 将对话添加到历史记录
func (a *ReActAgent) addToHistory(query, response string) {
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
func (a *ReActAgent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = make([]message.Message, 0)
}

// GetHistory 获取对话历史
func (a *ReActAgent) GetHistory() []message.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]message.Message, len(a.history))
	copy(result, a.history)
	return result
}

// compile-time interface check
var _ Agent = (*ReActAgent)(nil)
