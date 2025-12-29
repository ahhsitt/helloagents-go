package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// ToolHandler 工具处理函数类型
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (string, error)

// ResourceHandler 资源处理函数类型
type ResourceHandler func(ctx context.Context) (string, error)

// PromptHandler 提示词处理函数类型
type PromptHandler func(ctx context.Context, arguments map[string]string) ([]PromptMessage, error)

// ServerTool 服务器工具定义
type ServerTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     ToolHandler
}

// ServerResource 服务器资源定义
type ServerResource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Handler     ResourceHandler
}

// ServerPrompt 服务器提示词定义
type ServerPrompt struct {
	Name        string
	Description string
	Arguments   []PromptArgument
	Handler     PromptHandler
}

// Server MCP 服务器
//
// 将本地工具、资源、提示词以 MCP 协议暴露给外部客户端。
//
// 使用示例:
//
//	server := mcp.NewServer("my-server", "My MCP Server")
//
//	server.AddTool(mcp.ServerTool{
//	    Name:        "calculator",
//	    Description: "Perform calculations",
//	    InputSchema: map[string]interface{}{
//	        "type": "object",
//	        "properties": map[string]interface{}{
//	            "expression": map[string]interface{}{
//	                "type":        "string",
//	                "description": "Math expression to evaluate",
//	            },
//	        },
//	        "required": []string{"expression"},
//	    },
//	    Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
//	        expr := args["expression"].(string)
//	        return fmt.Sprintf("Result: %s", expr), nil
//	    },
//	})
//
//	server.Run(ctx, os.Stdin, os.Stdout)
type Server struct {
	name        string
	version     string
	description string

	tools     map[string]*ServerTool
	resources map[string]*ServerResource
	prompts   map[string]*ServerPrompt

	mu sync.RWMutex
}

// NewServer 创建 MCP 服务器
func NewServer(name, description string) *Server {
	return &Server{
		name:        name,
		version:     "1.0.0",
		description: description,
		tools:       make(map[string]*ServerTool),
		resources:   make(map[string]*ServerResource),
		prompts:     make(map[string]*ServerPrompt),
	}
}

// AddTool 添加工具
func (s *Server) AddTool(tool ServerTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = &tool
}

// AddResource 添加资源
func (s *Server) AddResource(resource ServerResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[resource.URI] = &resource
}

// AddPrompt 添加提示词
func (s *Server) AddPrompt(prompt ServerPrompt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[prompt.Name] = &prompt
}

// Run 运行服务器（Stdio 模式）
//
// 从 reader 读取请求，将响应写入 writer。
// 如果 reader 和 writer 为 nil，则使用 os.Stdin 和 os.Stdout。
func (s *Server) Run(ctx context.Context, reader io.Reader, writer io.Writer) error {
	if reader == nil {
		reader = os.Stdin
	}
	if writer == nil {
		writer = os.Stdout
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line size

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			return nil // EOF
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		response := s.handleRequest(ctx, line)
		if response != nil {
			responseBytes, err := json.Marshal(response)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(writer, "%s\n", responseBytes); err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
		}
	}
}

// handleRequest 处理单个请求
func (s *Server) handleRequest(ctx context.Context, data []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return s.errorResponse(nil, -32700, "Parse error", err.Error())
	}

	// 如果是通知（没有 ID），不返回响应
	if req.ID == nil {
		s.handleNotification(ctx, &req)
		return nil
	}

	return s.handleCall(ctx, &req)
}

// handleNotification 处理通知
func (s *Server) handleNotification(_ context.Context, req *JSONRPCRequest) {
	// 目前只处理 initialized 通知
	switch req.Method {
	case MethodInitialized:
		// 客户端已初始化，可以开始处理请求
	}
}

// handleCall 处理调用
func (s *Server) handleCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(ctx, req)
	case MethodListTools:
		return s.handleListTools(ctx, req)
	case MethodCallTool:
		return s.handleCallTool(ctx, req)
	case MethodListResources:
		return s.handleListResources(ctx, req)
	case MethodReadResource:
		return s.handleReadResource(ctx, req)
	case MethodListPrompts:
		return s.handleListPrompts(ctx, req)
	case MethodGetPrompt:
		return s.handleGetPrompt(ctx, req)
	case MethodPing:
		return s.handlePing(ctx, req)
	default:
		return s.errorResponse(req.ID, -32601, "Method not found", req.Method)
	}
}

