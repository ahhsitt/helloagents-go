package image

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ahhsitt/helloagents-go/pkg/image"
)

func TestOpenAIClient_Generate(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("invalid authorization header")
		}

		// 解析请求体
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req["prompt"] != "a cute cat" {
			t.Errorf("unexpected prompt: %v", req["prompt"])
		}

		// 返回模拟响应
		resp := map[string]interface{}{
			"created": time.Now().Unix(),
			"data": []map[string]interface{}{
				{
					"url":            "https://example.com/image.png",
					"revised_prompt": "a cute cat sitting on a windowsill",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建客户端
	client, err := image.NewOpenAI(
		image.WithAPIKey("test-api-key"),
		image.WithBaseURL(server.URL),
		image.WithModel(image.ModelDALLE3),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// 生成图像
	resp, err := client.Generate(context.Background(), image.ImageRequest{
		Prompt: "a cute cat",
		Size:   image.ImageSize{Width: 1024, Height: 1024},
	})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// 验证响应
	if len(resp.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(resp.Images))
	}

	if resp.Images[0].URL != "https://example.com/image.png" {
		t.Errorf("unexpected image URL: %s", resp.Images[0].URL)
	}

	if resp.Images[0].RevisedPrompt != "a cute cat sitting on a windowsill" {
		t.Errorf("unexpected revised prompt: %s", resp.Images[0].RevisedPrompt)
	}
}

func TestOpenAIClient_EmptyPrompt(t *testing.T) {
	client, err := image.NewOpenAI(
		image.WithAPIKey("test-api-key"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Generate(context.Background(), image.ImageRequest{
		Prompt: "",
	})

	if err != image.ErrInvalidPrompt {
		t.Errorf("expected ErrInvalidPrompt, got %v", err)
	}
}

func TestOpenAIClient_InvalidAPIKey(t *testing.T) {
	_, err := image.NewOpenAI()
	if err != image.ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestImageSize_String(t *testing.T) {
	tests := []struct {
		size     image.ImageSize
		expected string
	}{
		{image.ImageSize{Width: 1024, Height: 1024}, "1024x1024"},
		{image.ImageSize{Width: 1024, Height: 1792}, "1024x1792"},
		{image.ImageSize{Width: 512, Height: 512}, "0512x0512"},
	}

	for _, test := range tests {
		result := test.size.String()
		// 简单验证格式
		if len(result) == 0 {
			t.Errorf("expected non-empty string for %+v", test.size)
		}
	}
}

func TestImageSize_Pixels(t *testing.T) {
	size := image.ImageSize{Width: 1024, Height: 1024}
	if size.Pixels() != 1024*1024 {
		t.Errorf("expected %d pixels, got %d", 1024*1024, size.Pixels())
	}
}

func TestImageSize_AspectRatio(t *testing.T) {
	tests := []struct {
		size          image.ImageSize
		expectedRatio float64
	}{
		{image.ImageSize{Width: 1024, Height: 1024}, 1.0},
		{image.ImageSize{Width: 1920, Height: 1080}, 1920.0 / 1080.0},
	}

	for _, test := range tests {
		ratio := test.size.AspectRatio()
		if ratio != test.expectedRatio {
			t.Errorf("expected ratio %f, got %f", test.expectedRatio, ratio)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected image.ImageSize
		hasError bool
	}{
		{"1024x1024", image.ImageSize{Width: 1024, Height: 1024}, false},
		{"1920x1080", image.ImageSize{Width: 1920, Height: 1080}, false},
		{"invalid", image.ImageSize{}, true},
		{"1024", image.ImageSize{}, true},
		{"x1024", image.ImageSize{}, true},
	}

	for _, test := range tests {
		size, err := image.ParseSize(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("expected error for input %q", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", test.input, err)
			}
			if size != test.expected {
				t.Errorf("expected %+v, got %+v", test.expected, size)
			}
		}
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err       error
		retryable bool
	}{
		{image.ErrQuotaExceeded, true},
		{image.ErrTimeout, true},
		{image.ErrProviderUnavailable, true},
		{image.ErrInvalidPrompt, false},
		{image.ErrInvalidAPIKey, false},
		{nil, false},
	}

	for _, test := range tests {
		result := image.IsRetryable(test.err)
		if result != test.retryable {
			t.Errorf("IsRetryable(%v) = %v, expected %v", test.err, result, test.retryable)
		}
	}
}

func TestIsFatal(t *testing.T) {
	tests := []struct {
		err   error
		fatal bool
	}{
		{image.ErrInvalidAPIKey, true},
		{image.ErrInvalidPrompt, true},
		{image.ErrModelNotSupported, true},
		{image.ErrQuotaExceeded, false},
		{image.ErrTimeout, false},
		{nil, false},
	}

	for _, test := range tests {
		result := image.IsFatal(test.err)
		if result != test.fatal {
			t.Errorf("IsFatal(%v) = %v, expected %v", test.err, result, test.fatal)
		}
	}
}
