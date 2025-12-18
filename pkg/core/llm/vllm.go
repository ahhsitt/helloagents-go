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

// VLLMClient vLLM 客户端
//
// vLLM 提供 OpenAI 兼容的 API，用于自托管 LLM 服务。
type VLLMClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
	apiKey     string // 可���，某些部署可能需要
}

// VLLMOption vLLM 客户端选项
type VLLMOption func(*VLLMClient)

// WithVLLMBaseURL 设置基础 URL
func WithVLLMBaseURL(url string) VLLMOption {
	return func(c *VLLMClient) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithVLLMModel 设置模型
func WithVLLMModel(model string) VLLMOption {
	return func(c *VLLMClient) {
		c.model = model
	}
}

// WithVLLMAPIKey 设置 API Key（可选）
func WithVLLMAPIKey(apiKey string) VLLMOption {
	return func(c *VLLMClient) {
		c.apiKey = apiKey
	}
}

// WithVLLMHTTPClient 设置 HTTP 客户端
func WithVLLMHTTPClient(client *http.Client) VLLMOption {
	return func(c *VLLMClient) {
		c.httpClient = client
	}
}

// NewVLLMClient 创建 vLLM 客户端
func NewVLLMClient(opts ...VLLMOption) *VLLMClient {
	c := &VLLMClient{
		baseURL: "http://localhost:8000/v1",
		model:   "default",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// vllmRequest vLLM 请求结构（OpenAI 兼容格式）
type vllmRequest struct {
	Model       string        `json:"model"`
	Messages    []vllmMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// vllmMessage vLLM 消息
type vllmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// vllmResponse vLLM 响应
type vllmResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// vllmStreamResponse vLLM 流式响应
type vllmStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
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
func (c *VLLMClient) Generate(ctx context.Context, req Request) (Response, error) {
	vllmReq := c.buildRequest(req, false)

	body, err := json.Marshal(vllmReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return Response{}, fmt.Errorf("vllm error: %s - %s", resp.Status, string(bodyBytes))
	}

	var vllmResp vllmResponse
	if err := json.NewDecoder(resp.Body).Decode(&vllmResp); err != nil {
		return Response{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertResponse(vllmResp), nil
}

// GenerateStream 生成响应（流式）
func (c *VLLMClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error) {
	chunkCh := make(chan StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		vllmReq := c.buildRequest(req, true)

		body, err := json.Marshal(vllmReq)
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
		httpReq.Header.Set("Accept", "text/event-stream")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("vllm error: %s - %s", resp.Status, string(bodyBytes))
			return
		}

		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp vllmStreamResponse
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

			if choice.FinishReason != "" {
				chunk.Done = true
				chunk.FinishReason = choice.FinishReason
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
//
// vLLM 通过 sentence-transformers 模型支持嵌入。
func (c *VLLMClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": c.model,
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
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send embed request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vllm embed error: %s - %s", resp.Status, string(bodyBytes))
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
func (c *VLLMClient) Name() string {
	return "vllm"
}

// Model 返回当前模型名称
func (c *VLLMClient) Model() string {
	return c.model
}

// Close 关闭客户端连接
func (c *VLLMClient) Close() error {
	return nil
}

// buildRequest 构建请求
func (c *VLLMClient) buildRequest(req Request, stream bool) vllmRequest {
	vllmReq := vllmRequest{
		Model:       c.model,
		Messages:    make([]vllmMessage, len(req.Messages)),
		Stream:      stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}

	for i, msg := range req.Messages {
		vllmReq.Messages[i] = vllmMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	return vllmReq
}

// convertResponse 转换响应
func (c *VLLMClient) convertResponse(resp vllmResponse) Response {
	if len(resp.Choices) == 0 {
		return Response{}
	}

	choice := resp.Choices[0]
	return Response{
		ID:      resp.ID,
		Content: choice.Message.Content,
		TokenUsage: message.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		FinishReason: choice.FinishReason,
	}
}

// compile-time interface check
var _ Provider = (*VLLMClient)(nil)
