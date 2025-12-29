// MCP 客户端示例
//
// 演示如何使用 MCP 客户端连接到服务器并调用工具。
// 运行方式:
//   1. 先在一个终端运行服务器: go run examples/mcp/server/main.go
//   2. 修改此文件连接到服务器，或使用内置服务器测试
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建内存传输（用于演示，实际使用时替换为 StdioTransport 或 HTTPTransport）
	// 创建一个简单的模拟服务器响应
	transport := mcp.NewMemoryTransport(createMockHandler())

	// 创建客户端
	client := mcp.NewClient(transport)
	defer client.Close()

	fmt.Println("Connecting to MCP server...")

	// 初始化连接
	if err := client.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Println("Connected!")
	if info := client.ServerInfo(); info != nil {
		fmt.Printf("Server: %s v%s\n\n", info.Name, info.Version)
	}

	// 列出可用工具
	fmt.Println("=== Available Tools ===")
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	for _, tool := range tools {
		fmt.Printf("  %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// 调用工具
	fmt.Println("=== Calling Tools ===")

	// 调用加法工具
	result, err := client.CallTool(ctx, "add", map[string]interface{}{
		"a": 10,
		"b": 5,
	})
	if err != nil {
		log.Printf("Failed to call add: %v", err)
	} else {
		fmt.Printf("add(10, 5) = %s\n", result)
	}

	// 调用问候工具
	result, err = client.CallTool(ctx, "greet", map[string]interface{}{
		"name": "Alice",
	})
	if err != nil {
		log.Printf("Failed to call greet: %v", err)
	} else {
		fmt.Printf("greet('Alice') = %s\n", result)
	}

	fmt.Println("\nDemo completed!")
}

// createMockHandler 创建模拟服务器处理函数
func createMockHandler() func(request []byte) ([]byte, error) {
	return func(request []byte) ([]byte, error) {
		// 解析请求
		var req mcp.JSONRPCRequest
		if err := json.Unmarshal(request, &req); err != nil {
			resp := &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPCVersion,
				Error:   &mcp.JSONRPCError{Code: -32700, Message: "Parse error"},
			}
			return json.Marshal(resp)
		}

		// 根据方法返回响应
		switch req.Method {
		case mcp.MethodInitialize:
			result, _ := json.Marshal(mcp.InitializeResult{
				ProtocolVersion: mcp.MCPVersion,
				ServerInfo:      mcp.Implementation{Name: "mock-server", Version: "1.0.0"},
			})
			resp := &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPCVersion,
				ID:      req.ID,
				Result:  result,
			}
			return json.Marshal(resp)

		case mcp.MethodInitialized:
			return nil, nil

		case mcp.MethodListTools:
			result, _ := json.Marshal(mcp.ListToolsResult{
				Tools: []mcp.Tool{
					{Name: "add", Description: "Add two numbers"},
					{Name: "greet", Description: "Generate a greeting"},
				},
			})
			resp := &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPCVersion,
				ID:      req.ID,
				Result:  result,
			}
			return json.Marshal(resp)

		case mcp.MethodCallTool:
			var params mcp.CallToolParams
			json.Unmarshal(req.Params, &params)

			var text string
			switch params.Name {
			case "add":
				a, _ := params.Arguments["a"].(float64)
				b, _ := params.Arguments["b"].(float64)
				text = fmt.Sprintf("%.0f", a+b)
			case "greet":
				name, _ := params.Arguments["name"].(string)
				if name == "" {
					name = "World"
				}
				text = fmt.Sprintf("Hello, %s!", name)
			default:
				text = "Unknown tool"
			}

			result, _ := json.Marshal(mcp.CallToolResult{
				Content: []mcp.Content{{Type: "text", Text: text}},
			})
			resp := &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPCVersion,
				ID:      req.ID,
				Result:  result,
			}
			return json.Marshal(resp)

		default:
			resp := &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPCVersion,
				ID:      req.ID,
				Error:   &mcp.JSONRPCError{Code: -32601, Message: "Method not found"},
			}
			return json.Marshal(resp)
		}
	}
}
