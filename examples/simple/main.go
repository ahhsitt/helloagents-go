// Package main 演示 SimpleAgent 的基本用法
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/core/llm"
)

func main() {
	// 从环境变量获取 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	// 可选：自定义 API 基地址（用于兼容 OpenAI API 的服务）
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

	// 创建 SimpleAgent
	agent, err := agents.NewSimple(
		provider,
		agents.WithName("Assistant"),
		agents.WithSystemPrompt("You are a helpful assistant. Please respond in the same language as the user's input."),
	)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SimpleAgent Demo")
	fmt.Println("================")
	fmt.Println("Type your message and press Enter. Type 'quit' to exit.")
	fmt.Println("Type 'stream' to toggle streaming mode.")
	fmt.Println("Type 'clear' to clear conversation history.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	streaming := false

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 命令处理
		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		case "stream":
			streaming = !streaming
			fmt.Printf("Streaming mode: %v\n", streaming)
			continue
		case "clear":
			agent.ClearHistory()
			fmt.Println("Conversation history cleared.")
			continue
		}

		ctx := context.Background()

		if streaming {
			// 流式响应
			fmt.Print("Assistant: ")
			chunks, errs := agent.RunStream(ctx, agents.Input{Query: input})

		streamLoop:
			for {
				select {
				case err := <-errs:
					if err != nil {
						fmt.Printf("\nError: %v\n", err)
					}
					break streamLoop
				case chunk, ok := <-chunks:
					if !ok {
						break streamLoop
					}
					if chunk.Content != "" {
						fmt.Print(chunk.Content)
					}
					if chunk.Done {
						fmt.Println()
						break streamLoop
					}
				}
			}
		} else {
			// 非流式响应
			output, err := agent.Run(ctx, agents.Input{Query: input})
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
		}
		fmt.Println()
	}
}
