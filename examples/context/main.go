// Example: Basic usage of the context engineering module
package main

import (
	"context"
	"fmt"
	"time"

	agentctx "github.com/easyops/helloagents-go/pkg/context"
	"github.com/easyops/helloagents-go/pkg/core/message"
)

func main() {
	// Create a GSSC context builder with default configuration
	builder := agentctx.NewGSSCBuilder()

	// Prepare input with conversation history
	history := []message.Message{
		{Role: message.RoleUser, Content: "你好，我想了解一下 Go 语言", Timestamp: time.Now().Add(-10 * time.Minute)},
		{Role: message.RoleAssistant, Content: "你好！Go 是一门由 Google 开发的编程语言...", Timestamp: time.Now().Add(-9 * time.Minute)},
		{Role: message.RoleUser, Content: "Go 的并发模型是什么？", Timestamp: time.Now().Add(-5 * time.Minute)},
		{Role: message.RoleAssistant, Content: "Go 使用 goroutine 和 channel 来实现并发...", Timestamp: time.Now().Add(-4 * time.Minute)},
	}

	input := &agentctx.BuildInput{
		Query:              "如何在 Go 中使用 channel？",
		SystemInstructions: "你是一个专业的 Go 语言专家，擅长解释并发编程概念。",
		History:            history,
	}

	// Build the structured context
	ctx := context.Background()
	result, err := builder.Build(ctx, input)
	if err != nil {
		fmt.Printf("Error building context: %v\n", err)
		return
	}

	fmt.Println("=== Structured Context ===")
	fmt.Println(result)
	fmt.Println()

	// You can also build messages directly for LLM
	messages, err := builder.BuildMessages(ctx, input)
	if err != nil {
		fmt.Printf("Error building messages: %v\n", err)
		return
	}

	fmt.Println("=== Messages for LLM ===")
	for i, msg := range messages {
		fmt.Printf("[%d] Role: %s\n", i, msg.Role)
		fmt.Printf("    Content (first 100 chars): %.100s...\n", msg.Content)
	}
}
