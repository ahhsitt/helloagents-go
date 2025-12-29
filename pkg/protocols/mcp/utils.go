package mcp

import (
	"encoding/json"
	"fmt"
)

// Context MCP 上下文
type Context struct {
	Messages  []map[string]interface{} `json:"messages"`
	Tools     []map[string]interface{} `json:"tools"`
	Resources []map[string]interface{} `json:"resources"`
	Metadata  map[string]interface{}   `json:"metadata"`
}

// CreateContext 创建 MCP 上下文对象
//
// 参数:
//   - messages: 消息列表（可选）
//   - tools: 工具列表（可选）
//   - resources: 资源列表（可选）
//   - metadata: 元数据（可选）
//
// 返回:
//   - 上下文对象
func CreateContext(
	messages []map[string]interface{},
	tools []map[string]interface{},
	resources []map[string]interface{},
	metadata map[string]interface{},
) *Context {
	if messages == nil {
		messages = []map[string]interface{}{}
	}
	if tools == nil {
		tools = []map[string]interface{}{}
	}
	if resources == nil {
		resources = []map[string]interface{}{}
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}

	return &Context{
		Messages:  messages,
		Tools:     tools,
		Resources: resources,
		Metadata:  metadata,
	}
}

// ParseContext 解析 MCP 上下文
//
// 参数:
//   - data: JSON 字符串或字节切片
//
// 返回:
//   - 解析后的上下文对象
//   - 错误（如果 JSON 无效）
func ParseContext(data interface{}) (*Context, error) {
	var jsonData []byte

	switch v := data.(type) {
	case string:
		jsonData = []byte(v)
	case []byte:
		jsonData = v
	default:
		return nil, fmt.Errorf("context must be a string or []byte")
	}

	var ctx Context
	if err := json.Unmarshal(jsonData, &ctx); err != nil {
		return nil, fmt.Errorf("invalid JSON context: %w", err)
	}

	// 确保必需字段存在
	if ctx.Messages == nil {
		ctx.Messages = []map[string]interface{}{}
	}
	if ctx.Tools == nil {
		ctx.Tools = []map[string]interface{}{}
	}
	if ctx.Resources == nil {
		ctx.Resources = []map[string]interface{}{}
	}
	if ctx.Metadata == nil {
		ctx.Metadata = map[string]interface{}{}
	}

	return &ctx, nil
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Message string                 `json:"message"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// CreateErrorResponse 创建错误响应
//
// 参数:
//   - message: 错误消息
//   - code: 错误代码（可选，默认 "UNKNOWN_ERROR"）
//   - details: 错误详情（可选）
//
// 返回:
//   - 错误响应对象
func CreateErrorResponse(message string, code string, details map[string]interface{}) *ErrorResponse {
	if code == "" {
		code = "UNKNOWN_ERROR"
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Code:    code,
			Details: details,
		},
	}
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CreateSuccessResponse 创建成功响应
//
// 参数:
//   - data: 响应数据
//   - metadata: 元数据（可选）
//
// 返回:
//   - 成功响应对象
func CreateSuccessResponse(data interface{}, metadata map[string]interface{}) *SuccessResponse {
	return &SuccessResponse{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	}
}

// ToJSON 将上下文转换为 JSON 字符串
func (c *Context) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal context: %w", err)
	}
	return string(data), nil
}

// AddMessage 添加消息到上下文
func (c *Context) AddMessage(role, content string) {
	c.Messages = append(c.Messages, map[string]interface{}{
		"role":    role,
		"content": content,
	})
}

// AddTool 添加工具到上下文
func (c *Context) AddTool(name, description string) {
	c.Tools = append(c.Tools, map[string]interface{}{
		"name":        name,
		"description": description,
	})
}

// AddResource 添加资源到上下文
func (c *Context) AddResource(uri, name string) {
	c.Resources = append(c.Resources, map[string]interface{}{
		"uri":  uri,
		"name": name,
	})
}

// SetMetadata 设置元数据
func (c *Context) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}

// GetMetadata 获取元数据
func (c *Context) GetMetadata(key string) (interface{}, bool) {
	v, ok := c.Metadata[key]
	return v, ok
}
