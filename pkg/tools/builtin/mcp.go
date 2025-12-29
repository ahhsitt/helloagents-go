package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// MCPServerEnvMap MCP 服务器环境变量映射表
// 用于自动检测常见 MCP 服务器需要的环境变量
var MCPServerEnvMap = map[string][]string{
	"server-github":       {"GITHUB_PERSONAL_ACCESS_TOKEN"},
	"server-slack":        {"SLACK_BOT_TOKEN", "SLACK_TEAM_ID"},
	"server-google-drive": {"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REFRESH_TOKEN"},
	"server-postgres":     {"POSTGRES_CONNECTION_STRING"},
	"server-sqlite":       {},
	"server-filesystem":   {},
}

// MCPTool MCP (Model Context Protocol) 工具
//
// 连接到 MCP 服务器并调用其提供的工具、资源和提示词。
//
// 功能：
//   - 列出服务器提供的工具
//   - 调用服务器工具
//   - 读取服务器资源
//   - 获取提示词模板
//
// 使用示例:
//
//	// 方式1: 使用内置演示服务器
//	tool := builtin.NewMCPTool()
//	result, _ := tool.Execute(ctx, map[string]interface{}{"action": "list_tools"})
//
//	// 方式2: 连接到外部 MCP 服务器
//	tool := builtin.NewMCPTool(
//	    builtin.WithMCPCommand("python", "server.py"),
//	)
//
//	// 方式3: 使用自定义配置
//	tool := builtin.NewMCPTool(
//	    builtin.WithMCPName("github"),
//	    builtin.WithMCPCommand("npx", "-y", "@modelcontextprotocol/server-github"),
//	    builtin.WithMCPEnv(map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "xxx"}),
//	)
type MCPTool struct {
	name           string
	description    string
	serverCommand  []string
	serverEnv      map[string]string
	envKeys        []string
	autoExpand     bool
	prefix         string
	client         *mcp.Client
	transport      mcp.Transport
	availableTools []mcp.ToolInfo
	builtinServer  *mcp.Server
	mu             sync.Mutex
	initialized    bool
}

// MCPToolOption MCPTool 配置选项
type MCPToolOption func(*MCPTool)

// WithMCPName 设置工具名称
func WithMCPName(name string) MCPToolOption {
	return func(t *MCPTool) {
		t.name = name
	}
}

// WithMCPDescription 设置工具描述
func WithMCPDescription(description string) MCPToolOption {
	return func(t *MCPTool) {
		t.description = description
	}
}

// WithMCPCommand 设置服务器启动命令
func WithMCPCommand(command string, args ...string) MCPToolOption {
	return func(t *MCPTool) {
		t.serverCommand = append([]string{command}, args...)
	}
}

// WithMCPEnv 设置环境变量（最高优先级）
func WithMCPEnv(env map[string]string) MCPToolOption {
	return func(t *MCPTool) {
		t.serverEnv = env
	}
}

// WithMCPEnvKeys 设置要从系统环境变量加载的 key 列表
func WithMCPEnvKeys(keys []string) MCPToolOption {
	return func(t *MCPTool) {
		t.envKeys = keys
	}
}

// WithMCPAutoExpand 设置是否自动展开工具
func WithMCPAutoExpand(expand bool) MCPToolOption {
	return func(t *MCPTool) {
		t.autoExpand = expand
	}
}

