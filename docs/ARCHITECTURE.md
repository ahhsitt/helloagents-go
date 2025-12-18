# HelloAgents Go - Architecture Documentation

## Overview

HelloAgents Go 是一个模块化、可扩展的 Go 语言 AI Agent 框架，提供多种 Agent 模式、工具系统、记忆系统和 RAG 能力。

## 项目结构

```
HelloAgents/
├── pkg/                           # 核心包
│   ├── core/                      # 核心抽象层
│   │   ├── llm/                   # LLM 提供商接口和实现
│   │   ├── message/               # 消息类型定义
│   │   ├── config/                # 配置管理
│   │   └── errors/                # 统一错误处理
│   ├── agents/                    # Agent 实现
│   │   ├── agent.go               # Agent 接口定义
│   │   ├── simple.go              # SimpleAgent - 基础对话 Agent
│   │   ├── react.go               # ReActAgent - 推理+行动模式
│   │   ├── reflection.go          # ReflectionAgent - 自我反思 Agent
│   │   └── plansolve.go           # PlanAndSolveAgent - 计划执行模式
│   ├── tools/                     # 工具系统
│   │   ├── tool.go                # Tool 接口
│   │   ├── registry.go            # 工具注册表
│   │   ├── executor.go            # 工具执行器（带超时和重试）
│   │   ├── func_tool.go           # 函数式工具创建
│   │   └── builtin/               # 内置工具
│   ├── memory/                    # 记忆系统
│   │   ├── memory.go              # 记忆接口定义
│   │   ├── working.go             # 工作记忆（对话历史）
│   │   ├── episodic.go            # 情景记忆（事件存储）
│   │   └── semantic.go            # 语义记忆（向量搜索）
│   ├── rag/                       # RAG 管道
│   │   ├── pipeline.go            # RAG 流程编排
│   │   ├── document.go            # 文档和分块类型
│   │   ├── chunker.go             # 文档分块策略
│   │   ├── store.go               # 向量存储实现
│   │   └── retriever.go           # 检索策略
│   └── otel/                      # OpenTelemetry 可观测性
│       ├── provider.go            # 可观测性提供者
│       ├── tracer.go              # 分布式追踪
│       ├── metrics.go             # 指标收集
│       └── config.go              # OTel 配置
├── examples/                      # 示例应用
│   ├── simple/                    # 基础聊天机器人
│   ├── react/                     # ReAct Agent 工具调用
│   ├── memory/                    # 记忆系统演示
│   └── rag/                       # RAG 文档问答
└── tests/                         # 测试套件
    ├── unit/                      # 单元测试
    └── integration/               # 集成测试
```

---

## 核心架构

### 分层架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│                    (examples/, your apps)                    │
├─────────────────────────────────────────────────────────────┤
│                        Agent Layer                           │
│         SimpleAgent │ ReActAgent │ ReflectionAgent           │
│                     │            │ PlanAndSolveAgent         │
├─────────────────────────────────────────────────────────────┤
│      Tool System    │   Memory System   │   RAG Pipeline     │
│   Registry/Executor │ Working/Episodic  │ Chunker/Retriever  │
│   FuncTool/Builtin  │ Semantic Memory   │ VectorStore        │
├─────────────────────────────────────────────────────────────┤
│                      Core Abstractions                       │
│           LLM Provider │ Message │ Config │ Errors           │
├─────────────────────────────────────────────────────────────┤
│                    Observability Layer                       │
│              Tracer │ Metrics │ Logger (OpenTelemetry)       │
└─────────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. LLM 提供商层 (`pkg/core/llm`)

#### Provider 接口

```go
type Provider interface {
    // 同步生成响应
    Generate(ctx context.Context, req Request) (Response, error)
    // 流式生成响应
    GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, <-chan error)
    // 生成文本嵌入向量
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    // 提供商名称
    Name() string
    // 当前模型
    Model() string
    // 关闭连接
    Close() error
}
```

#### 支持的提供商

