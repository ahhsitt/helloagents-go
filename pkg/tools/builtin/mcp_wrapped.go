package builtin

import (
	"context"
	"encoding/json"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// MCPWrappedTool 包装的 MCP 工具
//
// 将 MCP 服务器的单个工具包装成 Agent 可直接调用的 Tool 接口
type MCPWrappedTool struct {
	mcpTool     *MCPTool
	toolInfo    mcp.ToolInfo
	name        string
	description string
}

// NewMCPWrappedTool 创建包装的 MCP 工具
func NewMCPWrappedTool(mcpTool *MCPTool, toolInfo mcp.ToolInfo, prefix string) *MCPWrappedTool {
	return &MCPWrappedTool{
		mcpTool:     mcpTool,
		toolInfo:    toolInfo,
		name:        prefix + toolInfo.Name,
		description: toolInfo.Description,
	}
}

// Name 返回工具名称
func (t *MCPWrappedTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *MCPWrappedTool) Description() string {
	return t.description
}

// Parameters 返回参数 Schema
func (t *MCPWrappedTool) Parameters() tools.ParameterSchema {
	schema := tools.ParameterSchema{
		Type:       "object",
		Properties: make(map[string]tools.PropertySchema),
	}

	// 从 InputSchema 转换
	if t.toolInfo.InputSchema != nil {
		if props, ok := t.toolInfo.InputSchema["properties"].(map[string]interface{}); ok {
			for name, propRaw := range props {
				if prop, ok := propRaw.(map[string]interface{}); ok {
					ps := tools.PropertySchema{}
					if t, ok := prop["type"].(string); ok {
						ps.Type = t
					}
					if d, ok := prop["description"].(string); ok {
						ps.Description = d
					}
					if e, ok := prop["enum"].([]interface{}); ok {
						for _, v := range e {
							if s, ok := v.(string); ok {
								ps.Enum = append(ps.Enum, s)
							}
						}
					}
					schema.Properties[name] = ps
				}
			}
		}

		if req, ok := t.toolInfo.InputSchema["required"].([]interface{}); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					schema.Required = append(schema.Required, s)
				}
			}
		}
	}

	return schema
}

// Execute 执行工具
func (t *MCPWrappedTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 直接调用 MCPTool 的 callTool 方法
	return t.mcpTool.Execute(ctx, map[string]interface{}{
		"action":    "call_tool",
		"tool_name": t.toolInfo.Name,
		"arguments": args,
	})
}

// 确保实现 Tool 接口
var _ tools.Tool = (*MCPWrappedTool)(nil)

// 辅助函数

// toFloat64 将 interface{} 转换为 float64
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// jsonMarshal JSON 编码
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// jsonUnmarshal JSON 解码
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// mustMarshal JSON 编码，忽略错误
func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
