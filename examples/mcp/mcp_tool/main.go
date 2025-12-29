// MCPTool 与 Agent 集成示例
//
// 演示如何将 MCPTool 作为 Agent 工具使用。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建 MCPTool（使用内置演示服务器）
	mcpTool := builtin.NewMCPTool(
		builtin.WithMCPName("calculator"),
		builtin.WithMCPAutoExpand(true),
	)
	defer mcpTool.Close()

	fmt.Println("=== MCPTool Demo ===\n")

	// 1. 列出可用工具
	fmt.Println("1. Listing available tools...")
	result, err := mcpTool.Execute(ctx, map[string]interface{}{
		"action": "list_tools",
	})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	fmt.Println(result)

	// 2. 调用加法工具
	fmt.Println("\n2. Calling 'add' tool...")
	result, err = mcpTool.Execute(ctx, map[string]interface{}{
		"action":    "call_tool",
		"tool_name": "add",
		"arguments": map[string]interface{}{
			"a": 15,
			"b": 27,
		},
	})
	if err != nil {
		log.Fatalf("Failed to call add: %v", err)
	}
	fmt.Println(result)

	// 3. 调用问候工具
	fmt.Println("\n3. Calling 'greet' tool...")
	result, err = mcpTool.Execute(ctx, map[string]interface{}{
		"action":    "call_tool",
		"tool_name": "greet",
		"arguments": map[string]interface{}{
			"name": "HelloAgents",
		},
	})
	if err != nil {
		log.Fatalf("Failed to call greet: %v", err)
	}
	fmt.Println(result)

	// 4. 获取展开的工具列表
	fmt.Println("\n4. Getting expanded tools...")
	expandedTools := mcpTool.GetExpandedTools()
	fmt.Printf("Found %d expanded tools:\n", len(expandedTools))
	for _, tool := range expandedTools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	// 5. 直接使用展开的工具
	if len(expandedTools) > 0 {
		fmt.Println("\n5. Using expanded tool directly...")
		for _, tool := range expandedTools {
			if tool.Name() == "calculator_multiply" {
				result, err := tool.Execute(ctx, map[string]interface{}{
					"a": 7,
					"b": 8,
				})
				if err != nil {
					log.Printf("Failed to execute: %v", err)
				} else {
					fmt.Printf("calculator_multiply(7, 8) = %s\n", result)
				}
				break
			}
		}
	}

	fmt.Println("\nDemo completed!")
}