| 提供商 | 实现文件 | 特性 |
|--------|----------|------|
| OpenAI | `openai.go` | GPT-4o, GPT-4, GPT-3.5, 流式支持 |
| Ollama | `ollama.go` | 本地模型运行 |
| Qwen | `qwen.go` | 阿里通义千问 |
| DeepSeek | `deepseek.go` | DeepSeek 模型 |
| vLLM | `vllm.go` | vLLM 服务器支持 |

#### 高级功能

- **FallbackProvider**: 多提供商降级策略
- **RetryProvider**: 自动重试机制（指数退避）
- **HealthcheckProvider**: 健康检查包装器
- **TracedProvider**: OpenTelemetry 追踪包装器

---

### 2. 消息系统 (`pkg/core/message`)

#### 消息结构

```go
type Message struct {
    ID         string                 // 消息唯一标识
    Role       Role                   // system, user, assistant, tool
    Content    string                 // 消息内容
    Name       string                 // 函数/工具名称
    ToolCalls  []ToolCall             // 工具调用列表
    ToolCallID string                 // 工具调用响应 ID
    Metadata   map[string]interface{} // 扩展元数据
    Timestamp  time.Time              // 时间戳
}

type ToolCall struct {
    ID        string                 // 调用 ID
    Name      string                 // 工具名称
    Arguments map[string]interface{} // 调用参数
}
```

#### 消息构建器

```go
msg := message.NewUserMessage("Hello")
msg := message.NewAssistantMessage("Hi there!")
msg := message.NewSystemMessage("You are a helpful assistant.")
msg := message.NewToolMessage("tool-call-id", "Result: 42")
```

---

### 3. Agent 系统 (`pkg/agents`)

#### Agent 接口

```go
type Agent interface {
    // 执行 Agent 任务
    Run(ctx context.Context, input Input) (Output, error)
    // 流式执行
    RunStream(ctx context.Context, input Input) (<-chan StreamChunk, <-chan error)
    // Agent 名称
    Name() string
    // 获取配置
    Config() config.AgentConfig
}
```

#### Agent 类型

##### SimpleAgent - 基础对话 Agent

```go
agent, _ := agents.NewSimple(provider,
    agents.WithName("ChatBot"),
    agents.WithSystemPrompt("You are a helpful assistant."),
)
output, _ := agent.Run(ctx, agents.Input{Query: "Hello"})
```

**特点**:
- 多轮对话支持
- 对话历史管理
- 流式响应支持

##### ReActAgent - 推理行动 Agent

```go
agent, _ := agents.NewReAct(provider, registry,
    agents.WithMaxIterations(10),
)
output, _ := agent.Run(ctx, agents.Input{Query: "Calculate 123 * 456"})
```

**特点**:
- Thought-Action-Observation 循环
- 工具调用支持
- 迭代推理

##### ReflectionAgent - 自我反思 Agent

```go
agent, _ := agents.NewReflection(provider,
    agents.WithMaxIterations(3),
)
```

**特点**:
- 生成-反思-改进循环
- 质量导向推理

##### PlanAndSolveAgent - 计划执行 Agent

```go
agent, _ := agents.NewPlanAndSolve(provider,
    agents.WithMaxIterations(5),
)
```

**特点**:
- 问题分解
- 步骤化执行
- 结果综合

#### Agent 执行流程

```
┌─────────────────────────────────────────┐
│ 用户输入 (Query + Context)              │
└──────────────────┬──────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│ 构建消息列表                             │
│ - 系统提示词                             │
│ - 对话历史                               │
│ - 当前用户消息                           │
│ - 工具定义 (如有)                        │
└──────────────────┬───────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────┐
│ LLM 生成                                 │
│ - 输入: 消息 + 工具定义                  │
│ - 输出: 内容 + 工具调用                  │
└──────────────────┬───────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
   无工具调用            有工具调用
        │                     │
        ▼                     ▼
   返回最终答案          执行工具
                         更新消息
                         继续循环
```

---

