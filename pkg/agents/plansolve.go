package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/config"
	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// PlanAndSolveAgent 实现计划-执行推理模式的 Agent
//
// PlanAndSolveAgent 通过先制定计划再逐步执行的方式解决复杂问题:
// 1. Plan: 分析任务并制定详细执行计划
// 2. Execute: 按计划逐步执行每个子任务
// 3. Synthesize: 综合所有步骤结果得出最终答案
//
// 使用示例:
//
//	agent, err := agents.NewPlanAndSolve(llmProvider, registry)
//	output, err := agent.Run(ctx, agents.Input{Query: "分析这段代码的性能问题并提出优化方案"})
type PlanAndSolveAgent struct {
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

// NewPlanAndSolve 创建 PlanAndSolveAgent 实例
func NewPlanAndSolve(provider llm.Provider, registry *tools.Registry, opts ...Option) (*PlanAndSolveAgent, error) {
	if provider == nil {
		return nil, errors.ErrProviderUnavailable
	}

	options := DefaultAgentOptions()
	options.MaxIterations = 10
	for _, opt := range opts {
		opt(options)
	}

	if options.SystemPrompt == "" {
		options.SystemPrompt = planAndSolveSystemPrompt
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

	return &PlanAndSolveAgent{
		provider: provider,
		options:  options,
		config:   cfg.WithDefaults(),
		registry: registry,
		executor: tools.NewExecutor(registry),
		history:  make([]message.Message, 0),
	}, nil
}

const planAndSolveSystemPrompt = `You are an intelligent assistant that solves complex problems by first creating a plan and then executing it step by step.

When given a task:
1. First, analyze the task and create a detailed plan with numbered steps
2. Then execute each step sequentially
3. Finally, synthesize all results into a comprehensive answer

Always respond with your plan first, then execute and report results.`

const planPromptTemplate = `Analyze the following task and create a detailed execution plan.

Task: %s

Respond with a JSON object containing:
{
  "analysis": "Brief analysis of the task",
  "steps": [
    {"id": 1, "description": "Step description", "requires_tool": false, "tool_name": ""},
    ...
  ],
  "expected_outcome": "What the final result should look like"
}

Create a plan with clear, actionable steps.`

// Plan 表示执行计划
type Plan struct {
	Analysis        string     `json:"analysis"`
	Steps           []PlanStep `json:"steps"`
	ExpectedOutcome string     `json:"expected_outcome"`
}

// PlanStep 计划中的单个步骤
type PlanStep struct {
	ID           int    `json:"id"`
	Description  string `json:"description"`
	RequiresTool bool   `json:"requires_tool"`
	ToolName     string `json:"tool_name,omitempty"`
}

// Name 返回 Agent 名称
func (a *PlanAndSolveAgent) Name() string {
	return a.config.Name
}

// Config 返回 Agent 配置
func (a *PlanAndSolveAgent) Config() config.AgentConfig {
	return a.config
}

// AddTool 向 Agent 注册工具
func (a *PlanAndSolveAgent) AddTool(tool tools.Tool) {
	a.registry.Register(tool)
}

// AddTools 批量注册工具
func (a *PlanAndSolveAgent) AddTools(toolList ...tools.Tool) {
	for _, tool := range toolList {
		a.registry.Register(tool)
	}
}

// Tools 返回已注册的工具列表
func (a *PlanAndSolveAgent) Tools() []tools.Tool {
	return a.registry.All()
}

// Run 执行 PlanAndSolve 推理
func (a *PlanAndSolveAgent) Run(ctx context.Context, input Input) (Output, error) {
	startTime := time.Now()

	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	var steps []ReasoningStep
	var totalUsage message.TokenUsage

	// Phase 1: Generate Plan
	plan, planUsage, err := a.generatePlan(ctx, input)
	if err != nil {
		return Output{
			Steps:      steps,
			TokenUsage: totalUsage,
			Duration:   time.Since(startTime),
			Error:      err.Error(),
		}, err
	}
	totalUsage = addTokenUsage(totalUsage, planUsage)

	// Record plan step
	planContent := fmt.Sprintf("Plan created with %d steps:\n%s", len(plan.Steps), plan.Analysis)
	steps = append(steps, NewPlanStep(planContent))

	// Phase 2: Execute each step
	stepResults := make([]string, 0, len(plan.Steps))

	for i, planStep := range plan.Steps {
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

		if i >= a.config.MaxIterations {
			break
		}

		result, stepUsage, err := a.executeStep(ctx, input, planStep, stepResults)
		if err != nil {
			steps = append(steps, NewObservationStep(fmt.Sprintf("step_%d", planStep.ID),
				fmt.Sprintf("Error: %v", err)))
			continue
		}

		totalUsage = addTokenUsage(totalUsage, stepUsage)
		stepResults = append(stepResults, result)

		// Record execution step
		steps = append(steps, NewActionStep(fmt.Sprintf("step_%d", planStep.ID),
			map[string]interface{}{"description": planStep.Description}))
		steps = append(steps, NewObservationStep(fmt.Sprintf("step_%d", planStep.ID), result))
	}

	// Phase 3: Synthesize results
	finalResponse, synthUsage, err := a.synthesize(ctx, input, plan, stepResults)
	if err != nil {
		return Output{
			Response:   strings.Join(stepResults, "\n\n"),
			Steps:      steps,
			TokenUsage: totalUsage,
			Duration:   time.Since(startTime),
			Error:      err.Error(),
		}, err
	}
	totalUsage = addTokenUsage(totalUsage, synthUsage)

	a.addToHistory(input.Query, finalResponse)

	return Output{
		Response:   finalResponse,
		Steps:      steps,
		TokenUsage: totalUsage,
		Duration:   time.Since(startTime),
	}, nil
}

// generatePlan 生成执行计划
func (a *PlanAndSolveAgent) generatePlan(ctx context.Context, input Input) (*Plan, message.TokenUsage, error) {
	planPrompt := fmt.Sprintf(planPromptTemplate, input.Query)

	messages := []message.Message{
		{Role: message.RoleSystem, Content: a.config.SystemPrompt},
		{Role: message.RoleUser, Content: planPrompt},
	}

	temp := a.config.Temperature
	maxTokens := a.config.MaxTokens
	req := llm.Request{
		Messages:    messages,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	resp, err := a.provider.Generate(ctx, req)
	if err != nil {
		return nil, message.TokenUsage{}, err
	}

	// Parse plan from response
	plan, err := parsePlan(resp.Content)
	if err != nil {
		// If JSON parsing fails, create a simple plan
		plan = &Plan{
			Analysis: "Direct execution",
			Steps: []PlanStep{
				{ID: 1, Description: input.Query, RequiresTool: false},
			},
			ExpectedOutcome: "Complete response to the query",
		}
	}

	return plan, resp.TokenUsage, nil
}

// parsePlan 从 LLM 响应中解析计划
func parsePlan(content string) (*Plan, error) {
	// Try to extract JSON from response
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := content[start : end+1]
	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

// executeStep 执行计划中的单个步骤
func (a *PlanAndSolveAgent) executeStep(ctx context.Context, input Input, step PlanStep, previousResults []string) (string, message.TokenUsage, error) {
	// Build context with previous results
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Original task: %s\n\n", input.Query))

	if len(previousResults) > 0 {
		contextBuilder.WriteString("Previous results:\n")
		for i, result := range previousResults {
			contextBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result))
		}
		contextBuilder.WriteString("\n")
	}

	contextBuilder.WriteString(fmt.Sprintf("Current step: %s\n", step.Description))
	contextBuilder.WriteString("Execute this step and provide the result:")

	messages := []message.Message{
		{Role: message.RoleSystem, Content: a.config.SystemPrompt},
		{Role: message.RoleUser, Content: contextBuilder.String()},
	}

	// If step requires a tool, use tool calling
	if step.RequiresTool && step.ToolName != "" && a.registry.Has(step.ToolName) {
		toolDefs := a.getToolDefinitions()
		temp := a.config.Temperature
		maxTokens := a.config.MaxTokens

		req := llm.Request{
			Messages:    messages,
			Tools:       toolDefs,
			ToolChoice:  "auto",
			Temperature: &temp,
			MaxTokens:   &maxTokens,
		}

		resp, err := a.provider.Generate(ctx, req)
		if err != nil {
			return "", message.TokenUsage{}, err
		}

		// Execute any tool calls
		if len(resp.ToolCalls) > 0 {
			var results []string
			for _, tc := range resp.ToolCalls {
				result := a.executor.Execute(ctx, tc.Name, tc.Arguments)
				if result.Success {
					results = append(results, result.Result)
				} else {
					results = append(results, fmt.Sprintf("Error: %s", result.Error))
				}
			}
			return strings.Join(results, "\n"), resp.TokenUsage, nil
		}

		return resp.Content, resp.TokenUsage, nil
	}

	// Regular execution without tools
	temp := a.config.Temperature
	maxTokens := a.config.MaxTokens
	req := llm.Request{
		Messages:    messages,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	resp, err := a.provider.Generate(ctx, req)
	if err != nil {
		return "", message.TokenUsage{}, err
	}

	return resp.Content, resp.TokenUsage, nil
}

// synthesize 综合所有步骤结果
func (a *PlanAndSolveAgent) synthesize(ctx context.Context, input Input, plan *Plan, results []string) (string, message.TokenUsage, error) {
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Original task: %s\n\n", input.Query))
	contextBuilder.WriteString(fmt.Sprintf("Expected outcome: %s\n\n", plan.ExpectedOutcome))
	contextBuilder.WriteString("Step results:\n")

	for i, result := range results {
		stepDesc := ""
		if i < len(plan.Steps) {
			stepDesc = plan.Steps[i].Description
		}
		contextBuilder.WriteString(fmt.Sprintf("Step %d (%s):\n%s\n\n", i+1, stepDesc, result))
	}

	contextBuilder.WriteString("Based on all the step results above, provide a comprehensive final answer that addresses the original task.")

	messages := []message.Message{
		{Role: message.RoleSystem, Content: a.config.SystemPrompt},
		{Role: message.RoleUser, Content: contextBuilder.String()},
	}

	temp := a.config.Temperature
	maxTokens := a.config.MaxTokens
	req := llm.Request{
		Messages:    messages,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	resp, err := a.provider.Generate(ctx, req)
	if err != nil {
		return "", message.TokenUsage{}, err
	}

	return resp.Content, resp.TokenUsage, nil
}

func (a *PlanAndSolveAgent) getToolDefinitions() []llm.ToolDefinition {
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

// RunStream 流式执行
func (a *PlanAndSolveAgent) RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error) {
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

func (a *PlanAndSolveAgent) addToHistory(query, response string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.history = append(a.history,
		message.Message{Role: message.RoleUser, Content: query, Timestamp: time.Now()},
		message.Message{Role: message.RoleAssistant, Content: response, Timestamp: time.Now()},
	)
}

// ClearHistory 清除对话历史
func (a *PlanAndSolveAgent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = make([]message.Message, 0)
}

// GetHistory 获取对话历史
func (a *PlanAndSolveAgent) GetHistory() []message.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]message.Message, len(a.history))
	copy(result, a.history)
	return result
}

var _ Agent = (*PlanAndSolveAgent)(nil)
