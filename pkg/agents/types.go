package agents

import (
	"time"

	"github.com/easyops/helloagents-go/pkg/core/message"
)

// Input 定义 Agent 的输入结构
type Input struct {
	// Query 用户查询内容（必填）
	Query string `json:"query"`
	// UserID 用户标识（可选，用于个性化）
	UserID string `json:"user_id,omitempty"`
	// SessionID 会话标识（可选，用于多轮对话）
	SessionID string `json:"session_id,omitempty"`
	// Context 额外上下文信息（可选）
	Context map[string]interface{} `json:"context,omitempty"`
}

// Output 定义 Agent 的输出结构
type Output struct {
	// Response 最终响应文本
	Response string `json:"response"`
	// Steps 推理步骤轨迹（ReAct 等模式）
	Steps []ReasoningStep `json:"steps,omitempty"`
	// TokenUsage Token 使用统计
	TokenUsage message.TokenUsage `json:"token_usage"`
	// Duration 总执行时间
	Duration time.Duration `json:"duration"`
	// Error 错误信息（如有）
	Error string `json:"error,omitempty"`
}

// HasError 检查输出是否包含错误
func (o *Output) HasError() bool {
	return o.Error != ""
}

// IsSuccess 检查是否执行成功
func (o *Output) IsSuccess() bool {
	return o.Error == ""
}
