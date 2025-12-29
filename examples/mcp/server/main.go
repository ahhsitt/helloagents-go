// MCP 服务器示例
//
// 演示如何创建一个 MCP 服务器，注册工具并运行。
// 运行方式: go run examples/mcp/server/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func main() {
	// 创建 MCP 服务器
	server := mcp.NewServer("example-server", "Example MCP Server with calculator and greeting tools")

	// 添加计算器工具
	server.AddTool(mcp.ServerTool{
		Name:        "calculator",
		Description: "Perform mathematical calculations",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Operation to perform: add, subtract, multiply, divide",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
				},
				"a": map[string]interface{}{
					"type":        "number",
					"description": "First operand",
				},
				"b": map[string]interface{}{
					"type":        "number",
					"description": "Second operand",
				},
			},
			"required": []string{"operation", "a", "b"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			op := args["operation"].(string)
			a := args["a"].(float64)
			b := args["b"].(float64)

			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return "", fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return "", fmt.Errorf("unknown operation: %s", op)
			}

			return fmt.Sprintf("%.2f", result), nil
		},
	})

	// 添加问候工具
	server.AddTool(mcp.ServerTool{
		Name:        "greet",
		Description: "Generate a friendly greeting",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name to greet",
					"default":     "World",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Language for greeting",
					"enum":        []string{"en", "zh", "ja"},
					"default":     "en",
				},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			name := "World"
			if n, ok := args["name"].(string); ok && n != "" {
				name = n
			}

			lang := "en"
			if l, ok := args["language"].(string); ok && l != "" {
				lang = l
			}

			switch lang {
			case "zh":
				return fmt.Sprintf("你好，%s！欢迎使用 HelloAgents MCP！", name), nil
			case "ja":
				return fmt.Sprintf("こんにちは、%sさん！HelloAgents MCP へようこそ！", name), nil
			default:
				return fmt.Sprintf("Hello, %s! Welcome to HelloAgents MCP!", name), nil
			}
		},
	})

	// 添加系统信息资源
	server.AddResource(mcp.ServerResource{
		URI:         "info://system",
		Name:        "System Info",
		Description: "Get system information",
		MimeType:    "application/json",
		Handler: func(ctx context.Context) (string, error) {
			return `{"server": "example-server", "version": "1.0.0", "protocol": "MCP"}`, nil
		},
	})

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// 打印服务器信息
	fmt.Fprintln(os.Stderr, "🚀 Starting MCP Server...")
	fmt.Fprintln(os.Stderr, "📝 Server: example-server")
	fmt.Fprintln(os.Stderr, "🔌 Protocol: MCP (stdio)")
	fmt.Fprintln(os.Stderr, "🛠️  Tools: calculator, greet")
	fmt.Fprintln(os.Stderr, "📁 Resources: info://system")
	fmt.Fprintln(os.Stderr, "")

	// 运行服务器（Stdio 模式）
	if err := server.Run(ctx, os.Stdin, os.Stdout); err != nil {
		if err != context.Canceled {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Fprintln(os.Stderr, "👋 Server stopped")
}
