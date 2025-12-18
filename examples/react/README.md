# ReActAgent 示例

本示例演示如何使用 `ReActAgent` 进行工具调用。

## 前置条件

- Go 1.21+
- OpenAI API Key（或兼容 OpenAI API 的服务）

## 运行方式

```bash
export OPENAI_API_KEY=your-api-key
go run ./examples/react
```

## 工具列表

- **calculator**: 执行数学运算
- **terminal**: 执行 shell 命令（受限制）

## 示例对话

```
ReActAgent Demo
================
This agent can use tools to help answer questions.
Available tools:
  - calculator: Perform mathematical calculations...
  - terminal: Execute shell commands...

You: 123 乘以 456 等于多少？

--- Reasoning Steps ---
[1] Action: calculator(map[expression:123 * 456])
[2] Observation: 56088
-----------------------