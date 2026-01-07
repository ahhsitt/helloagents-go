package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DashScopeClient 阿里云 DashScope 图像生成客户端
//
// 支持通义万象（Wanx）系列模型。
type DashScopeClient struct {
	httpClient *http.Client
	options    *Options
}

// DashScope 支持的模型
const (
	ModelWanxV1      = "wanx-v1"
	ModelWanx21Turbo = "wanx2.1-t2i-turbo"
	ModelWanx21Pro   = "wanx2.1-t2i-pro"
)

// DashScope API 端点
const (
	defaultDashScopeBaseURL = "https://dashscope.aliyuncs.com/api/v1"
	dashScopeImageEndpoint  = "/services/aigc/text2image/image-synthesis"
	dashScopeTaskEndpoint   = "/tasks"
)

// DashScope 支持的尺寸
var dashScopeSizes = []ImageSize{
	{Width: 1024, Height: 1024},
	{Width: 720, Height: 1280},
	{Width: 1280, Height: 720},
}

// DashScope 风格映射
var dashScopeStyleMap = map[ImageStyle]string{
	StylePhotographic: "<photography>",
	StyleAnime:        "<anime>",
	StyleDigitalArt:   "<3d cartoon>",
	StyleInkWash:      "<watercolor>",
	StyleNatural:      "<auto>",
	StyleVivid:        "<auto>",
}

// NewDashScope 创建 DashScope 图像生成客户端
func NewDashScope(opts ...Option) (*DashScopeClient, error) {
	options := DefaultOptions()
	ApplyOptions(options, opts...)

	if options.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if options.Model == "" {
		options.Model = ModelWanx21Turbo
	}

	if options.BaseURL == "" {
		options.BaseURL = defaultDashScopeBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	}

	return &DashScopeClient{
		httpClient: httpClient,
		options:    options,
	}, nil
}

// Name 返回提供商名称
func (c *DashScopeClient) Name() string {
	return "dashscope"
}

// Model 返回当前模型名称
func (c *DashScopeClient) Model() string {
	return c.options.Model
}

// SupportedSizes 返回支持的图像尺寸
func (c *DashScopeClient) SupportedSizes() []ImageSize {
	return dashScopeSizes
}

// Close 关闭客户端连接
func (c *DashScopeClient) Close() error {
	return nil
}

// Generate 生成图像
func (c *DashScopeClient) Generate(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 验证请求
	if req.Prompt == "" {
		return ImageResponse{}, ErrInvalidPrompt
	}

	// 执行请求（带重试）
	var resp ImageResponse
	var err error

	err = c.retry(ctx, func() error {
		resp, err = c.doRequest(ctx, req)
		return err
	})

	if err != nil {
		return ImageResponse{}, err
	}

	resp.Model = c.options.Model
	return resp, nil
}

// dashScopeRequest DashScope 图像生成请求
type dashScopeRequest struct {
	Model      string              `json:"model"`
	Input      dashScopeInput      `json:"input"`
	Parameters dashScopeParameters `json:"parameters,omitempty"`
}

type dashScopeInput struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
}

type dashScopeParameters struct {
	Size  string `json:"size,omitempty"`
	N     int    `json:"n,omitempty"`
	Seed  *int64 `json:"seed,omitempty"`
	Style string `json:"style,omitempty"`
}

// dashScopeResponse DashScope 响应
type dashScopeResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Results    []struct {
			URL string `json:"url"`
		} `json:"results"`
	} `json:"output"`
	Usage struct {
		ImageCount int `json:"image_count"`
	} `json:"usage"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// dashScopeTaskResponse 任务查询响应
type dashScopeTaskResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		Results    []struct {
			URL string `json:"url"`
		} `json:"results"`
		TaskMetrics struct {
			Total     int `json:"TOTAL"`
			Succeeded int `json:"SUCCEEDED"`
			Failed    int `json:"FAILED"`
		} `json:"task_metrics"`
	} `json:"output"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// doRequest 执行 HTTP 请求
func (c *DashScopeClient) doRequest(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 构建请求
	apiReq := c.buildRequest(req)

	// 序列化请求
	body, err := json.Marshal(apiReq)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to marshal request")
	}

	// 创建 HTTP 请求
	url := c.options.BaseURL + dashScopeImageEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.options.APIKey)
	httpReq.Header.Set("X-DashScope-Async", "enable") // 启用异步模式

	// 执行请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return ImageResponse{}, ErrTimeout
		}
		return ImageResponse{}, WrapError(err, "request failed")
	}
	defer httpResp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to read response")
	}

	// 解析响应
	var apiResp dashScopeResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return ImageResponse{}, WrapError(err, "failed to parse response")
	}

	// 检查错误
	if apiResp.Code != "" {
		return ImageResponse{}, c.mapError(httpResp.StatusCode, apiResp.Code, apiResp.Message)
	}

	if httpResp.StatusCode != http.StatusOK {
		return ImageResponse{}, WrapError(ErrGenerationFailed,
			fmt.Sprintf("unexpected status code: %d", httpResp.StatusCode))
	}

	// 如果是异步任务，需要轮询结果
	if apiResp.Output.TaskID != "" && apiResp.Output.TaskStatus != "SUCCEEDED" {
		return c.pollTaskResult(ctx, apiResp.Output.TaskID)
	}

	// 同步响应
	return c.parseResponse(apiResp), nil
}

