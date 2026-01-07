package image

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HunyuanClient 腾讯混元图像生成客户端
type HunyuanClient struct {
	httpClient *http.Client
	options    *Options
}

// Hunyuan 支持的模型
const (
	ModelHunyuanImage = "hunyuan-image"
)

// Hunyuan API 端点
const (
	defaultHunyuanHost = "hunyuan.tencentcloudapi.com"
	hunyuanService     = "hunyuan"
	hunyuanAction      = "TextToImage"
	hunyuanVersion     = "2023-09-01"
	hunyuanRegion      = "ap-guangzhou"
)

// Hunyuan 支持的尺寸
var hunyuanSizes = []ImageSize{
	{Width: 768, Height: 768},
	{Width: 768, Height: 1024},
	{Width: 1024, Height: 768},
	{Width: 1024, Height: 1024},
}

// NewHunyuan 创建腾讯混元图像生成客户端
func NewHunyuan(opts ...Option) (*HunyuanClient, error) {
	options := DefaultOptions()
	ApplyOptions(options, opts...)

	// Hunyuan 使用 APIKey 作为 SecretId, SecretKey 作为 SecretKey
	if options.APIKey == "" || options.SecretKey == "" {
		return nil, ErrInvalidAPIKey
	}

	if options.Model == "" {
		options.Model = ModelHunyuanImage
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	}

	return &HunyuanClient{
		httpClient: httpClient,
		options:    options,
	}, nil
}

// Name 返回提供商名称
func (c *HunyuanClient) Name() string {
	return "hunyuan"
}

// Model 返回当前模型名称
func (c *HunyuanClient) Model() string {
	return c.options.Model
}

// SupportedSizes 返回支持的图像尺寸
func (c *HunyuanClient) SupportedSizes() []ImageSize {
	return hunyuanSizes
}

// Close 关闭客户端连接
func (c *HunyuanClient) Close() error {
	return nil
}

