package tools

// ParameterSchema 定义工具参数的 JSON Schema
type ParameterSchema struct {
	// Type 参数类型（通常为 "object"）
	Type string `json:"type"`
	// Properties 参数属性定义
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	// Required 必需参数列表
	Required []string `json:"required,omitempty"`
	// AdditionalProperties 是否允许额外属性
	AdditionalProperties bool `json:"additionalProperties,omitempty"`
}

// PropertySchema 定义单个属性的 Schema
type PropertySchema struct {
	// Type 属性类型: "string", "number", "integer", "boolean", "array", "object"
	Type string `json:"type"`
	// Description 属性描述
	Description string `json:"description,omitempty"`
	// Enum 枚举值（可选）
	Enum []string `json:"enum,omitempty"`
	// Default 默认值（可选）
	Default interface{} `json:"default,omitempty"`
	// Items 数组元素 Schema（当 Type="array" 时）
	Items *PropertySchema `json:"items,omitempty"`
	// Properties 对象属性（当 Type="object" 时）
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	// Required 必需属性（当 Type="object" 时）
	Required []string `json:"required,omitempty"`
	// Minimum 最小值（数值类型）
	Minimum *float64 `json:"minimum,omitempty"`
	// Maximum 最大值（数值类型）
	Maximum *float64 `json:"maximum,omitempty"`
	// MinLength 最小长度（字符串类型）
	MinLength *int `json:"minLength,omitempty"`
	// MaxLength 最大长度（字符串类型）
	MaxLength *int `json:"maxLength,omitempty"`
	// Pattern 正则模式（字符串类型）
	Pattern string `json:"pattern,omitempty"`
}

// ToolDefinition 工具定义（用于序列化）
type ToolDefinition struct {
	// Name 工具名称
	Name string `json:"name"`
	// Description 工具描述
	Description string `json:"description"`
	// Parameters 参数 Schema
	Parameters ParameterSchema `json:"parameters"`
}

// ToDefinition 将 Tool 转换为 ToolDefinition
func ToDefinition(t Tool) ToolDefinition {
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

// ToLLMToolDefinition 将 Tool 转换为 LLM 工具定义格式
func ToLLMToolDefinition(t Tool) map[string]interface{} {
	schema := t.Parameters()
	return map[string]interface{}{
		"name":        t.Name(),
		"description": t.Description(),
		"parameters": map[string]interface{}{
			"type":       schema.Type,
			"properties": schema.Properties,
			"required":   schema.Required,
		},
	}
}

// ToolResult 工具执行结果
type ToolResult struct {
	// Name 工具名称
	Name string `json:"name"`
	// Success 是否成功
	Success bool `json:"success"`
	// Result 执行结果
	Result string `json:"result"`
	// Error 错误信息（如有）
	Error string `json:"error,omitempty"`
}

// NewToolResult 创建成功的工具结果
func NewToolResult(name, result string) ToolResult {
	return ToolResult{
		Name:    name,
		Success: true,
		Result:  result,
	}
}

// NewToolError 创建失败的工具结果
func NewToolError(name string, err error) ToolResult {
	return ToolResult{
		Name:    name,
		Success: false,
		Error:   err.Error(),
	}
}
