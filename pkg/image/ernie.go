package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ERNIEClient 百度文心一格图像生成客户端
//
// 支持 ERNIE-ViLG 系列模型。
type ERNIEClient struct {
	httpClient  *http.Client
	options     *Options
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
}

// ERNIE 支持的模型
const (
	ModelERNIEViLG2 = "ernie-vilg-v2"
)

// ERNIE API 端点
const (
	defaultERNIEBaseURL = "https://aip.baidubce.com"
	ernieTokenEndpoint  = "/oauth/2.0/token"
	ernieImageEndpoint  = "/rpc/2.0/ernievilg/v1/txt2imgv2"
)

// ERNIE 支持的尺寸
var ernieSizes = []ImageSize{
	{Width: 512, Height: 512},
	{Width: 640, Height: 360},
	{Width: 360, Height: 640},
	{Width: 1024, Height: 1024},
	{Width: 1280, Height: 720},
	{Width: 720, Height: 1280},
	{Width: 2048, Height: 2048},
}

// ERNIE 风格映射
var ernieStyleMap = map[ImageStyle]string{
	StylePhotographic: "写实风格",
	StyleAnime:        "二次元",
	StyleDigitalArt:   "概念艺术",
	StyleInkWash:      "古风",
	StyleNatural:      "写实风格",
	StyleVivid:        "探索无限",
}

// NewERNIE 创建百度 ERNIE 图像生成客户端
func NewERNIE(opts ...Option) (*ERNIEClient, error) {
	options := DefaultOptions()
	ApplyOptions(options, opts...)

	if options.APIKey == "" || options.SecretKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if options.Model == "" {
		options.Model = ModelERNIEViLG2
	}

	if options.BaseURL == "" {
		options.BaseURL = defaultERNIEBaseURL
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	}

	return &ERNIEClient{
		httpClient: httpClient,
		options:    options,
	}, nil
}

// Name 返回提供商名称
func (c *ERNIEClient) Name() string {
	return "ernie"
}

// Model 返回当前模型名称
func (c *ERNIEClient) Model() string {
	return c.options.Model
}

// SupportedSizes 返回支持的图像尺寸
func (c *ERNIEClient) SupportedSizes() []ImageSize {
	return ernieSizes
}

// Close 关闭客户端连接
func (c *ERNIEClient) Close() error {
	return nil
}