### 4. 工具系统 (`pkg/tools`)

#### Tool 接口

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() ParameterSchema
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

type ParameterSchema struct {
    Type       string                      // "object"
    Properties map[string]PropertySchema   // 参数属性
    Required   []string                    // 必需参数
}
```

#### 工具创建方式

##### FuncTool - 完整功能工具

```go
weatherTool := tools.NewFuncTool(
    "get_weather",
    "Get current weather for a location",
    tools.ParameterSchema{
        Type: "object",
        Properties: map[string]tools.PropertySchema{
            "location": {Type: "string", Description: "City name"},
        },
        Required: []string{"location"},
    },
    func(ctx context.Context, args map[string]interface{}) (string, error) {
        location := args["location"].(string)
        return fmt.Sprintf("Weather in %s: Sunny, 25°C", location), nil
    },
)
```

##### SimpleTool - 简单字符串参数工具

```go
greetTool := tools.NewSimpleTool(
    "greet",
    "Greet someone by name",
    "name",
    "Person's name to greet",
    func(ctx context.Context, name string) (string, error) {
        return "Hello, " + name + "!", nil
    },
)
```

#### 工具注册表

```go
registry := tools.NewRegistry()
registry.Register(calculatorTool)
registry.Register(terminalTool)
registry.RegisterAll(tool1, tool2, tool3)

tool, _ := registry.Get("calculator")
allTools := registry.All()
```

#### 工具执行器

```go
executor := tools.NewExecutor(registry,
    tools.WithTimeout(30*time.Second),
    tools.WithMaxRetries(3),
)

result, err := executor.Execute(ctx, "calculator", map[string]interface{}{
    "expression": "123 * 456",
})
```

#### 内置工具

| 工具 | 文件 | 功能 |
|------|------|------|
| Calculator | `builtin/calculator.go` | 数学表达式计算 |
| Terminal | `builtin/terminal.go` | Shell 命令执行 |
| RAG | `builtin/rag.go` | 文档检索 |

---

### 5. 记忆系统 (`pkg/memory`)

#### 记忆接口

```go
// ConversationMemory - 对话记忆
type ConversationMemory interface {
    AddMessage(ctx context.Context, msg message.Message) error
    GetHistory(ctx context.Context, limit int) ([]message.Message, error)
    GetRecentHistory(ctx context.Context, n int) ([]message.Message, error)
    Clear(ctx context.Context) error
    Size() int
}

// VectorMemory - 向量记忆
type VectorMemory interface {
    Store(ctx context.Context, id, content string, metadata map[string]interface{}) error
    Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
    Delete(ctx context.Context, id string) error
    Clear(ctx context.Context) error
    Size() int
}

// EpisodicMemory - 情景记忆
type EpisodicMemory interface {
    AddEpisode(ctx context.Context, episode Episode) error
    GetEpisodes(ctx context.Context, filter *EpisodeFilter) ([]Episode, error)
    GetByType(ctx context.Context, episodeType string, limit int) ([]Episode, error)
    GetMostImportant(ctx context.Context, limit int) ([]Episode, error)
    GetByTimeRange(ctx context.Context, start, end int64) ([]Episode, error)
    Clear(ctx context.Context) error
    Size() int
}
```

#### 记忆实现

##### WorkingMemory - 工作记忆

```go
mem := memory.NewWorkingMemory(
    memory.WithMaxSize(100),           // 最大消息数
    memory.WithTokenLimit(4000),       // Token 限制
    memory.WithTTL(30*time.Minute),    // 过期时间
)

mem.AddMessage(ctx, message.NewUserMessage("Hello"))
history, _ := mem.GetHistory(ctx, 10)
```

**特点**:
- LRU 淘汰策略
- Token 限制截断
- TTL 过期机制

##### SemanticMemory - 语义记忆

```go
embedder := memory.NewOpenAIEmbedder(apiKey)
mem := memory.NewSemanticMemory(embedder)

