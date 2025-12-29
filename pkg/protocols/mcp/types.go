// Package mcp 实现 MCP (Model Context Protocol) 协议
//
// MCP 是一种标准化的协议，用于 AI 模型与外部工具/服务之间的通信。
// 本包提供：
//   - MCPClient: 连接到 MCP 服务器，调用工具、读取资源、获取提示词
//   - MCPServer: 将本地工具以 MCP 协议暴露给外部客户端
//   - Transport: 传输层抽象，支持 Stdio 和 HTTP 两种方式
package mcp

import (
	"encoding/json"
)

// MCP 协议版本
const (
	MCPVersion     = "2024-11-05"
	JSONRPCVersion = "2.0"
)

// JSONRPCRequest JSON-RPC 2.0 请求结构
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 2.0 响应结构
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError JSON-RPC 2.0 错误结构
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// MCP 方法名称
const (
	MethodInitialize    = "initialize"
	MethodInitialized   = "notifications/initialized"
	MethodListTools     = "tools/list"
	MethodCallTool      = "tools/call"
	MethodListResources = "resources/list"
	MethodReadResource  = "resources/read"
	MethodListPrompts   = "prompts/list"
	MethodGetPrompt     = "prompts/get"
	MethodPing          = "ping"
)

// InitializeParams 初始化请求参数
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ClientInfo      Implementation `json:"clientInfo"`
}

// InitializeResult 初始化响应结果
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    Capabilities   `json:"capabilities"`
	ServerInfo      Implementation `json:"serverInfo"`
}

// Capabilities 协议能力
type Capabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability 工具能力
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability 提示词能力
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Implementation 客户端/服务器实现信息
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool MCP 工具定义
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListToolsResult 列出工具的响应
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams 调用工具的请求参数
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult 调用工具的响应
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content 内容块
type Content struct {
	Type     string           `json:"type"` // "text", "image", "resource"
	Text     string           `json:"text,omitempty"`
	Data     string           `json:"data,omitempty"`
	MimeType string           `json:"mimeType,omitempty"`
	Resource *ResourceContent `json:"resource,omitempty"`
}

// Resource MCP 资源定义
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult 列出资源的响应
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// ReadResourceParams 读取资源的请求参数
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult 读取资源的响应
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent 资源内容
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// Prompt MCP 提示词定义
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument 提示词参数
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult 列出提示词的响应
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams 获取提示词的请求参数
type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult 获取提示词的响应
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage 提示词消息
type PromptMessage struct {
	Role    string  `json:"role"` // "user", "assistant"
	Content Content `json:"content"`
}

// ToolInfo 简化的工具信息（用于展示）
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ResourceInfo 简化的资源信息（用于展示）
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PromptInfo 简化的提示词信息（用于展示）
type PromptInfo struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments"`
}
