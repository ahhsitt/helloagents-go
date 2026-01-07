package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

// StabilityClient Stability AI 图像生成客户端
//
// 支持 Stable Diffusion 3.5 系列模型。
type StabilityClient struct {
	httpClient *http.Client
	options    *Options
}

// Stability AI 支持的模型
const (
	ModelSD35Large       = "sd3.5-large"
	ModelSD35LargeTurbo  = "sd3.5-large-turbo"
	ModelSD35Medium      = "sd3.5-medium"
	ModelSD3Large        = "sd3-large"
	ModelSD3LargeTurbo   = "sd3-large-turbo"
	ModelSD3Medium       = "sd3-medium"
	ModelStableImageCore = "stable-image-core"
)

// Stability API 端点
const (
	defaultStabilityBaseURL = "https://api.stability.ai"
	stabilitySD35Endpoint   = "/v2beta/stable-image/generate/sd3"
	stabilityCoreEndpoint   = "/v2beta/stable-image/generate/core"
)

// Stability AI 支持的宽高比
var stabilityAspectRatios = []string{
	"1:1", "16:9", "9:16", "21:9", "9:21", "4:5", "5:4", "3:2", "2:3",
}

// Stability AI 宽高比到尺寸的映射
var stabilityAspectRatioSizes = map[string]ImageSize{
	"1:1":  {Width: 1024, Height: 1024},
	"16:9": {Width: 1536, Height: 864},
	"9:16": {Width: 864, Height: 1536},
	"21:9": {Width: 1536, Height: 640},
	"9:21": {Width: 640, Height: 1536},
	"4:5":  {Width: 896, Height: 1120},
	"5:4":  {Width: 1120, Height: 896},
	"3:2":  {Width: 1216, Height: 832},
	"2:3":  {Width: 832, Height: 1216},
}

// NewStability 创建 Stability AI 图像生成客户端
func NewStability(opts ...Option) (*StabilityClient, error) {
	options := DefaultOptions()
	ApplyOptions(options, opts...)

	if options.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if options.Model == "" {
		options.Model = ModelSD35Large
	}

	if options.BaseURL == "" {
		options.BaseURL = defaultStabilityBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	}

	return &StabilityClient{
		httpClient: httpClient,
		options:    options,
	}, nil
}

// Name 返回提供商名称
func (c *StabilityClient) Name() string {
	return "stability"
}

// Model 返回当前模型名称
func (c *StabilityClient) Model() string {
	return c.options.Model
}

// SupportedSizes 返回支持的图像尺寸
func (c *StabilityClient) SupportedSizes() []ImageSize {
	sizes := make([]ImageSize, 0, len(stabilityAspectRatioSizes))
	for _, size := range stabilityAspectRatioSizes {
		sizes = append(sizes, size)
	}
	return sizes
}

// Close 关闭客户端连接
func (c *StabilityClient) Close() error {
	return nil
}

// Generate 生成图像
func (c *StabilityClient) Generate(ctx context.Context, req ImageRequest) (ImageResponse, error) {
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

// doRequest 执行 HTTP 请求
func (c *StabilityClient) doRequest(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 构建 multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 添加 prompt
	if err := writer.WriteField("prompt", req.Prompt); err != nil {
		return ImageResponse{}, WrapError(err, "failed to write prompt")
	}

	// 添加 negative_prompt
	if req.NegativePrompt != "" {
		if err := writer.WriteField("negative_prompt", req.NegativePrompt); err != nil {
			return ImageResponse{}, WrapError(err, "failed to write negative_prompt")
		}
	}

	// 添加 aspect_ratio
	aspectRatio := c.mapAspectRatio(req)
	if err := writer.WriteField("aspect_ratio", aspectRatio); err != nil {
		return ImageResponse{}, WrapError(err, "failed to write aspect_ratio")
	}

	// 添加 seed
	if req.Seed != nil {
		if err := writer.WriteField("seed", strconv.FormatInt(*req.Seed, 10)); err != nil {
			return ImageResponse{}, WrapError(err, "failed to write seed")
		}
	}

	// 添加 output_format
	outputFormat := "png"
	if req.ResponseFormat == FormatBase64 {
		outputFormat = "png" // Stability 返回 base64 时也是 png
	}
	if err := writer.WriteField("output_format", outputFormat); err != nil {
		return ImageResponse{}, WrapError(err, "failed to write output_format")
	}

	// 添加 model
	if err := writer.WriteField("model", c.options.Model); err != nil {
		return ImageResponse{}, WrapError(err, "failed to write model")
	}

	if err := writer.Close(); err != nil {
		return ImageResponse{}, WrapError(err, "failed to close multipart writer")
	}

	// 确定端点
	endpoint := stabilitySD35Endpoint
	if c.options.Model == ModelStableImageCore {
		endpoint = stabilityCoreEndpoint
	}

	// 创建 HTTP 请求
	url := c.options.BaseURL + endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.options.APIKey)

	// 设置接受格式
	if req.ResponseFormat == FormatBase64 {
		httpReq.Header.Set("Accept", "application/json")
	} else {
		httpReq.Header.Set("Accept", "image/*")
	}

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

	// 检查错误
	if httpResp.StatusCode != http.StatusOK {
		return ImageResponse{}, c.mapError(httpResp.StatusCode, respBody)
	}

	// 解析响应
	return c.parseResponse(httpResp, respBody, req)
}

