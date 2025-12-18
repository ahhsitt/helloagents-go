package llm

import (
	"context"
	"fmt"

	"github.com/easyops/helloagents-go/pkg/core/errors"
	"github.com/easyops/helloagents-go/pkg/core/message"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient OpenAI LLM 客户端
type OpenAIClient struct {
	client  *openai.Client
	options *Options
}

// NewOpenAI 创建 OpenAI 客户端
func NewOpenAI(opts ...Option) (*OpenAIClient, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.APIKey == "" {
		return nil, errors.ErrInvalidAPIKey
	}
	if options.Model == "" {
		options.Model = "gpt-4o"
	}
	if options.EmbeddingModel == "" {
		options.EmbeddingModel = "text-embedding-3-small"
	}

	config := openai.DefaultConfig(options.APIKey)
	if options.BaseURL != "" {
		config.BaseURL = options.BaseURL
	}

	return &OpenAIClient{
		client:  openai.NewClientWithConfig(config),
		options: options,
	}, nil
}

// Name 返回提供商名称
func (c *OpenAIClient) Name() string {
	return "openai"
}

// Model 返回当前模型名称
func (c *OpenAIClient) Model() string {
	return c.options.Model
}

// Close 关闭客户端连接
func (c *OpenAIClient) Close() error {
	return nil
}

// Generate 生成响应（非流式）
func (c *OpenAIClient) Generate(ctx context.Context, req Request) (Response, error) {
	// 构建 OpenAI 请求
	chatReq := c.buildChatRequest(req)

	// 执行请求（带重试）
	var resp openai.ChatCompletionResponse
	var err error

	err = retry(ctx, c.options.MaxRetries, c.options.RetryDelay, func() error {
		resp, err = c.client.CreateChatCompletion(ctx, chatReq)
		return mapOpenAIError(err)
	})

	if err != nil {
		return Response{}, err
	}

	return c.parseResponse(resp), nil
}

// buildChatRequest 构建 OpenAI 请求
func (c *OpenAIClient) buildChatRequest(req Request) openai.ChatCompletionRequest {
	chatReq := openai.ChatCompletionRequest{
		Model:    c.options.Model,
		Messages: c.convertMessages(req.Messages),
	}

	// 设置温度
	if req.Temperature != nil {
		chatReq.Temperature = float32(*req.Temperature)
	} else {
		chatReq.Temperature = float32(c.options.Temperature)
	}

	// 设置最大 token
	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	} else {
		chatReq.MaxTokens = c.options.MaxTokens
	}

	// 设置停止序列
	if len(req.Stop) > 0 {
		chatReq.Stop = req.Stop
	}

	// 设置工具
	if len(req.Tools) > 0 {
		chatReq.Tools = c.convertTools(req.Tools)
		if req.ToolChoice != nil {
			switch v := req.ToolChoice.(type) {
			case string:
				chatReq.ToolChoice = v
			default:
				chatReq.ToolChoice = v
			}
		}
	}

	return chatReq
}

// convertMessages 转换消息格式
func (c *OpenAIClient) convertMessages(msgs []message.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, msg := range msgs {
		chatMsg := openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// 处理工具调用
		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				argsJSON, _ := marshalJSON(tc.Arguments)
				chatMsg.ToolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}

		// 处理工具调用 ID
		if msg.ToolCallID != "" {
			chatMsg.ToolCallID = msg.ToolCallID
		}

		// 处理名称
		if msg.Name != "" {
			chatMsg.Name = msg.Name
		}

		result = append(result, chatMsg)
	}
	return result
}

// convertTools 转换工具格式
func (c *OpenAIClient) convertTools(tools []ToolDefinition) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

// parseResponse 解析响应
func (c *OpenAIClient) parseResponse(resp openai.ChatCompletionResponse) Response {
	if len(resp.Choices) == 0 {
		return Response{}
	}

	choice := resp.Choices[0]
	result := Response{
		ID:           resp.ID,
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		TokenUsage: message.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// 解析工具调用
	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]message.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			args := make(map[string]interface{})
			_ = unmarshalJSON([]byte(tc.Function.Arguments), &args)
			result.ToolCalls[i] = message.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
	}

	return result
}