// Generate 生成图像
func (c *ERNIEClient) Generate(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 验证请求
	if req.Prompt == "" {
		return ImageResponse{}, ErrInvalidPrompt
	}

	// 确保有有效的 access token
	if err := c.ensureAccessToken(ctx); err != nil {
		return ImageResponse{}, err
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

// ensureAccessToken 确保有有效的 access token
func (c *ERNIEClient) ensureAccessToken(ctx context.Context) error {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		c.tokenMu.RUnlock()
		return nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// 双重检查
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	// 获取新 token
	return c.refreshAccessToken(ctx)
}

// refreshAccessToken 刷新 access token
func (c *ERNIEClient) refreshAccessToken(ctx context.Context) error {
	params := url.Values{}
	params.Set("grant_type", "client_credentials")
	params.Set("client_id", c.options.APIKey)
	params.Set("client_secret", c.options.SecretKey)

	url := c.options.BaseURL + ernieTokenEndpoint + "?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return WrapError(err, "failed to create token request")
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return WrapError(err, "token request failed")
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return WrapError(err, "failed to read token response")
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error,omitempty"`
		ErrorDesc   string `json:"error_description,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return WrapError(err, "failed to parse token response")
	}

	if tokenResp.Error != "" {
		return WrapError(ErrInvalidAPIKey, tokenResp.ErrorDesc)
	}

	c.accessToken = tokenResp.AccessToken
	// 提前 5 分钟过期
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)

	return nil
}

// ernieRequest ERNIE 图像生成请求
type ernieRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ImageNum       int    `json:"image_num,omitempty"`
	Style          string `json:"style,omitempty"`
	SamplerIndex   string `json:"sampler_index,omitempty"`
}

// ernieResponse ERNIE 响应
type ernieResponse struct {
	LogID int64 `json:"log_id"`
	Data  struct {
		TaskID  string `json:"task_id"`
		ImgUrls []struct {
			Image string `json:"image"`
		} `json:"img_urls"`
	} `json:"data"`
	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// doRequest 执行 HTTP 请求
func (c *ERNIEClient) doRequest(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 构建请求
	apiReq := c.buildRequest(req)

	// 序列化请求
	body, err := json.Marshal(apiReq)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to marshal request")
	}

	// 创建 HTTP 请求
	url := c.options.BaseURL + ernieImageEndpoint + "?access_token=" + c.accessToken
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
	var apiResp ernieResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return ImageResponse{}, WrapError(err, "failed to parse response")
	}

	// 检查错误
	if apiResp.ErrorCode != 0 {
		return ImageResponse{}, c.mapError(apiResp.ErrorCode, apiResp.ErrorMsg)
	}

	// 如果是异步任务，需要轮询结果
	if apiResp.Data.TaskID != "" && len(apiResp.Data.ImgUrls) == 0 {
		return c.pollTaskResult(ctx, apiResp.Data.TaskID)
	}

	return c.parseResponse(apiResp), nil
}

// pollTaskResult 轮询任务结果
func (c *ERNIEClient) pollTaskResult(ctx context.Context, taskID string) (ImageResponse, error) {
	// ERNIE 使用不同的查询端点
	queryEndpoint := "/rpc/2.0/ernievilg/v1/getImgv2"

	maxAttempts := 60
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return ImageResponse{}, ctx.Err()
		case <-time.After(time.Second):
		}

		queryReq := struct {
			TaskID string `json:"task_id"`
		}{TaskID: taskID}

		body, _ := json.Marshal(queryReq)

		url := c.options.BaseURL + queryEndpoint + "?access_token=" + c.accessToken
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			continue
		}

		respBody, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			continue
		}

		var taskResp struct {
			Data struct {
				TaskID  string `json:"task_id"`
				Status  int    `json:"status"`
				ImgUrls []struct {
					Image string `json:"image"`
				} `json:"img_urls"`
			} `json:"data"`
			ErrorCode int    `json:"error_code,omitempty"`
			ErrorMsg  string `json:"error_msg,omitempty"`
		}

		if err := json.Unmarshal(respBody, &taskResp); err != nil {
			continue
		}

		if taskResp.ErrorCode != 0 {
			return ImageResponse{}, c.mapError(taskResp.ErrorCode, taskResp.ErrorMsg)
		}

		// status: 0=init, 1=running, 2=success, 3=failed
		switch taskResp.Data.Status {
		case 2: // success
			result := ImageResponse{
				Created: time.Now().Unix(),
				Images:  make([]GeneratedImage, len(taskResp.Data.ImgUrls)),
			}
			for i, img := range taskResp.Data.ImgUrls {
				result.Images[i] = GeneratedImage{
					URL:         img.Image,
					ContentType: "image/png",
				}
			}
			return result, nil
		case 3: // failed
			return ImageResponse{}, WrapError(ErrGenerationFailed, "task failed")
		default:
			continue
		}
	}

	return ImageResponse{}, WrapError(ErrTimeout, "task polling timeout")
}

// buildRequest 构建 ERNIE 请求
func (c *ERNIEClient) buildRequest(req ImageRequest) ernieRequest {
	apiReq := ernieRequest{
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
	}

	// 设置生成数量
	if req.N > 0 && req.N <= 6 {
		apiReq.ImageNum = req.N
	} else {
		apiReq.ImageNum = 1
	}

	// 设置尺寸
	size := req.Size
	if size.Width == 0 || size.Height == 0 {
		size = c.options.DefaultSize
	}
	mappedSize := c.mapSize(size)
	apiReq.Width = mappedSize.Width
	apiReq.Height = mappedSize.Height

	// 设置风格
	if req.Style != "" {
		if styleStr, ok := ernieStyleMap[req.Style]; ok {
			apiReq.Style = styleStr
		}
	}

	return apiReq
}

// mapSize 映射尺寸到 ERNIE 支持的格式
func (c *ERNIEClient) mapSize(size ImageSize) ImageSize {
	// 查找完全匹配
	for _, s := range ernieSizes {
		if s.Width == size.Width && s.Height == size.Height {
			return s
		}
	}

	// 查找最接近的尺寸
	closest := ernieSizes[0]
	minDiff := abs(closest.Pixels() - size.Pixels())

	for _, s := range ernieSizes[1:] {
		diff := abs(s.Pixels() - size.Pixels())
		if diff < minDiff {
			minDiff = diff
			closest = s
		}
	}

	return closest
}

// parseResponse 解析 ERNIE 响应
func (c *ERNIEClient) parseResponse(resp ernieResponse) ImageResponse {
	result := ImageResponse{
		Created: time.Now().Unix(),
		Images:  make([]GeneratedImage, len(resp.Data.ImgUrls)),
	}

	for i, img := range resp.Data.ImgUrls {
		result.Images[i] = GeneratedImage{
			URL:         img.Image,
			ContentType: "image/png",
		}
	}

	return result
}

// mapError 映射 ERNIE 错误到框架错误
func (c *ERNIEClient) mapError(code int, message string) error {
	switch code {
	case 110, 111: // Access token 相关
		return ErrInvalidAPIKey
	case 18, 19: // QPS/配额限制
		return ErrQuotaExceeded
	case 17: // 每日调用量超限
		return ErrQuotaExceeded
	case 282000: // 内容审核
		return ErrContentFiltered
	default:
		if message != "" {
			return WrapError(ErrGenerationFailed, fmt.Sprintf("error %d: %s", code, message))
		}
		return WrapError(ErrGenerationFailed, fmt.Sprintf("error code: %d", code))
	}
}

// retry 执行带重试的操作
func (c *ERNIEClient) retry(ctx context.Context, fn func() error) error {
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
var _ ImageProvider = (*ERNIEClient)(nil)
