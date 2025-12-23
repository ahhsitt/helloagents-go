package rag

import (
	"context"
)

// QueryTransformer 查询变换器接口
// 将原始查询转换为一个或多个检索查询
type QueryTransformer interface {
	// Transform 变换查询
	// 输入单个查询，输出一个或多个变换后的查询
	Transform(ctx context.Context, query string) ([]TransformedQuery, error)
}

// TransformedQuery 变换后的查询
type TransformedQuery struct {
	// Query 变换后的查询文本
	Query string
	// Weight 权重（用于融合时加权，默认 1.0）
	Weight float32
	// Metadata 元数据（如来源变换器名称）
	Metadata map[string]string
}

// NewTransformedQuery 创建变换后的查询
func NewTransformedQuery(query string) TransformedQuery {
	return TransformedQuery{
		Query:  query,
		Weight: 1.0,
	}
}

// WithWeight 设置权重
func (q TransformedQuery) WithWeight(weight float32) TransformedQuery {
	q.Weight = weight
	return q
}

// WithMetadata 设置元数据
func (q TransformedQuery) WithMetadata(key, value string) TransformedQuery {
	if q.Metadata == nil {
		q.Metadata = make(map[string]string)
	}
	q.Metadata[key] = value
	return q
}

// LLMProvider LLM 提供者接口（用于查询变换）
type LLMProvider interface {
	// Generate 生成文本
	Generate(ctx context.Context, prompt string) (string, error)
}

// MultiQueryConfig MQE 配置
type MultiQueryConfig struct {
	// NumQueries 扩展查询数量，默认 3
	NumQueries int
	// IncludeOriginal 是否包含原始查询，默认 true
	IncludeOriginal bool
	// Prompt 自定义提示模板（可选）
	Prompt string
}

// DefaultMultiQueryConfig 默认 MQE 配置
func DefaultMultiQueryConfig() MultiQueryConfig {
	return MultiQueryConfig{
		NumQueries:      3,
		IncludeOriginal: true,
	}
}

// MultiQueryTransformer 多查询扩展变换器 (MQE)
type MultiQueryTransformer struct {
	llm    LLMProvider
	config MultiQueryConfig
}

// MultiQueryTransformerOption MQE 变换器选项
type MultiQueryTransformerOption func(*MultiQueryTransformer)

// WithNumQueries 设置扩展查询数量
func WithNumQueries(n int) MultiQueryTransformerOption {
	return func(t *MultiQueryTransformer) {
		t.config.NumQueries = n
	}
}

// WithIncludeOriginal 设置是否包含原始查询
func WithIncludeOriginal(include bool) MultiQueryTransformerOption {
	return func(t *MultiQueryTransformer) {
		t.config.IncludeOriginal = include
	}
}

// WithMQEPrompt 设置自定义提示模板
func WithMQEPrompt(prompt string) MultiQueryTransformerOption {
	return func(t *MultiQueryTransformer) {
		t.config.Prompt = prompt
	}
}

