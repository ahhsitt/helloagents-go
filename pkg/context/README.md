# Context 模块

Context 模块实现了 **GSSC 流水线**（Gather-Select-Structure-Compress），用于构建优化的 LLM 交互上下文。这是一个上下文工程模块，解决 LLM 在有限 Token 预算下如何高效组织信息的问题。

## 架构概述

```
┌─────────────────────────────────────────────────────────────────────┐
│                         GSSC Pipeline                                │
│                                                                      │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐         │
│  │  Gather  │ → │  Select  │ → │ Structure│ → │ Compress │ → Output │
│  │ 收集阶段  │   │ 筛选阶段  │   │ 结构化   │   │  压缩    │         │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘         │
│       ↑              ↑              ↑              ↑                │
│   Gatherers      Selector      Structurer     Compressor           │
│   收集器         筛选器         结构化器        压缩器              │
└─────────────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Packet（上下文包）

Packet 是上下文信息的基本单元，每个包携带内容和元数据：

```go
type Packet struct {
    Content        string                 // 实际文本内容
    Type           PacketType             // 类型/优先级
    Timestamp      time.Time              // 创建时间
    TokenCount     int                    // Token 数量
    RelevanceScore float64                // 相关性评分 (0.0-1.0)
    RecencyScore   float64                // 新近性评分 (0.0-1.0)
    CompositeScore float64                // 综合评分
    Metadata       map[string]interface{} // 额外元数据
    Source         string                 // 来源标识
}
```

**优先级分层（P0-P3）**：

| 优先级 | PacketType | 说明 | 截断顺序 |
|--------|------------|------|----------|
| P0 | `instructions` | 系统指令 | 最后截断 |
| P1 | `task`, `task_state` | 当前任务/状态 | 倒数第二 |
| P2 | `evidence` | Memory/RAG 证据 | 倒数第三 |
| P3 | `history` | 对话历史 | 最先截断 |
| P4 | `custom` | 自定义内容 | 最先截断 |

### 2. Config（配置）

管理上下文构建的所有配置参数：

```go
type Config struct {
    MaxTokens          int           // 总 Token 预算（默认 8000）
    ReserveRatio       float64       // 响应预留比例（默认 0.15 = 15%）
    MinRelevance       float64       // 最低相关性阈值（默认 0.3）
    RelevanceWeight    float64       // 相关性权重（默认 0.7）
    RecencyWeight      float64       // 新近性权重（默认 0.3）
    RecencyTau         float64       // 新近性衰减时间常数（默认 3600 秒）
    MaxHistoryMessages int           // 最大历史消息数（默认 10）
    EnableCompression  bool          // 是否启用压缩
    EnableMMR          bool          // 是否启用 MMR 多样性
    TokenCounter       TokenCounter  // Token 计数器
    OutputTemplate     string        // 输出格式模板
}
```

**可用 Token 计算**：

```go
func (c *Config) GetAvailableTokens() int {
    return int(float64(c.MaxTokens) * (1 - c.ReserveRatio))
}
// 例：8000 * 0.85 = 6800 可用 Token
```

### 3. TokenCounter（Token 计数）

提供精确和估算两种 Token 计数方式：

```go
type TokenCounter interface {
    Count(text string) int
    CountMessages(messages []message.Message) int
}
```

**实现**：

| 实现类 | 说明 | 精确度 |
|--------|------|--------|
| `TiktokenCounter` | 使用 tiktoken-go 库 | 精确 |
| `EstimatedCounter` | 字符估算（1 token ≈ 4 字符） | 近似 |

```go
// TiktokenCounter 使用真实的 tokenizer
func (c *TiktokenCounter) Count(text string) int {
    return len(c.encoding.Encode(text, nil, nil))
}

// EstimatedCounter 作为降级方案
func (c *EstimatedCounter) Count(text string) int {
    return int(float64(len(text)) / c.CharsPerToken)
}
```

## GSSC 流水线详解

### Phase 1: Gather（收集）

从各种来源收集上下文包：

```go
type Gatherer interface {
    Gather(ctx context.Context, input *GatherInput) ([]*Packet, error)
}
```

**内置收集器**：

| 收集器 | 功能 | 输出类型 |
|--------|------|----------|
| `InstructionsGatherer` | 收集系统指令 | `PacketTypeInstructions` |
| `TaskGatherer` | 收集当前查询 | `PacketTypeTask` |
| `HistoryGatherer` | 收集对话历史 | `PacketTypeHistory` |
| `MemoryGatherer` | 集成记忆系统 | `PacketTypeEvidence/TaskState` |
| `RAGGatherer` | 集成 RAG 检索 | `PacketTypeEvidence` |
| `CompositeGatherer` | 组合多个收集器 | 混合 |

**CompositeGatherer 支持并行收集**：

```go
func NewCompositeGatherer(gatherers []Gatherer, parallel bool) *CompositeGatherer

