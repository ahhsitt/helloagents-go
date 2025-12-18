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

// OllamaClient Ollama 客户端
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// OllamaOption Ollama 客户端选项
type OllamaOption func(*OllamaClient)

// WithOllamaBaseURL 设置基础 URL
func WithOllamaBaseURL(url string) OllamaOption {
	return func(c *OllamaClient) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithOllamaModel 设置模型名称
func WithOllamaModel(model string) OllamaOption {
	return func(c *OllamaClient) {
		c.model = model
	}
}

// WithOllamaHTTPClient 设置 HTTP 客户端
func WithOllamaHTTPClient(client *http.Client) OllamaOption {
	return func(c *OllamaClient) {
		c.httpClient = client
	}
}

// NewOllamaClient 创建 Ollama 客户端
func NewOllamaClient(opts ...OllamaOption) *OllamaClient {
	c := &OllamaClient{
		baseURL: "http://localhost:11434",
		model:   "llama3.2",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ollamaRequest Ollama 请求结构
type ollamaRequest struct {
	Model    string             `json:"model"`
	Messages []ollamaMessage    `json:"messages"`
	Stream   bool               `json:"stream"`
	Options  *ollamaOptions     `json:"options,omitempty"`
	Tools    []ollamaToolDef    `json:"tools,omitempty"`
}

// ollamaMessage Ollama 消息
type ollamaMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []ollamaToolCall     `json:"tool_calls,omitempty"`
}

// ollamaToolDef Ollama 工具定义
type ollamaToolDef struct {
	Type     string            `json:"type"`
	Function ollamaFunctionDef `json:"function"`
}

// ollamaFunctionDef Ollama 函数定义
type ollamaFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ollamaToolCall Ollama 工具调用
type ollamaToolCall struct {
	Function ollamaFunctionCall `json:"function"`
}

// ollamaFunctionCall Ollama 函数调用
type ollamaFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ollamaOptions Ollama 选项
type ollamaOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// ollamaResponse Ollama 响应
type ollamaResponse struct {
	Model              string          `json:"model"`
	Message            ollamaMessage   `json:"message"`
	Done               bool            `json:"done"`
	DoneReason         string          `json:"done_reason"`
	PromptEvalCount    int             `json:"prompt_eval_count"`
	EvalCount          int             `json:"eval_count"`
}

// Generate 生成响应（非流式）
func (c *OllamaClient) Generate(ctx context.Context, req Request) (Response, error) {
	ollamaReq := c.buildRequest(req, false)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Response{}, fmt.Errorf("ollama error: %s - %s", resp.Status, string(bodyBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return Response{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(ollamaResp), nil
}

// GenerateStream 生成响应（流式）
func (c *OllamaClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		ollamaReq := c.buildRequest(req, true)

		body, err := json.Marshal(ollamaReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("failed to create request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("ollama error: %s - %s", resp.Status, string(bodyBytes))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		var totalPromptTokens, totalCompletionTokens int

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var streamResp ollamaResponse
			if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
				errCh <- fmt.Errorf("failed to decode stream response: %w", err)
				return
			}

			totalPromptTokens = streamResp.PromptEvalCount
			totalCompletionTokens += streamResp.EvalCount

			chunk := StreamChunk{
				Content: streamResp.Message.Content,
				Done:    streamResp.Done,
			}

			if streamResp.Done {
				chunk.FinishReason = c.mapFinishReason(streamResp.DoneReason)
				chunk.TokenUsage = &message.TokenUsage{
					PromptTokens:     totalPromptTokens,
					CompletionTokens: totalCompletionTokens,
					TotalTokens:      totalPromptTokens + totalCompletionTokens,
				}

				// 处理工具调用
				if len(streamResp.Message.ToolCalls) > 0 {
					chunk.ToolCalls = c.convertToolCalls(streamResp.Message.ToolCalls)
				}
			}

			select {
			case chunkCh <- chunk:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}

			if streamResp.Done {
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
func (c *OllamaClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		reqBody := map[string]string{
			"model":  c.model,
			"prompt": text,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal embed request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create embed request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send embed request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("ollama embed error: %s - %s", resp.Status, string(bodyBytes))
		}

		var embedResp struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode embed response: %w", err)
		}
		resp.Body.Close()

		results[i] = embedResp.Embedding
	}

	return results, nil
}

// Name 返回提供商名称
func (c *OllamaClient) Name() string {
	return "ollama"
}

// Model 返回当前模型名称
func (c *OllamaClient) Model() string {
	return c.model
}

// Close 关闭客户端连接
func (c *OllamaClient) Close() error {
	return nil
}

// buildRequest 构建 Ollama 请求
func (c *OllamaClient) buildRequest(req Request, stream bool) ollamaRequest {
	ollamaReq := ollamaRequest{
		Model:    c.model,
		Messages: make([]ollamaMessage, len(req.Messages)),
		Stream:   stream,
	}

	// 转换消息
	for i, msg := range req.Messages {
		ollamaReq.Messages[i] = ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 设置选项
	if req.Temperature != nil || req.MaxTokens != nil || req.TopP != nil || len(req.Stop) > 0 {
		ollamaReq.Options = &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
			TopP:        req.TopP,
			Stop:        req.Stop,
		}
	}

	// 转换工具
	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]ollamaToolDef, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = ollamaToolDef{
				Type: "function",
				Function: ollamaFunctionDef{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	return ollamaReq
}

// convertResponse 转换 Ollama 响应
func (c *OllamaClient) convertResponse(resp ollamaResponse) Response {
	result := Response{
		Content: resp.Message.Content,
		TokenUsage: message.TokenUsage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		FinishReason: c.mapFinishReason(resp.DoneReason),
	}

	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = c.convertToolCalls(resp.Message.ToolCalls)
		result.FinishReason = "tool_calls"
	}

	return result
}

// convertToolCalls 转换工具调用
func (c *OllamaClient) convertToolCalls(calls []ollamaToolCall) []message.ToolCall {
	result := make([]message.ToolCall, len(calls))
	for i, call := range calls {
		args := make(map[string]interface{})
		_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		result[i] = message.ToolCall{
			ID:        fmt.Sprintf("call_%d", i),
			Name:      call.Function.Name,
			Arguments: args,
		}
	}
	return result
}

// mapFinishReason 映射结束原因
func (c *OllamaClient) mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "stop"
	case "length":
		return "length"
	default:
		return "stop"
	}
}

// compile-time interface check
var _ Provider = (*OllamaClient)(nil)
