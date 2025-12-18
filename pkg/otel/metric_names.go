package otel

// 预定义的指标名称
// 遵循 OpenTelemetry 语义约定
const (
	// Agent 指标
	MetricAgentRuns           = "agent.runs"               // 计数器: Agent 执行次数
	MetricAgentRunDuration    = "agent.run.duration"       // 直方图: Agent 执行时间(ms)
	MetricAgentIterations     = "agent.iterations"         // 直方图: Agent 迭代次数
	MetricAgentErrors         = "agent.errors"             // 计数器: Agent 错误次数
	MetricAgentActive         = "agent.active"             // 仪表: 活跃 Agent 数量

	// LLM 指标
	MetricLLMRequests         = "llm.requests"             // 计数器: LLM 请求次数
	MetricLLMRequestDuration  = "llm.request.duration"     // 直方图: LLM 请求时间(ms)
	MetricLLMTokensPrompt     = "llm.tokens.prompt"        // 计数器: Prompt Token 总数
	MetricLLMTokensCompletion = "llm.tokens.completion"    // 计数器: Completion Token 总数
	MetricLLMTokensTotal      = "llm.tokens.total"         // 计数器: 总 Token 数
	MetricLLMErrors           = "llm.errors"               // 计数器: LLM 错误次数
	MetricLLMRetries          = "llm.retries"              // 计数器: LLM 重试次数

	// Tool 指标
	MetricToolCalls           = "tool.calls"               // 计数器: 工具调用次数
	MetricToolCallDuration    = "tool.call.duration"       // 直方图: 工具调用时间(ms)
	MetricToolErrors          = "tool.errors"              // 计数器: 工具错误次数

	// Memory 指标
	MetricMemoryOperations    = "memory.operations"        // 计数器: 记忆操作次数
	MetricMemorySize          = "memory.size"              // 仪表: 记忆大小
	MetricMemoryHits          = "memory.hits"              // 计数器: 记忆命中次数
	MetricMemoryMisses        = "memory.misses"            // 计数器: 记忆未命中次数

	// RAG 指标
	MetricRAGQueries          = "rag.queries"              // 计数器: RAG 查询次数
	MetricRAGQueryDuration    = "rag.query.duration"       // 直方图: RAG 查询时间(ms)
	MetricRAGDocumentsLoaded  = "rag.documents.loaded"     // 计数器: 加载文档数
	MetricRAGChunksIndexed    = "rag.chunks.indexed"       // 计数器: 索引块数
)

// MetricUnit 指标单位
type MetricUnit string

const (
	UnitNone         MetricUnit = ""
	UnitMilliseconds MetricUnit = "ms"
	UnitSeconds      MetricUnit = "s"
	UnitBytes        MetricUnit = "By"
	UnitCount        MetricUnit = "1"
)

// MetricDescription 指标描述
type MetricDescription struct {
	Name        string
	Description string
	Unit        MetricUnit
	Type        string // counter, histogram, gauge
}

// PredefinedMetrics 预定义指标列表
var PredefinedMetrics = []MetricDescription{
	{MetricAgentRuns, "Number of agent runs", UnitCount, "counter"},
	{MetricAgentRunDuration, "Duration of agent runs", UnitMilliseconds, "histogram"},
	{MetricAgentIterations, "Number of iterations per agent run", UnitCount, "histogram"},
	{MetricAgentErrors, "Number of agent errors", UnitCount, "counter"},
	{MetricAgentActive, "Number of active agents", UnitCount, "gauge"},

	{MetricLLMRequests, "Number of LLM requests", UnitCount, "counter"},
	{MetricLLMRequestDuration, "Duration of LLM requests", UnitMilliseconds, "histogram"},
	{MetricLLMTokensPrompt, "Number of prompt tokens", UnitCount, "counter"},
	{MetricLLMTokensCompletion, "Number of completion tokens", UnitCount, "counter"},
	{MetricLLMTokensTotal, "Total number of tokens", UnitCount, "counter"},
	{MetricLLMErrors, "Number of LLM errors", UnitCount, "counter"},
	{MetricLLMRetries, "Number of LLM retries", UnitCount, "counter"},

	{MetricToolCalls, "Number of tool calls", UnitCount, "counter"},
	{MetricToolCallDuration, "Duration of tool calls", UnitMilliseconds, "histogram"},
	{MetricToolErrors, "Number of tool errors", UnitCount, "counter"},

	{MetricMemoryOperations, "Number of memory operations", UnitCount, "counter"},
	{MetricMemorySize, "Size of memory", UnitCount, "gauge"},
	{MetricMemoryHits, "Number of memory cache hits", UnitCount, "counter"},
	{MetricMemoryMisses, "Number of memory cache misses", UnitCount, "counter"},

	{MetricRAGQueries, "Number of RAG queries", UnitCount, "counter"},
	{MetricRAGQueryDuration, "Duration of RAG queries", UnitMilliseconds, "histogram"},
	{MetricRAGDocumentsLoaded, "Number of documents loaded", UnitCount, "counter"},
	{MetricRAGChunksIndexed, "Number of chunks indexed", UnitCount, "counter"},
}