// WithMCPServer 设置内置服务器（用于测试）
func WithMCPServer(server *mcp.Server) MCPToolOption {
	return func(t *MCPTool) {
		t.builtinServer = server
	}
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool(opts ...MCPToolOption) *MCPTool {
	t := &MCPTool{
		name:        "mcp",
		description: "连接到 MCP 服务器，调用工具、读取资源和获取提示词",
		autoExpand:  true,
		serverEnv:   make(map[string]string),
	}

	for _, opt := range opts {
		opt(t)
	}

	if t.autoExpand {
		t.prefix = t.name + "_"
	}

	return t
}

// Name 返回工具名称
func (t *MCPTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *MCPTool) Description() string {
	if t.description != "" {
		return t.description
	}

	if len(t.availableTools) > 0 {
		if t.autoExpand {
			return fmt.Sprintf("MCP工具服务器，包含%d个工具。这些工具会自动展开为独立的工具供Agent使用。", len(t.availableTools))
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("MCP工具服务器，提供%d个工具：\n", len(t.availableTools)))
		for _, tool := range t.availableTools {
			desc := tool.Description
			if idx := strings.Index(desc, "."); idx > 0 {
				desc = desc[:idx]
			}
			sb.WriteString(fmt.Sprintf("  • %s: %s\n", tool.Name, desc))
		}
		sb.WriteString("\n调用格式：返回JSON格式的参数\n")
		sb.WriteString(`{"action": "call_tool", "tool_name": "工具名", "arguments": {...}}`)
		return sb.String()
	}

	return "连接到 MCP 服务器，调用工具、读取资源和获取提示词。支持内置服务器和外部服务器。"
}

// Parameters 返回参数 Schema
func (t *MCPTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"action": {
				Type:        "string",
				Description: "操作类型: list_tools, call_tool, list_resources, read_resource, list_prompts, get_prompt",
				Enum:        []string{"list_tools", "call_tool", "list_resources", "read_resource", "list_prompts", "get_prompt"},
			},
			"tool_name": {
				Type:        "string",
				Description: "工具名称（call_tool 操作需要）",
			},
			"arguments": {
				Type:        "object",
				Description: "工具参数（call_tool 操作需要）",
			},
			"uri": {
				Type:        "string",
				Description: "资源 URI（read_resource 操作需要）",
			},
			"prompt_name": {
				Type:        "string",
				Description: "提示词名称（get_prompt 操作需要）",
			},
			"prompt_arguments": {
				Type:        "object",
				Description: "提示词参数（get_prompt 操作可选）",
			},
		},
		Required: []string{"action"},
	}
}

// Execute 执行 MCP 操作
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 确保客户端已初始化
	if err := t.ensureInitialized(ctx); err != nil {
		return "", fmt.Errorf("MCP 初始化失败: %w", err)
	}

	// 智能推断 action
	action, _ := args["action"].(string)
	if action == "" {
		if _, ok := args["tool_name"]; ok {
			action = "call_tool"
		}
	}

	if action == "" {
		return "", fmt.Errorf("必须指定 action 参数或 tool_name 参数")
	}

	switch strings.ToLower(action) {
	case "list_tools":
		return t.listTools(ctx)
	case "call_tool":
		return t.callTool(ctx, args)
	case "list_resources":
		return t.listResources(ctx)
	case "read_resource":
		return t.readResource(ctx, args)
	case "list_prompts":
		return t.listPrompts(ctx)
	case "get_prompt":
		return t.getPrompt(ctx, args)
	default:
		return "", fmt.Errorf("不支持的操作: %s", action)
	}
}

// ensureInitialized 确保客户端已初始化
func (t *MCPTool) ensureInitialized(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.initialized {
		return nil
	}

	// 准备环境变量
	env := t.prepareEnv()

	// 创建传输层
	var transport mcp.Transport
	var err error

	if t.builtinServer != nil {
		// 使用内置服务器（内存传输）
		transport = t.createMemoryTransport(t.builtinServer)
	} else if len(t.serverCommand) > 0 {
		// 使用外部服务器命令
		transport, err = mcp.NewStdioTransport(mcp.StdioTransportConfig{
			Command: t.serverCommand[0],
			Args:    t.serverCommand[1:],
			Env:     env,
		})
		if err != nil {
			return fmt.Errorf("创建传输失败: %w", err)
		}
	} else {
		// 创建内置演示服务器
		server := t.createBuiltinServer()
		transport = t.createMemoryTransport(server)
	}

	t.transport = transport
	t.client = mcp.NewClient(transport)

	// 初始化连接
	if err := t.client.Initialize(ctx); err != nil {
		t.transport.Close()
		return fmt.Errorf("初始化连接失败: %w", err)
	}

	// 发现可用工具
	tools, err := t.client.ListTools(ctx)
	if err == nil {
		t.availableTools = tools
	}

	t.initialized = true
	return nil
}

