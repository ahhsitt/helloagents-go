package tools

import (
	"context"
	"fmt"
	"strings"
)

// Chain 工具链接口
//
// 工具链允许将多个工具组合成一个执行序列。
type Chain interface {
	// Execute 执行工具链
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
	// Tools 返回链中的所有工具
	Tools() []Tool
}

// SequentialChain 顺序工具链
//
// 按顺序执行多个工具，前一个工具的输出可以作为下一个工具的输入。
type SequentialChain struct {
	name        string
	description string
	tools       []Tool
	// inputMapping 定义每个工具的输入映射
	// key: 工具索引, value: 参数名 -> 来源（"input.xxx" 或 "output.N.xxx"）
	inputMapping []map[string]string
}

// SequentialChainOption 顺序工具链配置
type SequentialChainOption func(*SequentialChain)

// NewSequentialChain 创建顺序工具链
func NewSequentialChain(name, description string, tools []Tool, opts ...SequentialChainOption) *SequentialChain {
	c := &SequentialChain{
		name:        name,
		description: description,
		tools:       tools,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithInputMapping 设置输入映射
func WithInputMapping(mapping []map[string]string) SequentialChainOption {
	return func(c *SequentialChain) {
		c.inputMapping = mapping
	}
}

// Execute 执行顺序工具链
func (c *SequentialChain) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	outputs := make([]map[string]interface{}, len(c.tools))
	var results []string

	for i, tool := range c.tools {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// 构建工具输入
		toolInput := c.buildToolInput(i, input, outputs)

		// 执行工具
		result, err := tool.Execute(ctx, toolInput)
		if err != nil {
			return "", fmt.Errorf("tool %s failed: %w", tool.Name(), err)
		}

		// 保存输出
		outputs[i] = map[string]interface{}{
			"result": result,
		}
		results = append(results, result)
	}

	// 返回最后一个工具的输出
	if len(results) > 0 {
		return results[len(results)-1], nil
	}
	return "", nil
}

// buildToolInput 构建工具输入
func (c *SequentialChain) buildToolInput(toolIndex int, input map[string]interface{}, outputs []map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 如果没有映射，使用原始输入
	if toolIndex >= len(c.inputMapping) || c.inputMapping[toolIndex] == nil {
		// 默认：第一个工具使用原始输入，后续工具使用前一个工具的输出
		if toolIndex == 0 {
			return input
		}
		// 将前一个工具的 result 作为 input 参数
		if toolIndex > 0 && outputs[toolIndex-1] != nil {
			return map[string]interface{}{
				"input": outputs[toolIndex-1]["result"],
			}
		}
		return input
	}

	// 根据映射构建输入
	for paramName, source := range c.inputMapping[toolIndex] {
		value := resolveSource(source, input, outputs)
		if value != nil {
			result[paramName] = value
		}
	}

	return result
}

// resolveSource 解析值来源
func resolveSource(source string, input map[string]interface{}, outputs []map[string]interface{}) interface{} {
	parts := strings.SplitN(source, ".", 2)
	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "input":
		return input[parts[1]]
	case "output":
		// 格式: output.N.field
		subParts := strings.SplitN(parts[1], ".", 2)
		if len(subParts) < 2 {
			return nil
		}
		var idx int
		fmt.Sscanf(subParts[0], "%d", &idx)
		if idx >= 0 && idx < len(outputs) && outputs[idx] != nil {
			return outputs[idx][subParts[1]]
		}
	}

	return nil
}

// Tools 返回链中的所有工具
func (c *SequentialChain) Tools() []Tool {
	return c.tools
}

// Name 返回链名称
func (c *SequentialChain) Name() string {
	return c.name
}

// Description 返回链描述
func (c *SequentialChain) Description() string {
	return c.description
}

// ConditionalChain 条件工具链
//
// 根据条件选择执行哪个工具。
type ConditionalChain struct {
	name        string
	description string
	condition   ConditionFunc
	trueTool    Tool
	falseTool   Tool
}

// ConditionFunc 条件判断函数
type ConditionFunc func(input map[string]interface{}) bool

// NewConditionalChain 创建条件工具链
func NewConditionalChain(name, description string, condition ConditionFunc, trueTool, falseTool Tool) *ConditionalChain {
	return &ConditionalChain{
		name:        name,
		description: description,
		condition:   condition,
		trueTool:    trueTool,
		falseTool:   falseTool,
	}
}

// Execute 执行条件工具链
func (c *ConditionalChain) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	if c.condition(input) {
		if c.trueTool != nil {
			return c.trueTool.Execute(ctx, input)
		}
		return "", nil
	}

	if c.falseTool != nil {
		return c.falseTool.Execute(ctx, input)
	}
	return "", nil
}

// Tools 返回链中的所有工具
func (c *ConditionalChain) Tools() []Tool {
	tools := make([]Tool, 0, 2)
	if c.trueTool != nil {
		tools = append(tools, c.trueTool)
	}
	if c.falseTool != nil {
		tools = append(tools, c.falseTool)
	}
	return tools
}

// Name 返回链名称
func (c *ConditionalChain) Name() string {
	return c.name
}

// Description 返回链描述
func (c *ConditionalChain) Description() string {
	return c.description
}

// ParallelChain 并行工具链
//
// 并行执行多个工具，合并所有结果。
type ParallelChain struct {
	name        string
	description string
	tools       []Tool
}

// NewParallelChain 创建并行工具链
func NewParallelChain(name, description string, tools []Tool) *ParallelChain {
	return &ParallelChain{
		name:        name,
		description: description,
		tools:       tools,
	}
}

// Execute 并行执行工具链
func (c *ParallelChain) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	type result struct {
		index  int
		output string
		err    error
	}

	results := make(chan result, len(c.tools))

	// 并行执行所有工具
	for i, tool := range c.tools {
		go func(idx int, t Tool) {
			output, err := t.Execute(ctx, input)
			results <- result{index: idx, output: output, err: err}
		}(i, tool)
	}

	// 收集结果
	outputs := make([]string, len(c.tools))
	var errors []string

	for range c.tools {
		r := <-results
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", c.tools[r.index].Name(), r.err))
		} else {
			outputs[r.index] = r.output
		}
	}

	if len(errors) > 0 {
		return "", fmt.Errorf("parallel chain errors: %s", strings.Join(errors, "; "))
	}

	// 合并输出
	return strings.Join(outputs, "\n---\n"), nil
}

// Tools 返回链中的所有工具
func (c *ParallelChain) Tools() []Tool {
	return c.tools
}

// Name 返回链名称
func (c *ParallelChain) Name() string {
	return c.name
}

// Description 返回链描述
func (c *ParallelChain) Description() string {
	return c.description
}
