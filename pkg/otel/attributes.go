package otel

import "go.opentelemetry.io/otel/attribute"

// 预定义的语义属性键
// 遵循 OpenTelemetry 语义约定
const (
	// Agent 相关属性
	AttrAgentName        = "agent.name"
	AttrAgentType        = "agent.type"
	AttrAgentIteration   = "agent.iteration"
	AttrAgentMaxIter     = "agent.max_iterations"

	// LLM 相关属性
	AttrLLMProvider      = "llm.provider"
	AttrLLMModel         = "llm.model"
	AttrLLMTemperature   = "llm.temperature"
	AttrLLMMaxTokens     = "llm.max_tokens"
	AttrLLMPromptTokens  = "llm.prompt_tokens"
	AttrLLMCompletionTokens = "llm.completion_tokens"
	AttrLLMTotalTokens   = "llm.total_tokens"

	// Tool 相关属性
	AttrToolName         = "tool.name"
	AttrToolArgs         = "tool.arguments"
	AttrToolResult       = "tool.result"
	AttrToolError        = "tool.error"
	AttrToolDuration     = "tool.duration_ms"

	// Message 相关属性
	AttrMessageRole      = "message.role"
	AttrMessageContent   = "message.content"
	AttrMessageTokens    = "message.tokens"

	// Memory 相关属性
	AttrMemoryType       = "memory.type"
	AttrMemoryCapacity   = "memory.capacity"
	AttrMemoryUsage      = "memory.usage"

	// RAG 相关属性
	AttrRAGDocCount      = "rag.document_count"
	AttrRAGChunkCount    = "rag.chunk_count"
	AttrRAGTopK          = "rag.top_k"
	AttrRAGScore         = "rag.score"

	// Error 相关属性
	AttrErrorType        = "error.type"
	AttrErrorMessage     = "error.message"
	AttrErrorRetryable   = "error.retryable"
)

// AgentName 创建 Agent 名称属性
func AgentName(name string) attribute.KeyValue {
	return attribute.String(AttrAgentName, name)
}

// AgentType 创建 Agent 类型属性
func AgentType(typ string) attribute.KeyValue {
	return attribute.String(AttrAgentType, typ)
}

// AgentIteration 创建 Agent 迭代次数属性
func AgentIteration(iter int) attribute.KeyValue {
	return attribute.Int(AttrAgentIteration, iter)
}

// LLMProvider 创建 LLM 提供商属性
func LLMProvider(provider string) attribute.KeyValue {
	return attribute.String(AttrLLMProvider, provider)
}

// LLMModel 创建 LLM 模型属性
func LLMModel(model string) attribute.KeyValue {
	return attribute.String(AttrLLMModel, model)
}

// LLMTokens 创建 LLM Token 使用属性
func LLMTokens(prompt, completion, total int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int(AttrLLMPromptTokens, prompt),
		attribute.Int(AttrLLMCompletionTokens, completion),
		attribute.Int(AttrLLMTotalTokens, total),
	}
}

// ToolName 创建工具名称属性
func ToolName(name string) attribute.KeyValue {
	return attribute.String(AttrToolName, name)
}

// ToolDuration 创建工具执行时间属性（毫秒）
func ToolDuration(ms int64) attribute.KeyValue {
	return attribute.Int64(AttrToolDuration, ms)
}

// ErrorAttrs 创建错误属性
func ErrorAttrs(errType, message string, retryable bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrErrorType, errType),
		attribute.String(AttrErrorMessage, message),
		attribute.Bool(AttrErrorRetryable, retryable),
	}
}
