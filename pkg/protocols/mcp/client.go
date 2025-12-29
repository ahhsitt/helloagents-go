package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

// Client MCP 客户端
//
// 用于连接 MCP 服务器，调用工具、读取资源、获取提示词。
//
// 使用示例:
//
//	// 创建 Stdio 传输
//	transport, err := mcp.NewStdioTransport(mcp.StdioTransportConfig{
//	    Command: "python",
//	    Args:    []string{"server.py"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 创建客户端
//	client := mcp.NewClient(transport)
//	defer client.Close()
//
//	// 初始化连接
//	if err := client.Initialize(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// 列出工具
//	tools, err := client.ListTools(ctx)
type Client struct {
	transport   Transport
	requestID   atomic.Int64
	initialized atomic.Bool
	serverInfo  *Implementation
	mu          sync.Mutex
}

// NewClient 创建 MCP 客户端
func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
	}
}

// Initialize 初始化客户端连接
func (c *Client) Initialize(ctx context.Context) error {
	if c.initialized.Load() {
		return nil
	}

	params := InitializeParams{
		ProtocolVersion: MCPVersion,
		Capabilities: Capabilities{
			Tools:     &ToolsCapability{},
			Resources: &ResourcesCapability{},
			Prompts:   &PromptsCapability{},
		},
		ClientInfo: Implementation{
			Name:    "helloagents-go",
			Version: "1.0.0",
		},
	}

	result, err := c.call(ctx, MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	c.serverInfo = &initResult.ServerInfo

	// 发送 initialized 通知
	if err := c.notify(ctx, MethodInitialized, nil); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.initialized.Store(true)
	return nil
}

// ListTools 列出所有可用工具
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	result, err := c.call(ctx, MethodListTools, nil)
	if err != nil {
		return nil, fmt.Errorf("list tools failed: %w", err)
	}

	var listResult ListToolsResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse list tools result: %w", err)
	}

	tools := make([]ToolInfo, len(listResult.Tools))
	for i, t := range listResult.Tools {
		var schema map[string]interface{}
		if len(t.InputSchema) > 0 {
			if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
				schema = make(map[string]interface{})
			}
		}
		tools[i] = ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		}
	}

	return tools, nil
}

// CallTool 调用工具
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (string, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return "", err
		}
	}

	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	result, err := c.call(ctx, MethodCallTool, params)
	if err != nil {
		return "", fmt.Errorf("call tool failed: %w", err)
	}

	var callResult CallToolResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return "", fmt.Errorf("failed to parse call tool result: %w", err)
	}

	if callResult.IsError {
		if len(callResult.Content) > 0 && callResult.Content[0].Text != "" {
			return "", fmt.Errorf("tool error: %s", callResult.Content[0].Text)
		}
		return "", fmt.Errorf("tool returned error")
	}

	// 提取文本内容
	var text string
	for _, content := range callResult.Content {
		if content.Type == "text" && content.Text != "" {
			if text != "" {
				text += "\n"
			}
			text += content.Text
		}
	}

	return text, nil
}

// ListResources 列出所有可用资源
func (c *Client) ListResources(ctx context.Context) ([]ResourceInfo, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	result, err := c.call(ctx, MethodListResources, nil)
	if err != nil {
		return nil, fmt.Errorf("list resources failed: %w", err)
	}

	var listResult ListResourcesResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse list resources result: %w", err)
	}

	resources := make([]ResourceInfo, len(listResult.Resources))
	for i, r := range listResult.Resources {
		resources[i] = ResourceInfo(r)
	}

	return resources, nil
}

// ReadResource 读取资源内容
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return "", err
		}
	}

	params := ReadResourceParams{URI: uri}

	result, err := c.call(ctx, MethodReadResource, params)
	if err != nil {
		return "", fmt.Errorf("read resource failed: %w", err)
	}

	var readResult ReadResourceResult
	if err := json.Unmarshal(result, &readResult); err != nil {
		return "", fmt.Errorf("failed to parse read resource result: %w", err)
	}

	// 提取内容
	var content string
	for _, c := range readResult.Contents {
		if c.Text != "" {
			if content != "" {
				content += "\n"
			}
			content += c.Text
		}
	}

	return content, nil
}

// ListPrompts 列出所有可用提示词
func (c *Client) ListPrompts(ctx context.Context) ([]PromptInfo, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	result, err := c.call(ctx, MethodListPrompts, nil)
	if err != nil {
		return nil, fmt.Errorf("list prompts failed: %w", err)
	}

	var listResult ListPromptsResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse list prompts result: %w", err)
	}

	prompts := make([]PromptInfo, len(listResult.Prompts))
	for i, p := range listResult.Prompts {
		prompts[i] = PromptInfo(p)
	}

	return prompts, nil
}

// GetPrompt 获取提示词内容
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) ([]PromptMessage, error) {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	params := GetPromptParams{
		Name:      name,
		Arguments: arguments,
	}

	result, err := c.call(ctx, MethodGetPrompt, params)
	if err != nil {
		return nil, fmt.Errorf("get prompt failed: %w", err)
	}

	var getResult GetPromptResult
	if err := json.Unmarshal(result, &getResult); err != nil {
		return nil, fmt.Errorf("failed to parse get prompt result: %w", err)
	}

	return getResult.Messages, nil
}

// Ping 测试服务器连接
func (c *Client) Ping(ctx context.Context) error {
	if !c.initialized.Load() {
		if err := c.Initialize(ctx); err != nil {
			return err
		}
	}

	_, err := c.call(ctx, MethodPing, nil)
	return err
}

// ServerInfo 返回服务器信息
func (c *Client) ServerInfo() *Implementation {
	return c.serverInfo
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.transport.Close()
}

// call 发送请求并等待响应
func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.requestID.Add(1)

	request, err := NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	response, err := c.transport.Send(ctx, request)
	if err != nil {
		return nil, err
	}

	resp, err := ParseResponse(response)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// notify 发送通知（不等待响应）
func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 通知没有 ID
	request, err := NewRequest(nil, method, params)
	if err != nil {
		return err
	}

	// 对于通知，我们不等待响应
	_, err = c.transport.Send(ctx, request)
	return err
}