// handleInitialize 处理初始化请求
func (s *Server) handleInitialize(_ context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities: Capabilities{
			Tools:     &ToolsCapability{ListChanged: false},
			Resources: &ResourcesCapability{ListChanged: false},
			Prompts:   &PromptsCapability{ListChanged: false},
		},
		ServerInfo: Implementation{
			Name:    s.name,
			Version: s.version,
		},
	}

	return s.successResponse(req.ID, result)
}

// handleListTools 处理列出工具请求
func (s *Server) handleListTools(_ context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		schemaBytes, _ := json.Marshal(t.InputSchema)
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schemaBytes,
		})
	}

	return s.successResponse(req.ID, ListToolsResult{Tools: tools})
}

// handleCallTool 处理调用工具请求
func (s *Server) handleCallTool(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return s.errorResponse(req.ID, -32602, "Tool not found", params.Name)
	}

	result, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		return s.successResponse(req.ID, CallToolResult{
			Content: []Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
	}

	return s.successResponse(req.ID, CallToolResult{
		Content: []Content{{Type: "text", Text: result}},
		IsError: false,
	})
}

// handleListResources 处理列出资源请求
func (s *Server) handleListResources(_ context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, Resource{
			URI:         r.URI,
			Name:        r.Name,
			Description: r.Description,
			MimeType:    r.MimeType,
		})
	}

	return s.successResponse(req.ID, ListResourcesResult{Resources: resources})
}

// handleReadResource 处理读取资源请求
func (s *Server) handleReadResource(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	s.mu.RLock()
	resource, ok := s.resources[params.URI]
	s.mu.RUnlock()

	if !ok {
		return s.errorResponse(req.ID, -32602, "Resource not found", params.URI)
	}

	content, err := resource.Handler(ctx)
	if err != nil {
		return s.errorResponse(req.ID, -32603, "Internal error", err.Error())
	}

	return s.successResponse(req.ID, ReadResourceResult{
		Contents: []ResourceContent{{
			URI:      resource.URI,
			MimeType: resource.MimeType,
			Text:     content,
		}},
	})
}

// handleListPrompts 处理列出提示词请求
func (s *Server) handleListPrompts(_ context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]Prompt, 0, len(s.prompts))
	for _, p := range s.prompts {
		prompts = append(prompts, Prompt{
			Name:        p.Name,
			Description: p.Description,
			Arguments:   p.Arguments,
		})
	}

	return s.successResponse(req.ID, ListPromptsResult{Prompts: prompts})
}

// handleGetPrompt 处理获取提示词请求
func (s *Server) handleGetPrompt(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params GetPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	s.mu.RLock()
	prompt, ok := s.prompts[params.Name]
	s.mu.RUnlock()

	if !ok {
		return s.errorResponse(req.ID, -32602, "Prompt not found", params.Name)
	}

	messages, err := prompt.Handler(ctx, params.Arguments)
	if err != nil {
		return s.errorResponse(req.ID, -32603, "Internal error", err.Error())
	}

	return s.successResponse(req.ID, GetPromptResult{
		Description: prompt.Description,
		Messages:    messages,
	})
}

// handlePing 处理 ping 请求
func (s *Server) handlePing(_ context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	return s.successResponse(req.ID, map[string]interface{}{})
}

// successResponse 创建成功响应
func (s *Server) successResponse(id interface{}, result interface{}) *JSONRPCResponse {
	resultBytes, _ := json.Marshal(result)
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultBytes,
	}
}

// errorResponse 创建错误响应
func (s *Server) errorResponse(id interface{}, code int, message, data string) *JSONRPCResponse {
	var dataBytes json.RawMessage
	if data != "" {
		dataBytes, _ = json.Marshal(data)
	}

	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    dataBytes,
		},
	}
}

// Info 返回服务器信息
func (s *Server) Info() map[string]interface{} {
	return map[string]interface{}{
		"name":        s.name,
		"version":     s.version,
		"description": s.description,
		"protocol":    "MCP",
	}
}
