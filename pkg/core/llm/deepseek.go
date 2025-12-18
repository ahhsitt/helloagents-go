package llm

import (
	"context"
	"fmt"

	"github.com/easyops/helloagents-go/pkg/core/errors"
	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekClient DeepSeek 客户端
//
// DeepSeek 提供 OpenAI 兼容的 API，基于 OpenAI SDK 实现。
type DeepSeekClient struct {
	client  *openai.Client
	options *Options
}

// NewDeepSeek 创建 DeepSeek 客户端
func NewDeepSeek(opts ...Option) (*DeepSeekClient, error) {
	options := DefaultOptions()
	options.BaseURL = "https://api.deepseek.com/v1"
	options.Model = "deepseek-chat"

	for _, opt := range opts {
		opt(options)
	}

	if options.APIKey == "" {
		return nil, errors.ErrInvalidAPIKey
	}

	config := openai.DefaultConfig(options.APIKey)
	config.BaseURL = options.BaseURL

	return &DeepSeekClient{
		client:  openai.NewClientWithConfig(config),
		options: options,
	}, nil
}

// Name 返回提供商名称
func (c *DeepSeekClient) Name() string {
	return "deepseek"
}

// Model 返回当前模型名称
func (c *DeepSeekClient) Model() string {
	return c.options.Model
}

// Close 关闭客户端连接
func (c *DeepSeekClient) Close() error {
	return nil
}

// Generate 生成响应（非流式）
func (c *DeepSeekClient) Generate(ctx context.Context, req Request) (Response, error) {
	chatReq := buildOpenAIChatRequest(req, c.options.Model)

	var resp openai.ChatCompletionResponse
	var err error

	err = retry(ctx, c.options.MaxRetries, c.options.RetryDelay, func() error {
		resp, err = c.client.CreateChatCompletion(ctx, chatReq)
		return mapOpenAIError(err)
	})

	if err != nil {
		return Response{}, err
	}

	return parseOpenAIResponse(resp), nil
}

// GenerateStream 生成响应（流式）
func (c *DeepSeekClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	return streamOpenAIResponse(ctx, c.client, req, c.options)
}

// Embed 生成文本嵌入向量
//
// DeepSeek 目前不支持嵌入 API，返回错误。
func (c *DeepSeekClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("deepseek does not support embedding API")
}

// compile-time interface check
var _ Provider = (*DeepSeekClient)(nil)