mem.Store(ctx, "id-1", "User prefers dark mode", nil)
results, _ := mem.Search(ctx, "color preference", 5)
```

**特点**:
- 向量相似度搜索
- 余弦相似度计算
- 分数阈值过滤

##### EpisodicMemory - 情景记忆

```go
mem := memory.NewEpisodicMemory()

mem.AddEpisode(ctx, memory.Episode{
    Type:       "user_action",
    Content:    "User requested weather info",
    Importance: 0.8,
})

important, _ := mem.GetMostImportant(ctx, 5)
```

---

### 6. RAG 管道 (`pkg/rag`)

#### RAG 架构

```
文档 → 加载器 → 分块器 → 嵌入器 → 向量存储
                                      ↓
查询 → 嵌入 → 检索器 → 向量搜索 → 排序/过滤
                                      ↓
                              答案生成器 → 响应
```

#### 核心类型

```go
// 文档
type Document struct {
    ID       string
    Content  string
    Metadata DocumentMetadata
}

// 文档分块
type DocumentChunk struct {
    ID          string
    DocumentID  string
    Content     string
    Index       int
    StartOffset int
    EndOffset   int
    Vector      []float32
}

// 检索结果
type RetrievalResult struct {
    Chunk DocumentChunk
    Score float32
}
```

#### 文档分块器

```go
// 递归字符分块
chunker := rag.NewRecursiveCharacterChunker(512, 50)
chunks := chunker.Chunk(document)

// 句子分块
chunker := rag.NewSentenceChunker(500, 100)
chunks := chunker.Chunk(document)
```

#### 向量存储

```go
store := rag.NewInMemoryVectorStore()
store.Add(ctx, chunks)
results, _ := store.Search(ctx, queryVector, 5)
```

#### 检索器

```go
// 基础向量检索
retriever := rag.NewVectorRetriever(store, embedder,
    rag.WithScoreThreshold(0.7),
)

// 多源检索
multiRetriever := rag.NewMultiRetriever(
    []rag.Retriever{retriever1, retriever2},
    &rag.ScoreBasedMerger{},
)

// 重排序检索
rerankRetriever := rag.NewRerankRetriever(
    baseRetriever,
    reranker,
    100, // 初始获取数量
)
```

#### RAG Pipeline

```go
pipeline := rag.NewPipeline(embedder, store, generator)

// 文档摄取
pipeline.Ingest(ctx, documents)

// 问答
response, _ := pipeline.Query(ctx, "What is the capital of France?")
fmt.Println(response.Answer)
fmt.Println(response.Sources)
```

---

### 7. 可观测性 (`pkg/otel`)

#### 提供者配置

```go
config := otel.Config{
    Enabled:        true,
    ServiceName:    "my-agent",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    Tracing: otel.TracingConfig{
        Enabled:    true,
        Endpoint:   "localhost:4317",
        SampleRate: 1.0,
    },
    Metrics: otel.MetricsConfig{
        Enabled:  true,
        Endpoint: "localhost:4317",
        Interval: 60 * time.Second,
    },
}
```

#### Tracer 使用

```go
tracer := otel.NewTracer(otelTracer)

ctx, span := tracer.Start(ctx, "agent.run",
    otel.WithSpanKind(otel.SpanKindServer),
)
defer span.End()

span.SetAttributes(
    attribute.String("agent.name", "MyAgent"),
    attribute.String("query", input.Query),
)
span.AddEvent("tool_called", attribute.String("tool", "calculator"))
```

#### Metrics 使用

```go
metrics := otel.NewInMemoryMetrics()

// 计数器
counter := metrics.Counter("agent.requests")
counter.Add(ctx, 1, otel.NewAttr("agent", "simple"))

// 直方图
histogram := metrics.Histogram("agent.latency")
histogram.Record(ctx, duration.Seconds())