// NewMultiQueryTransformer 创建多查询扩展变换器
func NewMultiQueryTransformer(llm LLMProvider, opts ...MultiQueryTransformerOption) *MultiQueryTransformer {
	t := &MultiQueryTransformer{
		llm:    llm,
		config: DefaultMultiQueryConfig(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// DefaultMQEPrompt 默认 MQE 提示模板
const DefaultMQEPrompt = `你是一个查询扩展助手。给定一个用户查询，生成 %d 个语义相关但表达不同的查询变体。
这些变体应该从不同角度覆盖用户可能的查询意图，有助于检索更全面的相关文档。

用户查询: %s

请直接输出 %d 个查询变体，每行一个，不要添加编号或其他格式：`

// Transform 执行多查询扩展
func (t *MultiQueryTransformer) Transform(ctx context.Context, query string) ([]TransformedQuery, error) {
	var results []TransformedQuery

	// 如果包含原始查询，先添加
	if t.config.IncludeOriginal {
		results = append(results, NewTransformedQuery(query).WithMetadata("source", "original"))
	}

	// 使用 LLM 生成扩展查询
	prompt := t.config.Prompt
	if prompt == "" {
		prompt = DefaultMQEPrompt
	}

	// 格式化提示
	formattedPrompt := formatMQEPrompt(prompt, query, t.config.NumQueries)

	response, err := t.llm.Generate(ctx, formattedPrompt)
	if err != nil {
		// 如果 LLM 调用失败但已有原始查询，降级返回
		if len(results) > 0 {
			return results, nil
		}
		return nil, err
	}

	// 解析生成的查询
	expandedQueries := parseMQEResponse(response, t.config.NumQueries)
	for _, q := range expandedQueries {
		results = append(results, NewTransformedQuery(q).WithMetadata("source", "mqe"))
	}

	return results, nil
}

// formatMQEPrompt 格式化 MQE 提示
func formatMQEPrompt(template, query string, numQueries int) string {
	// 简单的格式化，支持 %d 和 %s
	result := template
	// 替换第一个 %d
	for i := 0; i < 2; i++ {
		result = replaceFirst(result, "%d", intToString(numQueries))
	}
	result = replaceFirst(result, "%s", query)
	return result
}

// parseMQEResponse 解析 MQE 响应
func parseMQEResponse(response string, maxQueries int) []string {
	var queries []string
	lines := splitLines(response)
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" {
			continue
		}
		// 移除可能的编号前缀
		line = removeNumberPrefix(line)
		if line != "" {
			queries = append(queries, line)
		}
		if len(queries) >= maxQueries {
			break
		}
	}
	return queries
}

// HyDEConfig HyDE 配置
type HyDEConfig struct {
	// MaxTokens 假设文档最大 Token 数（提示中使用）
	MaxTokens int
	// Prompt 自定义提示模板（可选）
	Prompt string
}

// DefaultHyDEConfig 默认 HyDE 配置
func DefaultHyDEConfig() HyDEConfig {
	return HyDEConfig{
		MaxTokens: 512,
	}
}

// HyDETransformer 假设文档嵌入变换器
type HyDETransformer struct {
	llm    LLMProvider
	config HyDEConfig
}

// HyDETransformerOption HyDE 变换器选项
type HyDETransformerOption func(*HyDETransformer)

// WithHyDEMaxTokens 设置最大 Token 数
func WithHyDEMaxTokens(maxTokens int) HyDETransformerOption {
	return func(t *HyDETransformer) {
		t.config.MaxTokens = maxTokens
	}
}

// WithHyDEPrompt 设置自定义提示模板
func WithHyDEPrompt(prompt string) HyDETransformerOption {
	return func(t *HyDETransformer) {
		t.config.Prompt = prompt
	}
}

// NewHyDETransformer 创建 HyDE 变换器
func NewHyDETransformer(llm LLMProvider, opts ...HyDETransformerOption) *HyDETransformer {
	t := &HyDETransformer{
		llm:    llm,
		config: DefaultHyDEConfig(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// DefaultHyDEPrompt 默认 HyDE 提示模板
const DefaultHyDEPrompt = `请为以下问题写一段可能的答案文档。这个文档应该是信息丰富的，就像是从权威来源摘录的内容。
不需要完美准确，但应该包含与问题相关的关键概念和术语。

问题: %s

请直接输出答案文档（约 %d 字）：`

// Transform 执行 HyDE 变换
func (t *HyDETransformer) Transform(ctx context.Context, query string) ([]TransformedQuery, error) {
	prompt := t.config.Prompt
	if prompt == "" {
		prompt = DefaultHyDEPrompt
	}

	// 格式化提示
	formattedPrompt := formatHyDEPrompt(prompt, query, t.config.MaxTokens)

	response, err := t.llm.Generate(ctx, formattedPrompt)
	if err != nil {
		// HyDE 失败时降级为原始查询
		return []TransformedQuery{
			NewTransformedQuery(query).WithMetadata("source", "fallback"),
		}, nil
	}

	// 清理响应
	hypotheticalDoc := trimSpace(response)
	if hypotheticalDoc == "" {
		return []TransformedQuery{
			NewTransformedQuery(query).WithMetadata("source", "fallback"),
		}, nil
	}

	return []TransformedQuery{
		NewTransformedQuery(hypotheticalDoc).WithMetadata("source", "hyde"),
	}, nil
}

// formatHyDEPrompt 格式化 HyDE 提示
func formatHyDEPrompt(template, query string, maxTokens int) string {
	result := template
	result = replaceFirst(result, "%s", query)
	result = replaceFirst(result, "%d", intToString(maxTokens))
	return result
}

// 辅助函数

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func splitLines(s string) []string {
	var lines []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else if s[i] != '\r' {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func removeNumberPrefix(s string) string {
	// 移除类似 "1." "1)" 等前缀
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) {
		if s[i] == '.' || s[i] == ')' || s[i] == ':' {
			return trimSpace(s[i+1:])
		}
	}
	return s
}

// compile-time interface check
var _ QueryTransformer = (*MultiQueryTransformer)(nil)
var _ QueryTransformer = (*HyDETransformer)(nil)