// Embed 生成文本嵌入向量
func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(c.options.EmbeddingModel),
	}

	var resp openai.EmbeddingResponse
	var err error

	err = retry(ctx, c.options.MaxRetries, c.options.RetryDelay, func() error {
		resp, err = c.client.CreateEmbeddings(ctx, req)
		return mapOpenAIError(err)
	})

	if err != nil {
		return nil, err
	}

	result := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		result[i] = data.Embedding
	}

	return result, nil
}

// mapOpenAIError 映射 OpenAI 错误到框架错误
func mapOpenAIError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否是 OpenAI API 错误
	apiErr, ok := err.(*openai.APIError)
	if !ok {
		return errors.WrapError(err, "openai request failed")
	}

	switch apiErr.HTTPStatusCode {
	case 401:
		return errors.ErrInvalidAPIKey
	case 429:
		return errors.ErrRateLimited
	case 500, 502, 503:
		return errors.ErrProviderUnavailable
	default:
		return fmt.Errorf("openai error (code=%d): %w", apiErr.HTTPStatusCode, err)
	}
}

// buildOpenAIChatRequest 构建 OpenAI 格式的请求（供其他兼容客户端使用）
func buildOpenAIChatRequest(req Request, model string) openai.ChatCompletionRequest {
	chatReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: convertMessagesToOpenAI(req.Messages),
	}

	if req.Temperature != nil {
		chatReq.Temperature = float32(*req.Temperature)
	}

	if req.MaxTokens != nil {
		chatReq.MaxTokens = *req.MaxTokens
	}

	if len(req.Stop) > 0 {
		chatReq.Stop = req.Stop
	}

	if len(req.Tools) > 0 {
		chatReq.Tools = convertToolsToOpenAI(req.Tools)
		if req.ToolChoice != nil {
			chatReq.ToolChoice = req.ToolChoice
		}
	}

	return chatReq
}

// convertMessagesToOpenAI 转换消息格式到 OpenAI 格式
func convertMessagesToOpenAI(msgs []message.Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, msg := range msgs {
		chatMsg := openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				argsJSON, _ := marshalJSON(tc.Arguments)
				chatMsg.ToolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}

		if msg.ToolCallID != "" {
			chatMsg.ToolCallID = msg.ToolCallID
		}

		if msg.Name != "" {
			chatMsg.Name = msg.Name
		}

		result = append(result, chatMsg)
	}
	return result
}

// convertToolsToOpenAI 转换工具格式到 OpenAI 格式
func convertToolsToOpenAI(tools []ToolDefinition) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

// parseOpenAIResponse 解析 OpenAI 响应
func parseOpenAIResponse(resp openai.ChatCompletionResponse) Response {
	if len(resp.Choices) == 0 {
		return Response{}
	}

	choice := resp.Choices[0]
	result := Response{
		ID:           resp.ID,
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		TokenUsage: message.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]message.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			args := make(map[string]interface{})
			_ = unmarshalJSON([]byte(tc.Function.Arguments), &args)
			result.ToolCalls[i] = message.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
	}

	return result
}

// streamOpenAIResponse 流式处理 OpenAI 响应（供兼容客户端使用）
func streamOpenAIResponse(ctx context.Context, client *openai.Client, req Request, options *Options) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		chatReq := buildOpenAIChatRequest(req, options.Model)
		chatReq.Stream = true

		stream, err := client.CreateChatCompletionStream(ctx, chatReq)
		if err != nil {
			errCh <- mapOpenAIError(err)
			return
		}
		defer stream.Close()

		var accumulatedToolCalls []message.ToolCall

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				errCh <- mapOpenAIError(err)
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]
			chunk := StreamChunk{
				Content: choice.Delta.Content,
			}

			// 累积工具调用
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					if tc.Function.Name != "" {
						accumulatedToolCalls = append(accumulatedToolCalls, message.ToolCall{
							ID:        tc.ID,
							Name:      tc.Function.Name,
							Arguments: map[string]interface{}{},
						})
					}
				}
			}

			if choice.FinishReason != "" {
				chunk.Done = true
				chunk.FinishReason = string(choice.FinishReason)
				if len(accumulatedToolCalls) > 0 {
					chunk.ToolCalls = accumulatedToolCalls
				}
			}

			select {
			case chunkCh <- chunk:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}

			if chunk.Done {
				break
			}
		}
	}()

	return chunkCh, errCh
}