// pollTaskResult 轮询任务结果
func (c *DashScopeClient) pollTaskResult(ctx context.Context, taskID string) (ImageResponse, error) {
	url := c.options.BaseURL + dashScopeTaskEndpoint + "/" + taskID

	maxAttempts := 60 // 最多等待 60 秒
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return ImageResponse{}, ctx.Err()
		case <-time.After(time.Second):
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return ImageResponse{}, WrapError(err, "failed to create poll request")
		}

		httpReq.Header.Set("Authorization", "Bearer "+c.options.APIKey)

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			continue // 重试
		}

		respBody, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			continue
		}

		var taskResp dashScopeTaskResponse
		if err := json.Unmarshal(respBody, &taskResp); err != nil {
			continue
		}

		if taskResp.Code != "" {
			return ImageResponse{}, c.mapError(httpResp.StatusCode, taskResp.Code, taskResp.Message)
		}

		switch taskResp.Output.TaskStatus {
		case "SUCCEEDED":
			return c.parseTaskResponse(taskResp), nil
		case "FAILED":
			return ImageResponse{}, WrapError(ErrGenerationFailed, "task failed")
		case "PENDING", "RUNNING":
			continue
		default:
			continue
		}
	}

	return ImageResponse{}, WrapError(ErrTimeout, "task polling timeout")
}

// buildRequest 构建 DashScope 请求
func (c *DashScopeClient) buildRequest(req ImageRequest) dashScopeRequest {
	apiReq := dashScopeRequest{
		Model: c.options.Model,
		Input: dashScopeInput{
			Prompt:         req.Prompt,
			NegativePrompt: req.NegativePrompt,
		},
		Parameters: dashScopeParameters{},
	}

	// 设置生成数量
	if req.N > 0 {
		apiReq.Parameters.N = req.N
	} else {
		apiReq.Parameters.N = 1
	}

	// 设置尺寸
	size := req.Size
	if size.Width == 0 || size.Height == 0 {
		size = c.options.DefaultSize
	}
	apiReq.Parameters.Size = c.mapSize(size)

	// 设置种子
	if req.Seed != nil {
		apiReq.Parameters.Seed = req.Seed
	}

	// 设置风格
	if req.Style != "" {
		if styleStr, ok := dashScopeStyleMap[req.Style]; ok {
			apiReq.Parameters.Style = styleStr
		}
	}

	return apiReq
}

// mapSize 映射尺寸到 DashScope 支持的格式
func (c *DashScopeClient) mapSize(size ImageSize) string {
	// 查找完全匹配
	for _, s := range dashScopeSizes {
		if s.Width == size.Width && s.Height == size.Height {
			return fmt.Sprintf("%d*%d", size.Width, size.Height)
		}
	}

	// 查找最接近的尺寸
	closest := dashScopeSizes[0]
	minDiff := abs(closest.Pixels() - size.Pixels())

	for _, s := range dashScopeSizes[1:] {
		diff := abs(s.Pixels() - size.Pixels())
		if diff < minDiff {
			minDiff = diff
			closest = s
		}
	}

	return fmt.Sprintf("%d*%d", closest.Width, closest.Height)
}

// parseResponse 解析同步响应
func (c *DashScopeClient) parseResponse(resp dashScopeResponse) ImageResponse {
	result := ImageResponse{
		Created: time.Now().Unix(),
		Images:  make([]GeneratedImage, len(resp.Output.Results)),
	}

	for i, img := range resp.Output.Results {
		result.Images[i] = GeneratedImage{
			URL:         img.URL,
			ContentType: "image/png",
		}
	}

	return result
}

// parseTaskResponse 解析任务响应
func (c *DashScopeClient) parseTaskResponse(resp dashScopeTaskResponse) ImageResponse {
	result := ImageResponse{
		Created: time.Now().Unix(),
		Images:  make([]GeneratedImage, len(resp.Output.Results)),
	}

	for i, img := range resp.Output.Results {
		result.Images[i] = GeneratedImage{
			URL:         img.URL,
			ContentType: "image/png",
		}
	}

	return result
}

// mapError 映射 DashScope 错误到框架错误
func (c *DashScopeClient) mapError(statusCode int, code string, message string) error {
	switch code {
	case "InvalidApiKey":
		return ErrInvalidAPIKey
	case "Throttling":
		return ErrQuotaExceeded
	case "ContentFiltered", "DataInspectionFailed":
		return ErrContentFiltered
	default:
		if statusCode == 401 {
			return ErrInvalidAPIKey
		}
		if statusCode == 429 {
			return ErrQuotaExceeded
		}
		if message != "" {
			return WrapError(ErrGenerationFailed, message)
		}
		return WrapError(ErrGenerationFailed, code)
	}
}

// retry 执行带重试的操作
func (c *DashScopeClient) retry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.options.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !IsRetryable(err) {
			return err
		}

		if attempt < c.options.MaxRetries {
			// #nosec G115 - attempt is bounded by MaxRetries (typically < 10)
			delay := c.options.RetryDelay * time.Duration(1<<uint(attempt))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return lastErr
}

// compile-time interface check
var _ ImageProvider = (*DashScopeClient)(nil)