// Generate 生成图像
func (c *HunyuanClient) Generate(ctx context.Context, req ImageRequest) (ImageResponse, error) {
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

// hunyuanRequest 腾讯混元请求
type hunyuanRequest struct {
	Prompt         string `json:"Prompt"`
	NegativePrompt string `json:"NegativePrompt,omitempty"`
	Style          string `json:"Style,omitempty"`
	Resolution     string `json:"Resolution,omitempty"`
	Num            int    `json:"Num,omitempty"`
	Seed           *int64 `json:"Seed,omitempty"`
	RspImgType     string `json:"RspImgType,omitempty"`
}

// hunyuanResponse 腾讯混元响应
type hunyuanResponse struct {
	Response struct {
		RequestID   string `json:"RequestId"`
		ResultImage string `json:"ResultImage,omitempty"`
		Error       *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"Response"`
}

// doRequest 执行 HTTP 请求
func (c *HunyuanClient) doRequest(ctx context.Context, req ImageRequest) (ImageResponse, error) {
	// 构建请求
	apiReq := c.buildRequest(req)

	// 序列化请求
	body, err := json.Marshal(apiReq)
	if err != nil {
		return ImageResponse{}, WrapError(err, "failed to marshal request")
	}

	// 创建签名请求
	timestamp := time.Now().Unix()
	httpReq, err := c.createSignedRequest(ctx, body, timestamp)
	if err != nil {
		return ImageResponse{}, err
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

	// 解析响应
	var apiResp hunyuanResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return ImageResponse{}, WrapError(err, "failed to parse response")
	}

	// 检查错误
	if apiResp.Response.Error != nil {
		return ImageResponse{}, c.mapError(apiResp.Response.Error.Code, apiResp.Response.Error.Message)
	}

	return c.parseResponse(apiResp), nil
}

// createSignedRequest 创建带 TC3 签名的请求
func (c *HunyuanClient) createSignedRequest(ctx context.Context, body []byte, timestamp int64) (*http.Request, error) {
	host := defaultHunyuanHost
	algorithm := "TC3-HMAC-SHA256"
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := date + "/" + hunyuanService + "/tc3_request"

	// 计算请求签名
	hashedPayload := sha256Hex(body)

	// 规范请求
	canonicalHeaders := "content-type:application/json\nhost:" + host + "\nx-tc-action:" + strings.ToLower(hunyuanAction) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := "POST\n/\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + hashedPayload

	// 待签名字符串
	stringToSign := algorithm + "\n" + fmt.Sprintf("%d", timestamp) + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))

	// 计算签名
	secretDate := hmacSHA256([]byte("TC3"+c.options.SecretKey), date)
	secretService := hmacSHA256(secretDate, hunyuanService)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// 构建 Authorization
	authorization := algorithm + " Credential=" + c.options.APIKey + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	// 创建请求
	url := "https://" + host
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, WrapError(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-TC-Action", hunyuanAction)
	httpReq.Header.Set("X-TC-Version", hunyuanVersion)
	httpReq.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	httpReq.Header.Set("X-TC-Region", hunyuanRegion)
	httpReq.Header.Set("Authorization", authorization)

	return httpReq, nil
}

// buildRequest 构建混元请求
func (c *HunyuanClient) buildRequest(req ImageRequest) hunyuanRequest {
	apiReq := hunyuanRequest{
		Prompt:         req.Prompt,
		NegativePrompt: req.NegativePrompt,
	}

	// 设置生成数量
	if req.N > 0 && req.N <= 4 {
		apiReq.Num = req.N
	} else {
		apiReq.Num = 1
	}

	// 设置尺寸
	size := req.Size
	if size.Width == 0 || size.Height == 0 {
		size = c.options.DefaultSize
	}
	mappedSize := c.mapSize(size)
	apiReq.Resolution = fmt.Sprintf("%d:%d", mappedSize.Width, mappedSize.Height)

	// 设置种子
	if req.Seed != nil {
		apiReq.Seed = req.Seed
	}

	// 设置响应格式
	if req.ResponseFormat == FormatBase64 {
		apiReq.RspImgType = "base64"
	} else {
		apiReq.RspImgType = "url"
	}

	return apiReq
}

// mapSize 映射尺寸到混元支持的格式
func (c *HunyuanClient) mapSize(size ImageSize) ImageSize {
	// 查找完全匹配
	for _, s := range hunyuanSizes {
		if s.Width == size.Width && s.Height == size.Height {
			return s
		}
	}

	// 查找最接近的尺寸
	closest := hunyuanSizes[0]
	minDiff := abs(closest.Pixels() - size.Pixels())

	for _, s := range hunyuanSizes[1:] {
		diff := abs(s.Pixels() - size.Pixels())
		if diff < minDiff {
			minDiff = diff
			closest = s
		}
	}

	return closest
}

// parseResponse 解析混元响应
func (c *HunyuanClient) parseResponse(resp hunyuanResponse) ImageResponse {
	result := ImageResponse{
		Created: time.Now().Unix(),
		Images:  make([]GeneratedImage, 1),
	}

	// 检查是 URL 还是 Base64
	imgData := resp.Response.ResultImage
	if strings.HasPrefix(imgData, "http") {
		result.Images[0] = GeneratedImage{
			URL:         imgData,
			ContentType: "image/png",
		}
	} else {
		result.Images[0] = GeneratedImage{
			Base64:      imgData,
			ContentType: "image/png",
		}
	}

	return result
}

// mapError 映射混元错误到框架错误
func (c *HunyuanClient) mapError(code string, message string) error {
	switch code {
	case "AuthFailure", "AuthFailure.SecretIdNotFound", "AuthFailure.SignatureFailure":
		return ErrInvalidAPIKey
	case "RequestLimitExceeded", "LimitExceeded":
		return ErrQuotaExceeded
	case "UnsupportedOperation.ContentRiskDetected", "FailedOperation.ContentFilter":
		return ErrContentFiltered
	default:
		if message != "" {
			return WrapError(ErrGenerationFailed, fmt.Sprintf("%s: %s", code, message))
		}
		return WrapError(ErrGenerationFailed, code)
	}
}

// retry 执行带重试的操作
func (c *HunyuanClient) retry(ctx context.Context, fn func() error) error {
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

// sha256Hex 计算 SHA256 并返回十六进制字符串
func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// hmacSHA256 计算 HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// compile-time interface check
var _ ImageProvider = (*HunyuanClient)(nil)
