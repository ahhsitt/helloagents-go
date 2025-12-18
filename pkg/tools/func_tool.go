package tools

import (
	"context"
	"fmt"
)

// FuncTool 通过函数快速创建工具
//
// 使用示例:
//
//	weatherTool := tools.NewFuncTool(
//	    "weather",
//	    "Get weather information for a location",
//	    tools.ParameterSchema{
//	        Type: "object",
//	        Properties: map[string]tools.PropertySchema{
//	            "location": {Type: "string", Description: "City name"},
//	        },
//	        Required: []string{"location"},
//	    },
//	    func(ctx context.Context, args map[string]interface{}) (string, error) {
//	        location := args["location"].(string)
//	        return fmt.Sprintf("Weather in %s: Sunny, 25°C", location), nil
//	    },
//	)
type FuncTool struct {
	name        string
	description string
	params      ParameterSchema
	fn          ToolFunc
	validator   ValidatorFunc
}

// ToolFunc 工具执行函数类型
type ToolFunc func(ctx context.Context, args map[string]interface{}) (string, error)

// ValidatorFunc 参数验证函数类型
type ValidatorFunc func(args map[string]interface{}) error

// FuncToolOption FuncTool 配置选项
type FuncToolOption func(*FuncTool)

// NewFuncTool 创建函数工具
func NewFuncTool(name, description string, params ParameterSchema, fn ToolFunc, opts ...FuncToolOption) *FuncTool {
	t := &FuncTool{
		name:        name,
		description: description,
		params:      params,
		fn:          fn,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithValidator 设置参数验证函数
func WithValidator(v ValidatorFunc) FuncToolOption {
	return func(t *FuncTool) {
		t.validator = v
	}
}

// Name 返回工具名称
func (t *FuncTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *FuncTool) Description() string {
	return t.description
}

// Parameters 返回参数 Schema
func (t *FuncTool) Parameters() ParameterSchema {
	return t.params
}

// Execute 执行工具
func (t *FuncTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if t.fn == nil {
		return "", fmt.Errorf("tool function not set")
	}
	return t.fn(ctx, args)
}

// Validate 验证参数
func (t *FuncTool) Validate(args map[string]interface{}) error {
	if t.validator != nil {
		return t.validator(args)
	}
	// 默认验证必需参数
	return validateRequired(t.params, args)
}

// validateRequired 验证必需参数是否存在
func validateRequired(schema ParameterSchema, args map[string]interface{}) error {
	for _, req := range schema.Required {
		if _, ok := args[req]; !ok {
			return fmt.Errorf("missing required parameter: %s", req)
		}
	}
	return nil
}

// compile-time interface check
var _ Tool = (*FuncTool)(nil)
var _ ToolWithValidation = (*FuncTool)(nil)

// SimpleTool 更简化的工具创建方式
//
// 适用于只需要简单字符串参数的工具。
type SimpleTool struct {
	name        string
	description string
	paramName   string
	paramDesc   string
	fn          func(ctx context.Context, input string) (string, error)
}

// NewSimpleTool 创建简单工具（单个字符串参数）
func NewSimpleTool(name, description, paramName, paramDesc string, fn func(ctx context.Context, input string) (string, error)) *SimpleTool {
	return &SimpleTool{
		name:        name,
		description: description,
		paramName:   paramName,
		paramDesc:   paramDesc,
		fn:          fn,
	}
}

// Name 返回工具名称
func (t *SimpleTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *SimpleTool) Description() string {
	return t.description
}

// Parameters 返回参数 Schema
func (t *SimpleTool) Parameters() ParameterSchema {
	return ParameterSchema{
		Type: "object",
		Properties: map[string]PropertySchema{
			t.paramName: {
				Type:        "string",
				Description: t.paramDesc,
			},
		},
		Required: []string{t.paramName},
	}
}

// Execute 执行工具
func (t *SimpleTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	inputRaw, ok := args[t.paramName]
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", t.paramName)
	}

	input, ok := inputRaw.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string", t.paramName)
	}

	return t.fn(ctx, input)
}

// compile-time interface check
var _ Tool = (*SimpleTool)(nil)
