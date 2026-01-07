package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ahhsitt/helloagents-go/pkg/image"
	"github.com/ahhsitt/helloagents-go/pkg/tools"
)

// ImageGenerator 图像生成工具
//
// 使用文生图 AI 服务根据文本描述生成图像。
type ImageGenerator struct {
	provider image.ImageProvider
}

// NewImageGenerator 创建图像生成工具
//
// 参数:
//   - provider: 图像生成服务提供商（如 OpenAI, Stability AI 等）
func NewImageGenerator(provider image.ImageProvider) *ImageGenerator {
	return &ImageGenerator{
		provider: provider,
	}
}

// Name 返回工具名称
func (g *ImageGenerator) Name() string {
	return "image_generator"
}

// Description 返回工具描述
func (g *ImageGenerator) Description() string {
	return "Generate images from text descriptions using AI. Returns image URL or base64 encoded data."
}

// Parameters 返回参数 Schema
func (g *ImageGenerator) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"prompt": {
				Type:        "string",
				Description: "A detailed text description of the image to generate. Be specific about subject, style, colors, composition, and mood for best results.",
			},
			"negative_prompt": {
				Type:        "string",
				Description: "Things to avoid in the generated image. Optional.",
			},
			"size": {
				Type:        "string",
				Description: "Image dimensions in 'WIDTHxHEIGHT' format (e.g., '1024x1024', '1024x1792'). Optional, defaults to 1024x1024.",
				Enum:        []string{"1024x1024", "1024x1792", "1792x1024", "1024x1536", "1536x1024"},
			},
			"style": {
				Type:        "string",
				Description: "Image style preset. Optional.",
				Enum:        []string{"vivid", "natural", "anime", "photographic", "digital-art"},
			},
			"quality": {
				Type:        "string",
				Description: "Image quality level. Optional, defaults to 'standard'.",
				Enum:        []string{"standard", "hd"},
			},
		},
		Required: []string{"prompt"},
	}
}

// Execute 执行图像生成
func (g *ImageGenerator) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 解析参数
	req, err := g.parseArgs(args)
	if err != nil {
		return "", err
	}

	// 调用图像生成服务
	resp, err := g.provider.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("image generation failed: %w", err)
	}

	// 格式化结果
	return g.formatResult(resp)
}

// parseArgs 解析工具参数
func (g *ImageGenerator) parseArgs(args map[string]interface{}) (image.ImageRequest, error) {
	req := image.ImageRequest{
		ResponseFormat: image.FormatURL,
	}

	// 解析 prompt（必填）
	promptRaw, ok := args["prompt"]
	if !ok {
		return req, fmt.Errorf("missing required parameter: prompt")
	}
	prompt, ok := promptRaw.(string)
	if !ok {
		return req, fmt.Errorf("prompt must be a string")
	}
	if prompt == "" {
		return req, fmt.Errorf("prompt cannot be empty")
	}
	req.Prompt = prompt

	// 解析 negative_prompt（可选）
	if negPromptRaw, ok := args["negative_prompt"]; ok {
		if negPrompt, ok := negPromptRaw.(string); ok {
			req.NegativePrompt = negPrompt
		}
	}

	// 解析 size（可选）
	if sizeRaw, ok := args["size"]; ok {
		if sizeStr, ok := sizeRaw.(string); ok && sizeStr != "" {
			size, err := image.ParseSize(sizeStr)
			if err != nil {
				return req, fmt.Errorf("invalid size format: %s", sizeStr)
			}
			req.Size = size
		}
	}

	// 解析 style（可选）
	if styleRaw, ok := args["style"]; ok {
		if styleStr, ok := styleRaw.(string); ok && styleStr != "" {
			req.Style = image.ImageStyle(styleStr)
		}
	}

	// 解析 quality（可选）
	if qualityRaw, ok := args["quality"]; ok {
		if qualityStr, ok := qualityRaw.(string); ok && qualityStr != "" {
			req.Quality = image.ImageQuality(qualityStr)
		}
	}

	return req, nil
}

// formatResult 格式化生成结果
func (g *ImageGenerator) formatResult(resp image.ImageResponse) (string, error) {
	if len(resp.Images) == 0 {
		return "", fmt.Errorf("no images generated")
	}

	// 构建结果
	result := struct {
		Success bool   `json:"success"`
		Model   string `json:"model"`
		Images  []struct {
			URL           string `json:"url,omitempty"`
			Base64        string `json:"base64,omitempty"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		} `json:"images"`
	}{
		Success: true,
		Model:   resp.Model,
		Images: make([]struct {
			URL           string `json:"url,omitempty"`
			Base64        string `json:"base64,omitempty"`
			RevisedPrompt string `json:"revised_prompt,omitempty"`
		}, len(resp.Images)),
	}

	for i, img := range resp.Images {
		result.Images[i].URL = img.URL
		result.Images[i].RevisedPrompt = img.RevisedPrompt
		// 只在没有 URL 时返回 Base64（太长了）
		if img.URL == "" && img.Base64 != "" {
			// 截断 base64 数据以避免输出过长
			if len(img.Base64) > 100 {
				result.Images[i].Base64 = img.Base64[:100] + "...(truncated, full length: " + fmt.Sprintf("%d", len(img.Base64)) + ")"
			} else {
				result.Images[i].Base64 = img.Base64
			}
		}
	}

	// 序列化为 JSON
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to format result: %w", err)
	}

	return string(data), nil
}

// Validate 验证参数
func (g *ImageGenerator) Validate(args map[string]interface{}) error {
	prompt, ok := args["prompt"]
	if !ok {
		return fmt.Errorf("missing required parameter: prompt")
	}
	if _, ok := prompt.(string); !ok {
		return fmt.Errorf("prompt must be a string")
	}
	return nil
}

// compile-time interface check
var _ tools.Tool = (*ImageGenerator)(nil)
var _ tools.ToolWithValidation = (*ImageGenerator)(nil)