// prepareEnv 准备环境变量
func (t *MCPTool) prepareEnv() map[string]string {
	env := make(map[string]string)

	// 1. 自动检测（优先级最低）
	if len(t.serverCommand) > 0 {
		for _, part := range t.serverCommand {
			if strings.Contains(part, "server-") {
				serverName := part
				if idx := strings.LastIndex(part, "/"); idx >= 0 {
					serverName = part[idx+1:]
				}
				if keys, ok := MCPServerEnvMap[serverName]; ok {
					for _, key := range keys {
						if value := os.Getenv(key); value != "" {
							env[key] = value
						}
					}
				}
				break
			}
		}
	}

	// 2. envKeys 指定的环境变量（优先级中等）
	for _, key := range t.envKeys {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}

	// 3. 直接传递的 env（优先级最高）
	for k, v := range t.serverEnv {
		env[k] = v
	}

	return env
}

// createBuiltinServer 创建内置演示服务器
func (t *MCPTool) createBuiltinServer() *mcp.Server {
	server := mcp.NewServer("HelloAgents-BuiltinServer", "HelloAgents 内置 MCP 演示服务器")

	// 加法工具
	server.AddTool(mcp.ServerTool{
		Name:        "add",
		Description: "加法计算器",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number", "description": "第一个数"},
				"b": map[string]interface{}{"type": "number", "description": "第二个数"},
			},
			"required": []string{"a", "b"},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (string, error) {
			a, _ := toFloat64(args["a"])
			b, _ := toFloat64(args["b"])
			return fmt.Sprintf("%.2f", a+b), nil
		},
	})

	// 减法工具
	server.AddTool(mcp.ServerTool{
		Name:        "subtract",
		Description: "减法计算器",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number", "description": "被减数"},
				"b": map[string]interface{}{"type": "number", "description": "减数"},
			},
			"required": []string{"a", "b"},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (string, error) {
			a, _ := toFloat64(args["a"])
			b, _ := toFloat64(args["b"])
			return fmt.Sprintf("%.2f", a-b), nil
		},
	})

	// 乘法工具
	server.AddTool(mcp.ServerTool{
		Name:        "multiply",
		Description: "乘法计算器",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number", "description": "第一个数"},
				"b": map[string]interface{}{"type": "number", "description": "第二个数"},
			},
			"required": []string{"a", "b"},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (string, error) {
			a, _ := toFloat64(args["a"])
			b, _ := toFloat64(args["b"])
			return fmt.Sprintf("%.2f", a*b), nil
		},
	})

	// 除法工具
	server.AddTool(mcp.ServerTool{
		Name:        "divide",
		Description: "除法计算器",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number", "description": "被除数"},
				"b": map[string]interface{}{"type": "number", "description": "除数"},
			},
			"required": []string{"a", "b"},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (string, error) {
			a, _ := toFloat64(args["a"])
			b, _ := toFloat64(args["b"])
			if b == 0 {
				return "", fmt.Errorf("除数不能为零")
			}
			return fmt.Sprintf("%.2f", a/b), nil
		},
	})

	// 问候工具
	server.AddTool(mcp.ServerTool{
		Name:        "greet",
		Description: "友好问候",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "要问候的名字", "default": "World"},
			},
		},
		Handler: func(_ context.Context, args map[string]interface{}) (string, error) {
			name, ok := args["name"].(string)
			if !ok || name == "" {
				name = "World"
			}
			return fmt.Sprintf("Hello, %s! 欢迎使用 HelloAgents MCP 工具！", name), nil
		},
	})

	return server
}

// createMemoryTransport 创建内存传输（用于内置服务器）
func (t *MCPTool) createMemoryTransport(server *mcp.Server) mcp.Transport {
	return mcp.NewMemoryTransport(func(request []byte) ([]byte, error) {
		// 这里需要一个简单的请求处理器
		// 由于 Server.Run 是阻塞的，我们使用一个内部处理方法
		return t.handleServerRequest(server, request)
	})
}