// parallel=true 时使用 goroutine 并发收集
func (g *CompositeGatherer) gatherParallel(ctx context.Context, input *GatherInput) ([]*Packet, error) {
    var wg sync.WaitGroup
    for _, gatherer := range g.gatherers {
        wg.Add(1)
        go func(gth Gatherer) {
            defer wg.Done()
            packets, _ := gth.Gather(ctx, input)
            // 加锁后追加结果
        }(gatherer)
    }
    wg.Wait()
    return allPackets, nil
}
```

### Phase 2: Select（筛选）

对包进行评分和过滤：

```go
type Selector interface {
    Select(packets []*Packet, query string, config *Config) []*Packet
}

type Scorer interface {
    Score(packet *Packet, query string) float64
}
```

**评分机制**：

1. **RelevanceScorer（相关性评分）**：

```go
// 基于关键词重叠的 Jaccard-like 评分
func (s *RelevanceScorer) Score(packet *Packet, query string) float64 {
    queryTokens := tokenize(query)
    contentTokens := tokenize(packet.Content)
    overlap := countOverlap(queryTokens, contentTokens)
    return float64(overlap) / float64(len(queryTokens))
}
```

2. **RecencyScorer（新近性评分）**：

```go
// 指数衰减：e^(-Δt/τ)
func (s *RecencyScorer) Score(packet *Packet, _ string) float64 {
    delta := time.Since(packet.Timestamp).Seconds()
    return math.Exp(-delta / s.Tau)
}
```

3. **CompositeScorer（复合评分）**：

```go
// 加权组合：0.7 * 相关性 + 0.3 * 新近性
CompositeScore = RelevanceWeight * RelevanceScore + RecencyWeight * RecencyScore
```

**DefaultSelector 筛选逻辑**：

```go
func (s *DefaultSelector) Select(packets []*Packet, query string, config *Config) []*Packet {
    // 1. 对所有包计算评分
    for _, packet := range packets {
        packet.RelevanceScore = ...
        packet.RecencyScore = ...
        packet.CompositeScore = ...
    }

    // 2. P0 包（instructions, task）始终包含
    // 3. 按 MinRelevance 过滤其他包
    // 4. 按优先级和复合分数排序
    // 5. 在 Token 预算内选择
    for _, packet := range filtered {
        if usedTokens + packet.TokenCount <= availableTokens {
            selected = append(selected, packet)
            usedTokens += packet.TokenCount
        }
    }
    return selected
}
```

### Phase 3: Structure（结构化）

将包组织成结构化的上下文模板：

```go
type Structurer interface {
    Structure(packets []*Packet, query string, config *Config) string
}
```

**DefaultStructurer 输出格式**：

```
[Role & Policies]        ← P0: 系统指令
<系统指令内容>

[Task]                   ← P1: 当前任务
用户问题：<query>

[State]                  ← P1: 任务状态
关键进展与未决问题：
<任务状态信息>

[Evidence]               ← P2: 事实证据
事实与引用：
[来源: memory] <证据内容>
[来源: rag] <证据内容>

[Context]                ← P3: 对话历史
对话历史与背景：
<历史内容>

[Output]                 ← 输出约束
请按以下格式回答：
1. 结论（简洁明确）
2. 依据（列出支撑证据及来源）
3. 风险与假设（如有）
4. 下一步行动建议（如适用）
```

**其他结构化器**：

| 结构化器 | 说明 |
|----------|------|
| `MinimalStructurer` | 无分段标题，按优先级连接 |
| `CustomStructurer` | 支持模板占位符 `{{instructions}}`, `{{task}}` 等 |

### Phase 4: Compress（压缩）

在超出预算时压缩内容：

```go
type Compressor interface {
    Compress(context string, config *Config) string
}
```

**TruncateCompressor 压缩策略**：

```go
func (c *TruncateCompressor) compressWithStructure(context string, config *Config) string {
    // 截断优先级（从低到高，先截断低优先级）
    priorities := []string{
        "[Context]",         // P3 - 最先截断
        "[Evidence]",        // P2
        "[State]",           // P1
        "[Output]",          // 辅助
        "[Task]",            // P1
        "[Role & Policies]", // P0 - 最后截断
    }

    for _, priority := range priorities {
        if currentTokens <= availableTokens {
            break
        }
        // 先尝试部分截断（保留50%）
        sections[priority] = truncateSection(section, sectionTokens/2, counter)
        // 如果仍超预算，完全删除该分段
        if stillOverBudget {
            delete(sections, priority)
        }
    }
    return rebuildContext(sections, priorities)
}
```

截断时添加指示：

```
... (内容已截断)
```

## Builder（构建器）

整合 GSSC 流水线的入口：

```go
type Builder interface {
    Build(ctx context.Context, input *BuildInput) (string, error)
    BuildMessages(ctx context.Context, input *BuildInput) ([]message.Message, error)
}

