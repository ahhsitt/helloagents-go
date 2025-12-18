package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// QwenClient 通义千问客户端
type QwenClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// QwenOption 通义千问客户端选项
type QwenOption func(*QwenClient)

// WithQwenAPIKey 设置 API Key
func WithQwenAPIKey(apiKey string) QwenOption {
	return func(c *QwenClient) {
		c.apiKey = apiKey
	}
}

// WithQwenModel 设置模型
func WithQwenModel(model string) QwenOption {
	return func(c *QwenClient) {
		c.model = model
	}
}

// WithQwenBaseURL 设置基础 URL
func WithQwenBaseURL(url string) QwenOption {
	return func(c *QwenClient) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithQwenHTTPClient 设置 HTTP 客户端
func WithQwenHTTPClient(client *http.Client) QwenOption {
	return func(c *QwenClient) {
		c.httpClient = client
	}
}

// NewQwenClient 创建通义千问客户端
func NewQwenClient(opts ...QwenOption) *QwenClient {
	c := &QwenClient{
		baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		model:   "qwen-turbo",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// qwenRequest 通义千问请求结构（OpenAI 兼容格式）
type qwenRequest struct {
	Model       string        `json:"model"`
	Messages    []qwenMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	Tools       []qwenTool    `json:"tools,omitempty"`
}

// qwenMessage 通义千问消息
type qwenMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []qwenToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// qwenTool 通义千问工具
type qwenTool struct {
	Type     string       `json:"type"`
	Function qwenFunction `json:"function"`
}

// qwenFunction 通义千问函数
type qwenFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// qwenToolCall 通义千问工具调用
type qwenToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function qwenFunctionCall   `json:"function"`
}

// qwenFunctionCall 通义千问函数调用
type qwenFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// qwenResponse 通义千问响应
type qwenResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      qwenMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// qwenStreamResponse 通义千问流式响应
type qwenStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string         `json:"role,omitempty"`
			Content   string         `json:"content,omitempty"`
			ToolCalls []qwenToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Generate 生成响应（非流式）
func (c *QwenClient) Generate(ctx context.Context, req Request) (Response, error) {
	qwenReq := c.buildRequest(req, false)

	body, err := json.Marshal(qwenReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Response{}, fmt.Errorf("qwen error: %s - %s", resp.Status, string(bodyBytes))
	}

	var qwenResp qwenResponse
	if err := json.NewDecoder(resp.Body).Decode(&qwenResp); err != nil {
		return Response{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(qwenResp), nil
}

// GenerateStream 生成响应（流式）
func (c *QwenClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		qwenReq := c.buildRequest(req, true)

		body, err := json.Marshal(qwenReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("qwen error: %s - %s", resp.Status, string(bodyBytes))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		var accumulatedToolCalls []message.ToolCall

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp qwenStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]
			chunk := StreamChunk{
				Content: choice.Delta.Content,
			}

			// 累积工具调用
			if len(choice.Delta.ToolCalls) > 0 {
				for _, tc := range choice.Delta.ToolCalls {
					args := make(map[string]interface{})
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
					accumulatedToolCalls = append(accumulatedToolCalls, message.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: args,
					})
				}
			}

			if choice.FinishReason != "" {
				chunk.Done = true
				chunk.FinishReason = choice.FinishReason
				if len(accumulatedToolCalls) > 0 {
					chunk.ToolCalls = accumulatedToolCalls
					chunk.FinishReason = "tool_calls"
				}
				if streamResp.Usage != nil {
					chunk.TokenUsage = &message.TokenUsage{
						PromptTokens:     streamResp.Usage.PromptTokens,
						CompletionTokens: streamResp.Usage.CompletionTokens,
						TotalTokens:      streamResp.Usage.TotalTokens,
					}
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

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("stream read error: %w", err)
		}
	}()

	return chunkCh, errCh
}

// Embed 生成文本嵌入向量
func (c *QwenClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": "text-embedding-v2",
		"input": texts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qwen embed error: %s - %s", resp.Status, string(bodyBytes))
	}

	var embedResp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode embed response: %w", err)
	}

	results := make([][]float32, len(texts))
	for _, d := range embedResp.Data {
		if d.Index < len(results) {
			results[d.Index] = d.Embedding
		}
	}

	return results, nil
}

// Name 返回提供商名称
func (c *QwenClient) Name() string {
	return "qwen"
}

// Model 返回当前模型名称
func (c *QwenClient) Model() string {
	return c.model
}

// Close 关闭客户端连接
func (c *QwenClient) Close() error {
	return nil
}

// buildRequest 构建请求
func (c *QwenClient) buildRequest(req Request, stream bool) qwenRequest {
	qwenReq := qwenRequest{
		Model:       c.model,
		Messages:    make([]qwenMessage, len(req.Messages)),
		Stream:      stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}

	for i, msg := range req.Messages {
		qwenReq.Messages[i] = qwenMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
		if msg.ToolCallID != "" {
			qwenReq.Messages[i].ToolCallID = msg.ToolCallID
		}
	}

	if len(req.Tools) > 0 {
		qwenReq.Tools = make([]qwenTool, len(req.Tools))
		for i, tool := range req.Tools {
			qwenReq.Tools[i] = qwenTool{
				Type: "function",
				Function: qwenFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	return qwenReq
}

// convertResponse 转换响应
func (c *QwenClient) convertResponse(resp qwenResponse) Response {
	if len(resp.Choices) == 0 {
		return Response{}
	}

	choice := resp.Choices[0]
	result := Response{
		ID:      resp.ID,
		Content: choice.Message.Content,
		TokenUsage: message.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		FinishReason: choice.FinishReason,
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]message.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			args := make(map[string]interface{})
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			result.ToolCalls[i] = message.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			}
		}
		result.FinishReason = "tool_calls"
	}

	return result
}

// compile-time interface check
var _ Provider = (*QwenClient)(nil)
