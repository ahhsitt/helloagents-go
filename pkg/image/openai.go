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

// OpenAIClient OpenAI 图像生成客户端
//
// 支持 DALL-E 3 和 GPT Image 系列模型。
type OpenAIClient struct {
	httpClient *http.Client
	options    *Options
}

// OpenAI 支持的模型
const (
	ModelDALLE3       = "dall-e-3"
	ModelDALLE2       = "dall-e-2"
	ModelGPTImage1    = "gpt-image-1"
	ModelGPTImage1_5  = "gpt-image-1.5"
	ModelGPTImage1Min = "gpt-image-1-mini"
)

// OpenAI API 端点
const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	openAIImagesEndpoint = "/images/generations"
)

// DALL-E 3 支持的尺寸
var openAIDALLE3Sizes = []ImageSize{
	{Width: 1024, Height: 1024},
	{Width: 1024, Height: 1792},
	{Width: 1792, Height: 1024},
}

// GPT Image 支持的尺寸
var openAIGPTImageSizes = []ImageSize{
	{Width: 1024, Height: 1024},
	{Width: 1024, Height: 1536},
	{Width: 1536, Height: 1024},
}

// NewOpenAI 创建 OpenAI 图像生成客户端
func NewOpenAI(opts ...Option) (*OpenAIClient, error) {
	options := DefaultOptions()
	ApplyOptions(options, opts...)

	if options.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if options.Model == "" {
		options.Model = ModelDALLE3
	}

	if options.BaseURL == "" {
		options.BaseURL = defaultOpenAIBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	}

	return &OpenAIClient{
		httpClient: httpClient,
		options:    options,
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

// SupportedSizes 返回支持的图像尺寸
func (c *OpenAIClient) SupportedSizes() []ImageSize {
	if isGPTImageModel(c.options.Model) {
		return openAIGPTImageSizes
	}
	return openAIDALLE3Sizes
}

// Close 关闭客户端连接
func (c *OpenAIClient) Close() error {
	return nil
}

// Generate 生成图像
func (c *OpenAIClient) Generate(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 验证请求
	if req.Prompt == "" {
		return ImageResponse{}, ErrInvalidPrompt
	}

	// 构建请求
	apiReq := c.buildRequest(req)

	// 执行请求（带重试）
	var resp ImageResponse
	var err error

	err = c.retry(ctx, func() error {
		resp, err = c.doRequest(ctx, apiReq)
		return err
	})

	if err != nil {
		return ImageResponse{}, err
	}

	resp.Model = c.options.Model
	return resp, nil
}

// openAIImageRequest OpenAI 图像生成 API 请求
type openAIImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// openAIImageResponse OpenAI 图像生成 API 响应
type openAIImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		B64JSON       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
	Error *openAIError `json:"error,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// buildRequest 构建 OpenAI 请求
func (c *OpenAIClient) buildRequest(req ImageRequest) openAIImageRequest {
	apiReq := openAIImageRequest{
		Model:  c.options.Model,
		Prompt: req.Prompt,
	}

	// 设置生成数量
	if req.N > 0 {
		apiReq.N = req.N
	} else {
		apiReq.N = 1
	}

	// DALL-E 3 只支持 n=1
	if c.options.Model == ModelDALLE3 && apiReq.N > 1 {
		apiReq.N = 1
	}

	// 设置尺寸
	size := req.Size
	if size.Width == 0 || size.Height == 0 {
		size = c.options.DefaultSize
	}
	apiReq.Size = c.mapSize(size)

	// 设置质量（DALL-E 3 支持）
	if c.options.Model == ModelDALLE3 {
		quality := req.Quality
		if quality == "" {
			quality = c.options.DefaultQuality
		}
		if quality == QualityHD {
			apiReq.Quality = "hd"
		} else {
			apiReq.Quality = "standard"
		}

		// 设置风格
		style := req.Style
		if style == "" {
			style = c.options.DefaultStyle
		}
		if style == StyleNatural {
			apiReq.Style = "natural"
		} else if style == StyleVivid || style != "" {
			apiReq.Style = "vivid"
		}
	}

	// 设置响应格式
	format := req.ResponseFormat
	if format == "" {
		format = c.options.DefaultFormat
	}
	if format == FormatBase64 {
		apiReq.ResponseFormat = "b64_json"
	} else {
		apiReq.ResponseFormat = "url"
	}

	return apiReq
}

// mapSize 映射尺寸到 OpenAI 支持的格式
func (c *OpenAIClient) mapSize(size ImageSize) string {
	supportedSizes := c.SupportedSizes()

	// 查找完全匹配
	for _, s := range supportedSizes {
		if s.Width == size.Width && s.Height == size.Height {
			return fmt.Sprintf("%dx%d", size.Width, size.Height)
		}
	}

	// 查找最接近的尺寸
	closest := supportedSizes[0]
	minDiff := abs(closest.Pixels() - size.Pixels())

	for _, s := range supportedSizes[1:] {
		diff := abs(s.Pixels() - size.Pixels())
		if diff < minDiff {
			minDiff = diff
			closest = s
		}
	}

	return fmt.Sprintf("%dx%d", closest.Width, closest.Height)
}

// doRequest 执行 HTTP 请求
func (c *OpenAIClient) doRequest(ctx context.Context, apiReq openAIImageRequest) (ImageResponse, error) {
	// 序列化请求
	body, err := json.Marshal(apiReq)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to marshal request")
	}

	// 创建 HTTP 请求
	url := c.options.BaseURL + openAIImagesEndpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.options.APIKey)

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
	var apiResp openAIImageResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return ImageResponse{}, WrapError(err, "failed to parse response")
	}

	// 检查错误
	if apiResp.Error != nil {
		return ImageResponse{}, c.mapError(httpResp.StatusCode, apiResp.Error)
	}

	if httpResp.StatusCode != http.StatusOK {
		return ImageResponse{}, WrapError(ErrGenerationFailed,
			fmt.Sprintf("unexpected status code: %d", httpResp.StatusCode))
	}

	// 转换响应
	return c.parseResponse(apiResp), nil
}

// parseResponse 解析 OpenAI 响应
func (c *OpenAIClient) parseResponse(resp openAIImageResponse) ImageResponse {
	result := ImageResponse{
		Created: resp.Created,
		Images:  make([]GeneratedImage, len(resp.Data)),
	}

	for i, img := range resp.Data {
		result.Images[i] = GeneratedImage{
			URL:           img.URL,
			Base64:        img.B64JSON,
			RevisedPrompt: img.RevisedPrompt,
			ContentType:   "image/png",
		}
	}

	return result
}

// mapError 映射 OpenAI 错误到框架错误
func (c *OpenAIClient) mapError(statusCode int, apiErr *openAIError) error {
	switch statusCode {
	case 401:
		return ErrInvalidAPIKey
	case 429:
		return ErrQuotaExceeded
	case 400:
		if apiErr.Code == "content_policy_violation" {
			return ErrContentFiltered
		}
		return WrapError(ErrGenerationFailed, apiErr.Message)
	case 500, 502, 503:
		return ErrProviderUnavailable
	default:
		return WrapError(ErrGenerationFailed, apiErr.Message)
	}
}

// retry 执行带重试的操作
func (c *OpenAIClient) retry(ctx context.Context, fn func() error) error {
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

// isGPTImageModel 判断是否是 GPT Image 系列模型
func isGPTImageModel(model string) bool {
	return model == ModelGPTImage1 ||
		model == ModelGPTImage1_5 ||
		model == ModelGPTImage1Min
}

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// compile-time interface check
var _ ImageProvider = (*OpenAIClient)(nil)
