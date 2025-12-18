package llm

import (
	"context"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// GenerateStream 生成响应（流式）
func (c *OpenAIClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 构建请求
		chatReq := c.buildChatRequest(req)
		chatReq.Stream = true

		// 创建流
		stream, err := c.client.CreateChatCompletionStream(ctx, chatReq)
		if err != nil {
			errChan <- mapOpenAIError(err)
			return
		}
		defer stream.Close()

		// 累积工具调用
		var toolCalls []message.ToolCall
		toolCallsMap := make(map[int]*message.ToolCall)

		// 读取流
		for {
			response, err := stream.Recv()
			if err != nil {
				// 检查是否是流结束
				if err.Error() == "EOF" {
					// 发送完成标记
					chunkChan <- StreamChunk{
						Done:       true,
						ToolCalls:  toolCalls,
						TokenUsage: nil, // 流式响应通常不包含 token 统计
					}
					return
				}
				errChan <- mapOpenAIError(err)
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]

			// 处理文本内容
			if choice.Delta.Content != "" {
				chunkChan <- StreamChunk{
					Content: choice.Delta.Content,
				}
			}

			// 处理工具调用增量
			for _, tc := range choice.Delta.ToolCalls {
				idx := tc.Index
				if idx == nil {
					continue
				}

				// 获取或创建工具调用
				if _, exists := toolCallsMap[*idx]; !exists {
					toolCallsMap[*idx] = &message.ToolCall{
						ID:        tc.ID,
						Arguments: make(map[string]interface{}),
					}
				}
				toolCall := toolCallsMap[*idx]

				// 更新工具调用信息
				if tc.ID != "" {
					toolCall.ID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCall.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					// 累积参数 JSON 字符串
					// 注意：这里简化处理，实际需要累积后解析
					var args map[string]interface{}
					if err := unmarshalJSON([]byte(tc.Function.Arguments), &args); err == nil {
						for k, v := range args {
							toolCall.Arguments[k] = v
						}
					}
				}
			}

			// 检查完成原因
			if choice.FinishReason != "" {
				// 收集所有工具调用
				for i := 0; i < len(toolCallsMap); i++ {
					if tc, ok := toolCallsMap[i]; ok {
						toolCalls = append(toolCalls, *tc)
					}
				}

				chunkChan <- StreamChunk{
					Done:         true,
					FinishReason: string(choice.FinishReason),
					ToolCalls:    toolCalls,
				}
				return
			}
		}
	}()

	return chunkChan, errChan
}