// mapAspectRatio 映射尺寸到宽高比
func (c *StabilityClient) mapAspectRatio(req ImageRequest) string {
	// 如果指定了宽高比，直接使用
	if req.AspectRatio != "" {
		for _, ar := range stabilityAspectRatios {
			if ar == req.AspectRatio {
				return ar
			}
		}
	}

	// 如果指定了尺寸，计算最接近的宽高比
	size := req.Size
	if size.Width == 0 || size.Height == 0 {
		size = c.options.DefaultSize
	}

	targetRatio := float64(size.Width) / float64(size.Height)
	closestAR := "1:1"
	minDiff := 999.0

	for ar, s := range stabilityAspectRatioSizes {
		ratio := float64(s.Width) / float64(s.Height)
		diff := absFloat(ratio - targetRatio)
		if diff < minDiff {
			minDiff = diff
			closestAR = ar
		}
	}

	return closestAR
}

// parseResponse 解析 Stability 响应
func (c *StabilityClient) parseResponse(httpResp *http.Response, body []byte, req ImageRequest) (ImageResponse, error) {
	result := ImageResponse{
		Created: time.Now().Unix(),
		Images:  make([]GeneratedImage, 1),
	}

	contentType := httpResp.Header.Get("Content-Type")

	if req.ResponseFormat == FormatBase64 || contentType == "application/json" {
		// JSON 响应
		var jsonResp struct {
			Image        string `json:"image"`
			FinishReason string `json:"finish_reason"`
			Seed         int64  `json:"seed"`
		}
		if err := json.Unmarshal(body, &jsonResp); err != nil {
			return ImageResponse{}, WrapError(err, "failed to parse JSON response")
		}

		seed := jsonResp.Seed
		result.Images[0] = GeneratedImage{
			Base64:      jsonResp.Image,
			Seed:        &seed,
			ContentType: "image/png",
		}
	} else {
		// Binary 响应
		result.Images[0] = GeneratedImage{
			Base64:      base64.StdEncoding.EncodeToString(body),
			ContentType: contentType,
		}
	}

	// 解析 seed header
	if seedStr := httpResp.Header.Get("seed"); seedStr != "" {
		if seed, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			result.Images[0].Seed = &seed
		}
	}

	return result, nil
}

// mapError 映射 Stability 错误到框架错误
func (c *StabilityClient) mapError(statusCode int, body []byte) error {
	var errResp struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &errResp)

	switch statusCode {
	case 401:
		return ErrInvalidAPIKey
	case 402:
		return WrapError(ErrQuotaExceeded, "insufficient credits")
	case 429:
		return ErrQuotaExceeded
	case 400:
		if errResp.Name == "content_moderation" {
			return ErrContentFiltered
		}
		return WrapError(ErrGenerationFailed, errResp.Message)
	case 500, 502, 503:
		return ErrProviderUnavailable
	default:
		msg := errResp.Message
		if msg == "" {
			msg = fmt.Sprintf("status code: %d", statusCode)
		}
		return WrapError(ErrGenerationFailed, msg)
	}
}

// retry 执行带重试的操作
func (c *StabilityClient) retry(ctx context.Context, fn func() error) error {
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

// absFloat 返回浮点数绝对值
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// compile-time interface check
var _ ImageProvider = (*StabilityClient)(nil)
