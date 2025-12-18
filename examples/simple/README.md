# SimpleAgent 示例

本示例演示如何使用 `SimpleAgent` 进行基础对话。

## 前置条件

- Go 1.21+
- OpenAI API Key（或兼容 OpenAI API 的服务）

## 运行方式

### 使用 OpenAI

```bash
export OPENAI_API_KEY=your-api-key
go run ./examples/simple
```

### 使用兼容服务（如 Azure OpenAI、OneAPI 等）

```bash
export OPENAI_API_KEY=your-api-key
export OPENAI_BASE_URL=https://your-api-endpoint/v1
go run ./examples/simple
```

## 命令

- 输入任意文本进行对话
- `stream` - 切换流式输出模式
- `clear` - 清除对话历史
- `quit` 或 `exit` - 退出程序

## 示例对话

```
SimpleAgent Demo
================
Type your message and press Enter. Type 'quit' to exit.
Type 'stream' to toggle streaming mode.
Type 'clear' to clear conversation history.

You: 你好
Assistant: 你好！有什么我可以帮助你的吗？
(Tokens: prompt=25, completion=15, total=40, duration=1.2s)

You: 请用一句话介绍自己
Assistant: 我是一个人工智能助手，可以帮助你回答问题、提供信息和完成各种任务。
(Tokens: prompt=45, completion=28, total=73, duration=0.8s)

You: quit
Goodbye!
```

## 代码说明

1. **创建 LLM Provider**: 使用 `llm.NewOpenAI()` 创建 OpenAI 客户端
2. **创建 Agent**: 使用 `agents.NewSimple()` 创建简单对话 Agent
3. **配置选项**:
   - `WithName()`: 设置 Agent 名称
   - `WithSystemPrompt()`: 设置系统提示词
4. **对话**: 使用 `agent.Run()` 或 `agent.RunStream()` 进行对话
5. **历史管理**: 使用 `agent.ClearHistory()` 清除历史