// handleServerRequest 处理内存传输的请求
func (t *MCPTool) handleServerRequest(server *mcp.Server, request []byte) ([]byte, error) {
	// 创建管道用于通信
	ctx := context.Background()

	// 解析请求
	var req mcp.JSONRPCRequest
	if err := jsonUnmarshal(request, &req); err != nil {
		return jsonMarshal(&mcp.JSONRPCResponse{
			JSONRPC: mcp.JSONRPCVersion,
			Error: &mcp.JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
	}

	// 根据方法调用相应的处理函数
	var result interface{}
	var rpcErr *mcp.JSONRPCError

	switch req.Method {
	case mcp.MethodInitialize:
		result = mcp.InitializeResult{
			ProtocolVersion: mcp.MCPVersion,
			Capabilities: mcp.Capabilities{
				Tools:     &mcp.ToolsCapability{},
				Resources: &mcp.ResourcesCapability{},
				Prompts:   &mcp.PromptsCapability{},
			},
			ServerInfo: mcp.Implementation{
				Name:    "HelloAgents-BuiltinServer",
				Version: "1.0.0",
			},
		}
	case mcp.MethodInitialized:
		// 通知，不需要响应
		return nil, nil
	case mcp.MethodListTools:
		tools := t.getServerTools(server)
		result = mcp.ListToolsResult{Tools: tools}
	case mcp.MethodCallTool:
		var params mcp.CallToolParams
		if err := jsonUnmarshal(req.Params, &params); err != nil {
			rpcErr = &mcp.JSONRPCError{Code: -32602, Message: "Invalid params"}
		} else {
			output, err := t.callServerTool(ctx, server, params.Name, params.Arguments)
			if err != nil {
				result = mcp.CallToolResult{
					Content: []mcp.Content{{Type: "text", Text: err.Error()}},
					IsError: true,
				}
			} else {
				result = mcp.CallToolResult{
					Content: []mcp.Content{{Type: "text", Text: output}},
					IsError: false,
				}
			}
		}
	case mcp.MethodListResources:
		result = mcp.ListResourcesResult{Resources: []mcp.Resource{}}
	case mcp.MethodReadResource:
		rpcErr = &mcp.JSONRPCError{Code: -32602, Message: "Resource not found"}
	case mcp.MethodListPrompts:
		result = mcp.ListPromptsResult{Prompts: []mcp.Prompt{}}
	case mcp.MethodGetPrompt:
		rpcErr = &mcp.JSONRPCError{Code: -32602, Message: "Prompt not found"}
	case mcp.MethodPing:
		result = map[string]interface{}{}
	default:
		rpcErr = &mcp.JSONRPCError{Code: -32601, Message: "Method not found"}
	}

	// 构建响应
	resp := &mcp.JSONRPCResponse{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      req.ID,
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else if result != nil {
		resultBytes, _ := jsonMarshal(result)
		resp.Result = resultBytes
	}

	return jsonMarshal(resp)
}

// getServerTools 获取服务器工具列表
func (t *MCPTool) getServerTools(_ *mcp.Server) []mcp.Tool {
	// 返回内置工具列表
	return []mcp.Tool{
		{Name: "add", Description: "加法计算器", InputSchema: mustMarshal(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"a": map[string]interface{}{"type": "number"}, "b": map[string]interface{}{"type": "number"}}, "required": []string{"a", "b"}})},
		{Name: "subtract", Description: "减法计算器", InputSchema: mustMarshal(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"a": map[string]interface{}{"type": "number"}, "b": map[string]interface{}{"type": "number"}}, "required": []string{"a", "b"}})},
		{Name: "multiply", Description: "乘法计算器", InputSchema: mustMarshal(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"a": map[string]interface{}{"type": "number"}, "b": map[string]interface{}{"type": "number"}}, "required": []string{"a", "b"}})},
		{Name: "divide", Description: "除法计算器", InputSchema: mustMarshal(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"a": map[string]interface{}{"type": "number"}, "b": map[string]interface{}{"type": "number"}}, "required": []string{"a", "b"}})},
		{Name: "greet", Description: "友好问候", InputSchema: mustMarshal(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}})},
	}
}

// callServerTool 调用服务器工具
func (t *MCPTool) callServerTool(ctx context.Context, _ *mcp.Server, name string, arguments map[string]interface{}) (string, error) {
	// 直接调用内置工具
	switch name {
	case "add":
		a, _ := toFloat64(arguments["a"])
		b, _ := toFloat64(arguments["b"])
		return fmt.Sprintf("%.2f", a+b), nil
	case "subtract":
		a, _ := toFloat64(arguments["a"])
		b, _ := toFloat64(arguments["b"])
		return fmt.Sprintf("%.2f", a-b), nil
	case "multiply":
		a, _ := toFloat64(arguments["a"])
		b, _ := toFloat64(arguments["b"])
		return fmt.Sprintf("%.2f", a*b), nil
	case "divide":
		a, _ := toFloat64(arguments["a"])
		b, _ := toFloat64(arguments["b"])
		if b == 0 {
			return "", fmt.Errorf("除数不能为零")
		}
		return fmt.Sprintf("%.2f", a/b), nil
	case "greet":
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			name = "World"
		}
		return fmt.Sprintf("Hello, %s! 欢迎使用 HelloAgents MCP 工具！", name), nil
	default:
		return "", fmt.Errorf("工具不存在: %s", name)
	}
}