// 仪表
gauge := metrics.Gauge("agent.active_sessions")
gauge.Set(ctx, 10)
```

#### TracedProvider 包装

```go
tracedProvider := otel.NewTracedProvider(provider, tracer, metrics)
// 自动追踪所有 LLM 调用
```

---

## 数据流

### ReAct Agent 完整执行流程

```
┌─────────────────────────────────────────────────────────────┐
│                        用户输入                              │
│                   Query: "123 乘以 456"                      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                     ReActAgent.Run()                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 1. 构建初始消息                                       │    │
│  │    - System: "You are a helpful assistant..."       │    │
│  │    - User: "123 乘以 456"                            │    │
│  │    - Tools: [Calculator, Terminal]                  │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      迭代 1/10                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ LLM.Generate()                                       │    │
│  │ Response:                                            │    │
│  │   Content: "I'll calculate this for you."           │    │
│  │   ToolCalls: [{name: "calculator",                  │    │
│  │                args: {expression: "123 * 456"}}]    │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 记录推理步骤                                          │    │
│  │   StepType: Thought                                  │    │
│  │   Content: "I'll calculate this for you."           │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 执行工具                                              │    │
│  │   Tool: Calculator                                   │    │
│  │   Input: {expression: "123 * 456"}                  │    │
│  │   Output: "56088"                                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 记录观察步骤                                          │    │
│  │   StepType: Observation                              │    │
│  │   Content: "56088"                                   │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      迭代 2/10                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ LLM.Generate() - 包含工具结果                         │    │
│  │ Response:                                            │    │
│  │   Content: "123 乘以 456 等于 56088"                  │    │
│  │   ToolCalls: [] (无工具调用)                          │    │
│  └─────────────────────────────────────────────────────┘    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                        返回结果                              │
│  Output {                                                    │
│    Response: "123 乘以 456 等于 56088"                       │
│    Steps: [Thought, Action, Observation, Thought]           │
│    TokenUsage: {Prompt: 150, Completion: 50, Total: 200}   │
│    Duration: 1.2s                                           │
│  }                                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 配置管理

### 环境变量

```bash
# LLM 配置
HELLOAGENTS_LLM_API_KEY=sk-xxx
HELLOAGENTS_LLM_MODEL=gpt-4o
HELLOAGENTS_LLM_BASE_URL=https://api.openai.com/v1
HELLOAGENTS_LLM_TIMEOUT=30s
HELLOAGENTS_LLM_MAX_RETRIES=3

# Agent 配置
HELLOAGENTS_AGENT_MAX_ITERATIONS=10
HELLOAGENTS_AGENT_TEMPERATURE=0.7
HELLOAGENTS_AGENT_MAX_TOKENS=4096
HELLOAGENTS_AGENT_TIMEOUT=5m

# 可观测性配置
HELLOAGENTS_OBSERVABILITY_ENABLED=true
HELLOAGENTS_OBSERVABILITY_TRACING_ENDPOINT=localhost:4317
HELLOAGENTS_OBSERVABILITY_METRICS_ENDPOINT=localhost:4317
```

### 配置文件 (YAML)

```yaml
llm:
  api_key: ${OPENAI_API_KEY}
  model: gpt-4o
  timeout: 30s
  max_retries: 3

agent:
  max_iterations: 10
  temperature: 0.7
  max_tokens: 4096
  timeout: 5m

observability:
  enabled: true
  service_name: my-agent
  tracing:
    enabled: true
    endpoint: localhost:4317
    sample_rate: 1.0
  metrics:
    enabled: true
    endpoint: localhost:4317
```

---

## 并发和线程安全

### 同步机制

- **Agent 历史存储**: `sync.RWMutex`
- **工具注册表**: `sync.RWMutex`
- **记忆系统**: `sync.RWMutex`
- **向量存储**: `sync.RWMutex`
- **指标收集**: `sync.RWMutex`

### Channel 使用

```go
// 流式响应
chunkCh, errCh := agent.RunStream(ctx, input)
for chunk := range chunkCh {
    fmt.Print(chunk.Content)
}
if err := <-errCh; err != nil {
    log.Fatal(err)
}
```

### Context 传播

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

