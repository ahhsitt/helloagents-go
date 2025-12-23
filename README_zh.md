# HelloAgents Go

<p align="center">
  <strong>高性能、模块化的 Go 语言 AI Agent 框架</strong>
</p>

<p align="center">
  <a href="#特性">特性</a> |
  <a href="#快速开始">快速开始</a> |
  <a href="#agent-类型">Agent 类型</a> |
  <a href="#文档">文档</a>
</p>

---

HelloAgents Go 是一个生产级的 AI Agent 框架，提供统一、类型安全的接口，用于构建具有多种推理模式、工具集成、记忆系统和 RAG 能力的智能 Agent。

## 特性

### Agent 模式
- **SimpleAgent** - 基础对话 Agent，支持多轮对话
- **ReActAgent** - 推理+行动模式，支持工具调用
- **ReflectionAgent** - 自我反思改进，生成-反思-优化循环
- **PlanAndSolveAgent** - 策略规划，分步执行

### 工具系统
- 灵活的工具注册和发现机制
- 内置工具：计算器、终端、RAG
- 通过 `FuncTool` 和 `SimpleTool` 轻松创建自定义工具
- 工具链和条件执行
- 超时和重试机制

### 记忆系统
- **工作记忆** - 对话历史，支持 LRU 淘汰和 TTL
- **情景记忆** - 事件存储，带时间戳和重要性评分
- **语义记忆** - 向量相似度搜索

### RAG 管道
- 文档加载（文本、Markdown）
- 递归字符分块，支持重叠
- 向量嵌入和存储
- 多种检索策略（相似度、多源、重排序）
- 高级策略：MQE（多查询扩展）、HyDE（假设文档嵌入）
- 结果融合（RRF、基于分数）和后处理管道

### LLM 提供商
- OpenAI（GPT-4o、GPT-4、GPT-3.5）
- Ollama（本地模型）
- 通义千问（Qwen）
- DeepSeek
- vLLM
- 降级和健康检查包装器

### 可观测性
- 完整的 OpenTelemetry 集成
- 分布式追踪，支持 Span 传播
- 指标收集（计数器、直方图、仪表）
- 结构化日志，关联 Trace ID

---

## 快速开始

### 安装

```bash
go get github.com/easyops/helloagents-go
```

### 基础示例

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/easyops/helloagents-go/pkg/agents"
    "github.com/easyops/helloagents-go/pkg/core/llm"
)