// listTools 列出工具
func (t *MCPTool) listTools(ctx context.Context) (string, error) {
	tools, err := t.client.ListTools(ctx)
	if err != nil {
		return "", err
	}

	if len(tools) == 0 {
		return "没有找到可用的工具", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个工具:\n", len(tools)))
	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}
	return sb.String(), nil
}

// callTool 调用工具
func (t *MCPTool) callTool(ctx context.Context, args map[string]interface{}) (string, error) {
	toolName, ok := args["tool_name"].(string)
	if !ok || toolName == "" {
		return "", fmt.Errorf("必须指定 tool_name 参数")
	}

	arguments, _ := args["arguments"].(map[string]interface{})
	if arguments == nil {
		arguments = make(map[string]interface{})
	}

	result, err := t.client.CallTool(ctx, toolName, arguments)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("工具 '%s' 执行结果:\n%s", toolName, result), nil
}

// listResources 列出资源
func (t *MCPTool) listResources(ctx context.Context) (string, error) {
	resources, err := t.client.ListResources(ctx)
	if err != nil {
		return "", err
	}

	if len(resources) == 0 {
		return "没有找到可用的资源", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个资源:\n", len(resources)))
	for _, r := range resources {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", r.URI, r.Name))
	}
	return sb.String(), nil
}

// readResource 读取资源
func (t *MCPTool) readResource(ctx context.Context, args map[string]interface{}) (string, error) {
	uri, ok := args["uri"].(string)
	if !ok || uri == "" {
		return "", fmt.Errorf("必须指定 uri 参数")
	}

	content, err := t.client.ReadResource(ctx, uri)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("资源 '%s' 内容:\n%s", uri, content), nil
}

// listPrompts 列出提示词
func (t *MCPTool) listPrompts(ctx context.Context) (string, error) {
	prompts, err := t.client.ListPrompts(ctx)
	if err != nil {
		return "", err
	}

	if len(prompts) == 0 {
		return "没有找到可用的提示词", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个提示词:\n", len(prompts)))
	for _, p := range prompts {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", p.Name, p.Description))
	}
	return sb.String(), nil
}

// getPrompt 获取提示词
func (t *MCPTool) getPrompt(ctx context.Context, args map[string]interface{}) (string, error) {
	promptName, ok := args["prompt_name"].(string)
	if !ok || promptName == "" {
		return "", fmt.Errorf("必须指定 prompt_name 参数")
	}

	promptArgs := make(map[string]string)
	if pa, ok := args["prompt_arguments"].(map[string]interface{}); ok {
		for k, v := range pa {
			if s, ok := v.(string); ok {
				promptArgs[k] = s
			}
		}
	}

	messages, err := t.client.GetPrompt(ctx, promptName, promptArgs)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("提示词 '%s':\n", promptName))
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content.Text))
	}
	return sb.String(), nil
}

// GetExpandedTools 获取展开的工具列表
//
// 将 MCP 服务器的每个工具包装成独立的 Tool 对象
func (t *MCPTool) GetExpandedTools() []tools.Tool {
	if !t.autoExpand {
		return nil
	}

	ctx := context.Background()
	if err := t.ensureInitialized(ctx); err != nil {
		return nil
	}

	expandedTools := make([]tools.Tool, 0, len(t.availableTools))
	for _, toolInfo := range t.availableTools {
		wrapped := NewMCPWrappedTool(t, toolInfo, t.prefix)
		expandedTools = append(expandedTools, wrapped)
	}

	return expandedTools
}

// Close 关闭 MCP 工具
func (t *MCPTool) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.transport != nil {
		return t.transport.Close()
	}
	return nil
}

// 确保实现 Tool 接口
var _ tools.Tool = (*MCPTool)(nil)
