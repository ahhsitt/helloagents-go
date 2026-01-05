# HelloAgents Go

一个轻量级、模块化的 Go 语言 AI Agent 框架。

> **致谢**: 本项目基于 [jjyaoao/HelloAgents](https://github.com/jjyaoao/HelloAgents) 使用 Go 语言重新实现。
> 感谢原作者 [@jjyaoao](https://github.com/jjyaoao) 和 [Datawhale](https://github.com/datawhalechina) 社区的贡献。

[![License: CC BY-NC-SA 4.0](https://img.shields.io/badge/License-CC%20BY--NC--SA%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-sa/4.0/)
[![Go 1.24+](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)

## 特性

- **多种 Agent 模式** - SimpleAgent、ReActAgent、ReflectionAgent、PlanAndSolveAgent
- **工具系统** - 内置计算器、终端工具，支持自定义工具
- **记忆系统** - 工作记忆、情景记忆、语义记忆
- **RAG 管道** - 文档分块、向量检索、MQE/HyDE 高级策略
- **MCP 协议** - 支持 Model Context Protocol 客户端和服务端
- **多 LLM 支持** - OpenAI、DeepSeek、Qwen、Ollama、vLLM

## 快速开始

### 安装

```bash
go get github.com/easyops/helloagents-go
```

### 环境配置

```bash
export OPENAI_API_KEY=your-api-key
export OPENAI_BASE_URL=https://api.openai.com/v1  # 可选，用于兼容 API
```

## 示例

### 示例 1: 简单对话 (SimpleAgent)

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
    provider, _ := llm.NewOpenAI(llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
    defer provider.Close()

    agent, _ := agents.NewSimple(provider,
        agents.WithName("助手"),
        agents.WithSystemPrompt("你是一个有帮助的 AI 助手"),
    )

    output, _ := agent.Run(context.Background(), agents.Input{
        Query: "你好，请介绍一下自己",
    })
    fmt.Println(output.Response)
}
```

### 示例 2: 工具调用 (ReActAgent)

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
    provider, _ := llm.NewOpenAI(llm.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
    defer provider.Close()

    // 注册工具
    registry := tools.NewRegistry()
    registry.MustRegister(builtin.NewCalculator())
    registry.MustRegister(builtin.NewTerminal())

    // 创建 ReAct Agent
    agent, _ := agents.NewReAct(provider, registry,
        agents.WithMaxIterations(5),
    )

    output, _ := agent.Run(context.Background(), agents.Input{
        Query: "计算 123 * 456 等于多少？",
    })

    fmt.Println(output.Response)
    for _, step := range output.Steps {
        fmt.Printf("[%s] %s\n", step.Type, step.Content)
    }
}
```

### 示例 3: RAG 文档问答

```go
package main

import (
    "context"
    "fmt"

    "github.com/easyops/helloagents-go/pkg/rag"
)

func main() {
    ctx := context.Background()

    // 创建 RAG 组件
    chunker := rag.NewRecursiveCharacterChunker(200, 20)
    store := rag.NewInMemoryVectorStore()

    // 创建文档并摄取
    docs := []rag.Document{
        {ID: "1", Content: "HelloAgents 是一个 Go 语言 AI Agent 框架..."},
        {ID: "2", Content: "ReAct 模式让 Agent 能够边推理边行动..."},
    }

    pipeline := rag.NewRAGPipeline(
        rag.WithChunker(chunker),
        rag.WithStore(store),
    )
    pipeline.Ingest(ctx, docs)

    // 查询
    response, _ := pipeline.Query(ctx, "什么是 HelloAgents?", 3)
    fmt.Println(response.Answer)
}
```

### 示例 4: MCP 协议集成

```go
package main

import (
    "context"
    "fmt"

    "github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func main() {
    ctx := context.Background()

    // 创建 MCP 客户端
    transport := mcp.NewStdioTransport("path/to/mcp-server")
    client := mcp.NewClient(transport)
    defer client.Close()

    client.Initialize(ctx)

    // 列出可用工具
    tools, _ := client.ListTools(ctx)
    for _, tool := range tools {
        fmt.Printf("工具: %s - %s\n", tool.Name, tool.Description)
    }

    // 调用工具
    result, _ := client.CallTool(ctx, "add", map[string]interface{}{
        "a": 10, "b": 5,
    })
    fmt.Println("结果:", result)
}
```

## 运行示例

```bash
# 克隆仓库
git clone https://github.com/easyops/helloagents-go.git
cd helloagents-go

# 设置环境变量
export OPENAI_API_KEY=your-key

# 运行示例
go run examples/simple/main.go      # 简单对话
go run examples/react/main.go       # 工具调用
go run examples/rag/main.go         # RAG 问答
go run examples/mcp/client/main.go  # MCP 客户端
```

## 更多示例

完整示例请参考 [examples/](examples/) 目录：

| 目录 | 说明 |
|------|------|
| `examples/simple/` | SimpleAgent 基础对话 |
| `examples/react/` | ReActAgent 工具调用 |
| `examples/memory/` | 记忆系统使用 |
| `examples/rag/` | RAG 文档问答 |
| `examples/rag-advanced/` | 高级 RAG (MQE, HyDE) |
| `examples/mcp/` | MCP 协议客户端/服务端 |
| `examples/evaluation/` | Agent 性能评估 |

## 测试

```bash
go test ./...                           # 运行所有测试
go test ./tests/unit/... -v             # 运行单元测试
go test ./... -coverprofile=coverage.out  # 生成覆盖率报告
```

## License

本项目采用 [CC BY-NC-SA 4.0](https://creativecommons.org/licenses/by-nc-sa/4.0/) 协议，与原项目保持一致。

**协议要求：**
- **署名 (BY)** - 必须注明原作者
- **非商业 (NC)** - 仅限非商业用途
- **相同方式共享 (SA)** - 衍生作品必须使用相同协议

## 致谢

- 原项目: [jjyaoao/HelloAgents](https://github.com/jjyaoao/HelloAgents)
- 教程来源: [Datawhale hello-agents](https://github.com/datawhalechina/hello-agents)