output, err := agent.Run(ctx, input)
// 所有异步操作都尊重 context 取消
```

---

## 错误处理

### 错误类型

```go
// LLM 错误
errors.ErrInvalidAPIKey
errors.ErrRateLimited
errors.ErrTimeout
errors.ErrTokenLimitExceeded

// Agent 错误
errors.ErrMaxIterationsExceeded
errors.ErrAgentNotReady
errors.ErrNoToolsAvailable

// 工具错误
errors.ErrToolNotFound
errors.ErrToolExecutionFailed
errors.ErrInvalidToolArgs

// 记忆错误
errors.ErrMemoryFull
errors.ErrMemoryNotFound

// RAG 错误
errors.ErrDocumentNotFound
errors.ErrEmbeddingFailed
```

### 错误处理策略

```go
// 检查可重试错误
if errors.IsRetryable(err) {
    // 执行重试逻辑
}

// 检查致命错误
if errors.IsFatal(err) {
    // 终止操作
}

// 错误包装
return errors.WrapError(err, "failed to execute tool")
```

---

## 扩展点

### 自定义 LLM Provider

```go
type MyProvider struct{}

func (p *MyProvider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
    // 实现你的 LLM 调用逻辑
}

func (p *MyProvider) GenerateStream(ctx context.Context, req llm.Request) (<-chan llm.StreamChunk, <-chan error) {
    // 实现流式响应
}

func (p *MyProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // 实现嵌入生成
}
```

### 自定义 Tool

```go
type MyTool struct{}

func (t *MyTool) Name() string        { return "my_tool" }
func (t *MyTool) Description() string { return "My custom tool" }
func (t *MyTool) Parameters() tools.ParameterSchema {
    return tools.ParameterSchema{
        Type: "object",
        Properties: map[string]tools.PropertySchema{
            "input": {Type: "string", Description: "Input value"},
        },
        Required: []string{"input"},
    }
}
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // 实现工具逻辑
}
```

### 自定义 Memory

```go
type MyMemory struct{}

func (m *MyMemory) AddMessage(ctx context.Context, msg message.Message) error {
    // 实现消息存储
}

func (m *MyMemory) GetHistory(ctx context.Context, limit int) ([]message.Message, error) {
    // 实现历史检索
}
```

### 自定义 Retriever

```go
type MyRetriever struct{}

func (r *MyRetriever) Retrieve(ctx context.Context, query string, topK int) ([]rag.RetrievalResult, error) {
    // 实现检索逻辑
}
```

---

## 测试

### 单元测试结构

```
tests/unit/
├── agents/
│   └── simple_test.go
├── llm/
│   └── openai_test.go
├── memory/
│   ├── working_test.go
│   ├── episodic_test.go
│   └── semantic_test.go
├── tools/
│   ├── registry_test.go
│   └── func_tool_test.go
├── otel/
│   ├── tracer_test.go
│   ├── metrics_test.go
│   └── config_test.go
└── rag/
    ├── chunker_test.go
    ├── store_test.go
    ├── retriever_test.go
    └── document_test.go
```

### 运行测试

```bash
# 运行所有单元测试
go test ./tests/unit/...

# 带覆盖率
go test ./tests/unit/... -coverpkg=./pkg/...

# 详细输出
go test ./tests/unit/... -v
```

---

## 性能考虑

### 连接管理
- HTTP 连接复用
- 可配置超时
- 连接池优化

### 内存管理
- 对话历史容量限制
- LRU 淘汰策略
- TTL 过期机制

### 并发控制
- RWMutex 用于读多写少场景
- Channel 用于流式数据
- Context 用于超时控制

---

## 版本兼容性

- **Go 版本**: 1.23+
- **OpenTelemetry**: 1.x
- **OpenAI API**: 兼容 v1 API

---

## 相关文档

- [快速开始](../README.md)
- [示例代码](../examples/)
- [API 参考](./API.md)
- [设计规格](../specs/001-golang-agent-framework/)