func main() {
    ctx := context.Background()

    // 创建 LLM 提供商
    provider, err := llm.NewOpenAI(
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel("gpt-4o-mini"),
    )
    if err != nil {
        panic(err)
    }
    defer provider.Close()

    // 创建 Agent
    agent, err := agents.NewSimple(provider,
        agents.WithName("助手"),
        agents.WithSystemPrompt("你是一个有帮助的 AI 助手。"),
    )
    if err != nil {
        panic(err)
    }

    // 运行
    output, err := agent.Run(ctx, agents.Input{
        Query: "法国的首都是哪里？",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(output.Response)
    fmt.Printf("Token: %d, 耗时: %s\n", output.TokenUsage.TotalTokens, output.Duration)
}
```

### 带工具的 ReAct Agent

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/easyops/helloagents-go/pkg/agents"
    "github.com/easyops/helloagents-go/pkg/core/llm"
    "github.com/easyops/helloagents-go/pkg/tools"
    "github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func main() {
    ctx := context.Background()

    // 创建提供商
    provider, _ := llm.NewOpenAI(
        llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
        llm.WithModel("gpt-4o"),
    )
    defer provider.Close()

    // 设置工具
    registry := tools.NewRegistry()
    registry.Register(builtin.NewCalculator())
    registry.Register(builtin.NewTerminal())

    // 创建 ReAct Agent
    agent, _ := agents.NewReAct(provider, registry,
        agents.WithName("工具Agent"),
        agents.WithMaxIterations(10),
    )

    // 运行带工具调用的任务
    output, _ := agent.Run(ctx, agents.Input{
        Query: "计算 123 * 456，然后列出当前目录的文件。",
    })

    fmt.Println(output.Response)

    // 打印推理步骤
    for _, step := range output.Steps {
        fmt.Printf("[%s] %s\n", step.Type, step.Content)
    }
}
```

---

## Agent 类型

### SimpleAgent

基础对话 Agent，用于问答和多轮对话。

```go
agent, _ := agents.NewSimple(provider,
    agents.WithName("聊天机器人"),
    agents.WithSystemPrompt("你是一个友好的助手。"),
    agents.WithAgentTemperature(0.7),
    agents.WithAgentMaxTokens(2048),
)

// 多轮对话
output1, _ := agent.Run(ctx, agents.Input{Query: "你好，我是小明"})
output2, _ := agent.Run(ctx, agents.Input{Query: "我叫什么名字？"}) // 记住上下文
```

### ReActAgent

推理和行动 Agent，通过工具完成复杂任务。

```go
agent, _ := agents.NewReAct(provider, registry,
    agents.WithName("工具使用者"),
    agents.WithMaxIterations(10),
    agents.WithSystemPrompt("你是一个可以使用工具的助手。"),
)

// Agent 会根据需要进行推理和调用工具
output, _ := agent.Run(ctx, agents.Input{
    Query: "计算半径为5的圆的面积，然后保存到 area.txt 文件",
})
```

### ReflectionAgent

自我改进的 Agent，生成、评估并优化响应。

```go
agent, _ := agents.NewReflection(provider,
    agents.WithMaxIterations(3),
    agents.WithSystemPrompt("你是一个专业的代码审查员。"),
)

output, _ := agent.Run(ctx, agents.Input{
    Query: "写一个判断素数的函数",
})
// Agent 会生成代码，反思质量，然后改进
```

### PlanAndSolveAgent

策略 Agent，创建并执行分步计划。

```go
agent, _ := agents.NewPlanAndSolve(provider, registry,
    agents.WithMaxIterations(5),
)

output, _ := agent.Run(ctx, agents.Input{
    Query: "研究前3个 Go Web 框架并进行比较",
})
// Agent 会创建计划，执行每个步骤，综合结果
```

---

## LLM 提供商

### OpenAI

```go
provider, _ := llm.NewOpenAI(
    llm.WithAPIKey("sk-..."),
    llm.WithModel("gpt-4o"),           // 或 gpt-4o-mini, gpt-4, gpt-3.5-turbo
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(4096),
    llm.WithMaxRetries(3),
)
```

### Ollama（本地）

```go
provider := llm.NewOllamaClient(
    llm.WithOllamaBaseURL("http://localhost:11434"),
    llm.WithOllamaModel("llama3.2"),
)
```

### 通义千问

```go
provider := llm.NewQwenClient(
    llm.WithQwenAPIKey("your-api-key"),
    llm.WithQwenModel("qwen-turbo"),
)
```

### DeepSeek

```go
provider := llm.NewDeepSeekClient(
    llm.WithDeepSeekAPIKey("your-api-key"),
    llm.WithDeepSeekModel("deepseek-chat"),
)
```

### vLLM

```go
provider := llm.NewVLLMClient(
    llm.WithVLLMBaseURL("http://localhost:8000"),
    llm.WithVLLMModel("meta-llama/Llama-2-7b-chat-hf"),
)
```

### 降级提供商

```go
provider := llm.NewFallbackProvider(
    primaryProvider,   // 主提供商
    fallbackProvider,  // 备用提供商
)
```

---

## 工具系统

### 内置工具

```go
import "github.com/easyops/helloagents-go/pkg/tools/builtin"

registry := tools.NewRegistry()

// 计算器 - 计算数学表达式
registry.Register(builtin.NewCalculator())

// 终端 - 执行 Shell 命令（带超时）
registry.Register(builtin.NewTerminal())

// RAG - 从文档库检索
registry.Register(builtin.NewRAGTool(retriever))
```

### 使用 FuncTool 创建自定义工具

```go
weatherTool := tools.NewFuncTool(
    "get_weather",
    "获取城市的当前天气",
    tools.ParameterSchema{
        Type: "object",
        Properties: map[string]tools.PropertySchema{
            "city": {
                Type:        "string",
                Description: "城市名称",
            },
            "unit": {
                Type:        "string",
                Description: "温度单位",
                Enum:        []string{"celsius", "fahrenheit"},
                Default:     "celsius",
            },
        },
        Required: []string{"city"},
    },
    func(ctx context.Context, args map[string]interface{}) (string, error) {
        city := args["city"].(string)
        // 调用天气 API...
        return fmt.Sprintf("%s 的天气：晴天，25°C", city), nil
    },
)
registry.Register(weatherTool)
```

### 简单工具（单个字符串参数）

```go
greetTool := tools.NewSimpleTool(
    "greet",
    "向某人问好",
    "name",
    "人名",
    func(ctx context.Context, name string) (string, error) {
        return "你好，" + name + "！", nil
    },
)
```

### 带超时的工具执行器

```go
executor := tools.NewExecutor(registry,
    tools.WithTimeout(30*time.Second),
    tools.WithMaxRetries(3),
)

result, err := executor.Execute(ctx, "calculator", map[string]interface{}{
    "expression": "2 + 2",
})
```

---

## 记忆系统

### 工作记忆

短期对话历史，带容量管理。

```go
mem := memory.NewWorkingMemory(
    memory.WithMaxSize(100),           // 最大消息数
    memory.WithTokenLimit(4000),       // Token 预算
    memory.WithTTL(30*time.Minute),    // 过期时间
)

// 存储消息
mem.AddMessage(ctx, message.NewUserMessage("你好"))
mem.AddMessage(ctx, message.NewAssistantMessage("你好！"))

// 检索历史
history, _ := mem.GetHistory(ctx, 10)       // 最后 10 条消息
recent, _ := mem.GetRecentHistory(ctx, 5)   // 最后 5 条消息
```

### 情景记忆

基于事件的记忆，带重要性评分。

```go
mem := memory.NewEpisodicMemory()

// 存储事件
mem.AddEpisode(ctx, memory.Episode{
    Type:       "user_preference",
    Content:    "用户喜欢深色模式",
    Importance: 0.8,
})

// 查询事件
important, _ := mem.GetMostImportant(ctx, 5)
byType, _ := mem.GetByType(ctx, "user_preference", 10)
byTime, _ := mem.GetByTimeRange(ctx, startTime, endTime)
```

### 语义记忆

基于向量的相似度搜索记忆。

```go
embedder := memory.NewOpenAIEmbedder(provider)
mem := memory.NewSemanticMemory(embedder)

// 存储（带可选元数据）
mem.Store(ctx, "id-1", "用户喜欢 Go 编程", map[string]interface{}{
    "category": "preferences",
})

// 语义搜索
results, _ := mem.Search(ctx, "编程语言偏好", 5)
for _, r := range results {
    fmt.Printf("%.2f: %s\n", r.Score, r.Content)
}
```

---

## RAG 管道

### 完整 RAG 示例

```go
// 1. 创建组件
chunker := rag.NewRecursiveCharacterChunker(512, 50)
store := rag.NewInMemoryVectorStore()
embedder := &OpenAIEmbedder{provider: provider}

// 2. 加载和处理文档
loader := rag.NewTextLoader()
doc, _ := loader.Load("document.txt")
chunks := chunker.Chunk(doc)

// 3. 生成嵌入并存储
embeddings, _ := embedder.Embed(ctx, getContents(chunks))
for i, chunk := range chunks {
    chunk.Vector = embeddings[i]
}
store.Add(ctx, chunks)

// 4. 创建检索器
retriever := rag.NewVectorRetriever(store, embedder,
    rag.WithScoreThreshold(0.7),
)

// 5. 查询
results, _ := retriever.Retrieve(ctx, "什么是 HelloAgents？", 3)
for _, r := range results {
    fmt.Printf("分数: %.2f\n内容: %s\n\n", r.Score, r.Chunk.Content)
}
```

### RAG 管道封装

```go
pipeline := rag.NewPipeline(embedder, store, answerGenerator)

// 摄取文档
docs := []rag.Document{doc1, doc2, doc3}
pipeline.Ingest(ctx, docs)

// 带来源追踪的查询
response, _ := pipeline.Query(ctx, "HelloAgents 有什么功能？")
fmt.Println("答案:", response.Answer)
fmt.Println("来源:", response.Sources)
```

### 多源检索

```go
multiRetriever := rag.NewMultiRetriever(
    []rag.Retriever{retriever1, retriever2, retriever3},
    &rag.ScoreBasedMerger{},
)
```

### 高级检索策略

HelloAgents 通过统一的管道架构支持高级检索增强策略。

#### MQE（多查询扩展）

将单个查询扩展为多个语义相关的变体，以提高召回率：

```go
retriever := rag.NewVectorRetriever(store, embedder)

// 使用 LLM 生成查询变体
results, _ := retriever.RetrieveWithOptions(ctx, "什么是机器学习？", 5,
    rag.WithMQE(llmProvider, 3), // 生成 3 个扩展查询
)
```

#### HyDE（假设文档嵌入）

生成假设性答案文档，使用其嵌入进行检索：

```go
results, _ := retriever.RetrieveWithOptions(ctx, "什么是机器学习？", 5,
    rag.WithHyDE(llmProvider),
)
```

#### 组合策略

组合多种策略以获得最佳检索质量：

```go
results, _ := retriever.RetrieveWithOptions(ctx, "什么是机器学习？", 5,
    rag.WithMQE(llmProvider, 2),        // 多查询扩展
    rag.WithHyDE(llmProvider),           // 假设文档
    rag.WithRerank(reranker),            // 重排序后处理
    rag.WithRRFFusion(60),               // RRF 融合多查询结果
)
```

#### 自定义变换器

使用自定义选项配置变换器：

```go
mqeTransformer := rag.NewMultiQueryTransformer(llmProvider,
    rag.WithNumQueries(3),
    rag.WithIncludeOriginal(true),
)

hydeTransformer := rag.NewHyDETransformer(llmProvider,
    rag.WithHyDEMaxTokens(256),
)

results, _ := retriever.RetrieveWithOptions(ctx, query, 5,
    rag.WithMQETransformer(mqeTransformer),
    rag.WithHyDETransformer(hydeTransformer),
    rag.WithTimeout(10*time.Second),
)
```

#### 可用策略

| 策略 | 类型 | 描述 |
|------|------|------|
| MQE | QueryTransformer | 将查询扩展为多个语义变体 |
| HyDE | QueryTransformer | 生成假设性答案文档 |
| RRF 融合 | FusionStrategy | 倒数排名融合多查询结果 |
| 分数融合 | FusionStrategy | 按最高分数合并 |
| 重排序 | PostProcessor | 使用交叉编码器重新评分 |

---

## 可观测性

### 设置 OpenTelemetry

```go
config := otel.Config{
    Enabled:        true,
    ServiceName:    "my-agent-service",
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
    },
}

provider, _ := otel.NewProvider(config)
defer provider.Shutdown(ctx)
```

### 带追踪的 LLM 提供商

```go
tracedProvider := otel.NewTracedProvider(llmProvider, tracer, metrics)

// 所有 LLM 调用自动追踪，包括：
// - 每次 Generate/GenerateStream 调用的 Span
// - Token 使用量指标
// - 延迟直方图
// - 错误追踪
```

### 自定义追踪

```go
tracer := otel.NewTracer(otelTracer)

ctx, span := tracer.Start(ctx, "my-operation",
    otel.WithSpanKind(otel.SpanKindInternal),
)
defer span.End()

span.SetAttributes(
    attribute.String("query", input.Query),
    attribute.Int("iteration", i),
)
span.AddEvent("tool_executed", attribute.String("tool", toolName))
```

### 指标

```go
metrics := otel.NewInMemoryMetrics()

// 计数器
counter := metrics.Counter("agent.requests")
counter.Add(ctx, 1, otel.NewAttr("agent_type", "react"))

// 直方图
histogram := metrics.Histogram("agent.latency_seconds")
histogram.Record(ctx, duration.Seconds())

// 仪表
gauge := metrics.Gauge("agent.active_sessions")
gauge.Set(ctx, float64(activeSessions))
```

---

## 项目结构

```
HelloAgents/
├── pkg/
│   ├── core/                    # 核心抽象
│   │   ├── llm/                 # LLM 提供商（OpenAI、Ollama 等）
│   │   ├── message/             # 消息类型和角色
│   │   ├── config/              # 配置管理
│   │   └── errors/              # 错误类型和处理
│   ├── agents/                  # Agent 实现
│   │   ├── simple.go            # SimpleAgent
│   │   ├── react.go             # ReActAgent
│   │   ├── reflection.go        # ReflectionAgent
│   │   └── plansolve.go         # PlanAndSolveAgent
│   ├── tools/                   # 工具系统
│   │   ├── registry.go          # 工具注册
│   │   ├── executor.go          # 工具执行
│   │   ├── func_tool.go         # FuncTool/SimpleTool
│   │   └── builtin/             # 内置工具
│   ├── memory/                  # 记忆系统
│   │   ├── working.go           # 工作记忆
│   │   ├── episodic.go          # 情景记忆
│   │   └── semantic.go          # 语义记忆
│   ├── rag/                     # RAG 管道
│   │   ├── chunker.go           # 文档分块
│   │   ├── store.go             # 向量存储
│   │   ├── retriever.go         # 检索策略
│   │   └── pipeline.go          # 管道编排
│   └── otel/                    # 可观测性
│       ├── tracer.go            # 分布式追踪
│       ├── metrics.go           # 指标收集
│       └── traced_provider.go   # 带追踪的 LLM 包装器
├── examples/                    # 示例应用
│   ├── simple/                  # 基础聊天
│   ├── react/                   # 带工具的 Agent
│   ├── memory/                  # 记忆演示
│   ├── rag/                     # RAG 问答
│   └── rag-advanced/            # 高级 RAG（MQE、HyDE）
├── tests/                       # 测试套件
│   ├── unit/                    # 单元测试
│   └── integration/             # 集成测试
└── docs/                        # 文档
    └── ARCHITECTURE.md          # 架构指南
```

---

## 配置

### 环境变量

```bash
# LLM 配置
export OPENAI_API_KEY=sk-...
export HELLOAGENTS_LLM_MODEL=gpt-4o
export HELLOAGENTS_LLM_TIMEOUT=30s
export HELLOAGENTS_LLM_MAX_RETRIES=3

# Agent 配置
export HELLOAGENTS_AGENT_MAX_ITERATIONS=10
export HELLOAGENTS_AGENT_TEMPERATURE=0.7
export HELLOAGENTS_AGENT_MAX_TOKENS=4096

# 可观测性
export HELLOAGENTS_OTEL_ENABLED=true
export HELLOAGENTS_OTEL_ENDPOINT=localhost:4317
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

observability:
  enabled: true
  service_name: my-agent
  tracing:
    enabled: true
    endpoint: localhost:4317
```

---

## 示例

```bash
# 克隆仓库
git clone https://github.com/easyops/helloagents-go.git
cd helloagents-go

# 设置 API Key
export OPENAI_API_KEY=your-key

# 运行示例
cd examples/simple && go run main.go     # 基础聊天
cd examples/react && go run main.go      # 带工具的 ReAct
cd examples/memory && go run main.go     # 记忆系统
cd examples/rag && go run main.go        # RAG 文档问答
```

---

## 测试

```bash
# 运行所有测试
go test ./...

# 只运行单元测试
go test ./tests/unit/...

# 带覆盖率
go test ./tests/unit/... -coverpkg=./pkg/...

# 详细输出
go test ./tests/unit/... -v
```

---

## 文档

- **[架构指南](docs/ARCHITECTURE.md)** - 详细的系统架构
- **[示例代码](examples/)** - 可运行的代码示例

---

## 系统要求

- Go 1.24 或更高版本
- LLM 提供商 API Key（OpenAI 或其他）

---

## 贡献

欢迎贡献！请在提交 PR 之前阅读贡献指南。

---

## 许可证

MIT License - 详见 [LICENSE](LICENSE)
