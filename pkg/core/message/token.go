package message

// TokenUsage 表示 Token 使用统计
type TokenUsage struct {
	// PromptTokens 输入 Token 数
	PromptTokens int `json:"prompt_tokens"`
	// CompletionTokens 输出 Token 数
	CompletionTokens int `json:"completion_tokens"`
	// TotalTokens 总 Token 数
	TotalTokens int `json:"total_tokens"`
}

// Add 累加 Token 使用量
func (t *TokenUsage) Add(other TokenUsage) {
	t.PromptTokens += other.PromptTokens
	t.CompletionTokens += other.CompletionTokens
	t.TotalTokens += other.TotalTokens
}

// IsEmpty 检查是否为空
func (t *TokenUsage) IsEmpty() bool {
	return t.TotalTokens == 0
}
