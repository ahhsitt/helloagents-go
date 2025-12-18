// Package main 演示 ReActAgent 的工具调用能力
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/tools"
	"github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func main() {
	// 从环境变量获取 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	// 可选：自定义 API 基地址
	baseURL := os.Getenv("OPENAI_BASE_URL")

	// 创建 OpenAI 客户端
	var provider llm.Provider
	var err error
	if baseURL != "" {
		provider, err = llm.NewOpenAI(
			llm.WithAPIKey(apiKey),
			llm.WithBaseURL(baseURL),
		)
	} else {
		provider, err = llm.NewOpenAI(llm.WithAPIKey(apiKey))
	}
	if err != nil {
		fmt.Printf("Error creating LLM provider: %v\n", err)
		os.Exit(1)
	}
	defer provider.Close()

	// 创建工具注册表
	registry := tools.NewRegistry()

	// 注册内置工具
	registry.MustRegister(builtin.NewCalculator())
	registry.MustRegister(builtin.NewTerminal(
		builtin.WithAllowedCommands([]string{
			"ls", "pwd", "date", "whoami", "echo", "cat",
		}),
	))

	// 创建 ReActAgent
	agent, err := agents.NewReAct(
		provider,
		registry,
		agents.WithName("ReActAssistant"),
		agents.WithMaxIterations(5),
	)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("ReActAgent Demo")
	fmt.Println("================")
	fmt.Println("This agent can use tools to help answer questions.")
	fmt.Println("Available tools:")
	for _, t := range registry.All() {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}
	fmt.Println()
	fmt.Println("Try asking:")
	fmt.Println("  - 123 乘以 456 等于多少？")
	fmt.Println("  - What is 15% of 250?")
	fmt.Println("  - 今天是什么日期？")
	fmt.Println()
	fmt.Println("Type 'quit' to exit, 'clear' to clear history.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		case "clear":
			agent.ClearHistory()
			fmt.Println("Conversation history cleared.")
			continue
		}

		ctx := context.Background()
		output, err := agent.Run(ctx, agents.Input{Query: input})

		// 显示推理步骤
		if len(output.Steps) > 0 {
			fmt.Println("\n--- Reasoning Steps ---")
			for i, step := range output.Steps {
				switch step.Type {
				case agents.StepTypeThought:
					fmt.Printf("[%d] Thought: %s\n", i+1, step.Content)
				case agents.StepTypeAction:
					fmt.Printf("[%d] Action: %s(%v)\n", i+1, step.ToolName, step.ToolArgs)
				case agents.StepTypeObservation:
					fmt.Printf("[%d] Observation: %s\n", i+1, step.ToolResult)
				}
			}
			fmt.Println("-----------------------")
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Assistant: %s\n", output.Response)
		fmt.Printf("(Tokens: prompt=%d, completion=%d, total=%d, duration=%v)\n",
			output.TokenUsage.PromptTokens,
			output.TokenUsage.CompletionTokens,
			output.TokenUsage.TotalTokens,
			output.Duration)
		fmt.Println()
	}
}