type BuildInput struct {
    Query              string            // 当前查询
    SystemInstructions string            // 系统指令
    History            []message.Message // 对话历史
    AdditionalPackets  []*Packet         // 额外的包
}
```

**GSSCBuilder 流程**：

```go
func (b *GSSCBuilder) Build(ctx context.Context, input *BuildInput) (string, error) {
    // 1. Gather: 收集候选包
    packets, _ := b.gatherer.Gather(ctx, gatherInput)
    packets = append(packets, input.AdditionalPackets...)

    // 2. Select: 评分和过滤
    selected := b.selector.Select(packets, input.Query, b.config)

    // 3. Structure: 组织成模板
    structured := b.structurer.Structure(selected, input.Query, b.config)

    // 4. Compress: 适应预算
    compressed := b.compressor.Compress(structured, b.config)

    return compressed, nil
}
```

**使用 Option 模式配置**：

```go
builder := NewGSSCBuilder(
    WithConfig(NewConfig(
        WithMaxTokens(16000),
        WithMinRelevance(0.5),
        WithScoringWeights(0.8, 0.2),
    )),
    WithGatherer(customGatherer),
    WithStructurer(NewMinimalStructurer()),
    WithCompressor(NewNoOpCompressor()),
)
```

## Agent 集成

通过 `WithContextBuilder` 选项将 Context 模块集成到 Agent：

```go
// pkg/agents/options.go
func WithContextBuilder(builder agentctx.Builder) Option {
    return func(o *AgentOptions) {
        o.ContextBuilder = builder
    }
}

// pkg/agents/simple.go
func (a *SimpleAgent) buildMessages(ctx context.Context, query string) ([]message.Message, error) {
    if a.opts.ContextBuilder != nil {
        return a.opts.ContextBuilder.BuildMessages(ctx, &agentctx.BuildInput{
            Query:              query,
            SystemInstructions: a.opts.SystemPrompt,
            History:            a.history,
        })
    }
    // 降级到默认消息构建
    ...
}
```

## 使用示例

### 基本用法

```go
builder := context.NewGSSCBuilder()
ctx, _ := builder.Build(context.Background(), &context.BuildInput{
    Query:              "今天天气怎么样？",
    SystemInstructions: "你是一个有帮助的助手。",
    History:            conversationHistory,
})
```

### 集成 Memory 和 RAG

```go
memoryGatherer := context.NewMemoryGatherer(func(ctx context.Context, query string, limit int) ([]context.MemoryResult, error) {
    // 记忆检索逻辑
    return results, nil
}, 5)

ragGatherer := context.NewRAGGatherer(func(ctx context.Context, query string, topK int) ([]context.RAGResult, error) {
    // RAG 检索逻辑
    return results, nil
}, 5)

builder := context.NewGSSCBuilder(
    context.WithGatherer(context.NewCompositeGatherer([]context.Gatherer{
        context.NewInstructionsGatherer(),
        context.NewTaskGatherer(),
        context.NewHistoryGatherer(10),
        memoryGatherer,
        ragGatherer,
    }, true)), // 并行收集
)
```

### 自定义配置

```go
config := context.NewConfig(
    context.WithMaxTokens(16000),
    context.WithMinRelevance(0.5),
    context.WithScoringWeights(0.8, 0.2), // 80% 相关性, 20% 新近性
    context.WithRecencyTau(7200),         // 2 小时衰减
)

builder := context.NewGSSCBuilder(
    context.WithConfig(config),
    context.WithStructurer(context.NewMinimalStructurer()),
)
```

## 架构优势

1. **模块化设计**：每个阶段独立实现，可单独替换
2. **优先级分层**：确保重要信息不被截断
3. **Token 预算管理**：精确控制上下文长度
4. **可扩展性**：通过接口轻松添加新的收集器、评分器等
5. **并行收集**：支持并发从多个来源获取数据
6. **Option 模式**：灵活的配置方式

## 文件结构

```
pkg/context/
├── doc.go        # 包文档
├── token.go      # Token 计数器
├── packet.go     # 上下文包定义
├── config.go     # 配置管理
├── gather.go     # 收集器实现
├── selector.go   # 筛选器和评分器
├── structure.go  # 结构化器
├── compress.go   # 压缩器
├── builder.go    # GSSC 构建器
└── README.md     # 本文档
```
