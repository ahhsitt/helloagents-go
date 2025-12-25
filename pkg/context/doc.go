// Package context 为 HelloAgents 框架提供上下文工程能力。
//
// 本包实现了 GSSC (Gather-Select-Structure-Compress) 流水线，
// 用于构建优化的 LLM 交互上下文。主要功能包括：
//
//   - Token 计数和预算管理
//   - 多源上下文收集（系统指令、历史、记忆、RAG）
//   - 基于相关性和新近性的筛选
//   - 结构化上下文模板（P0-P3 优先级分层）
//   - 压缩策略以满足 Token 预算
//
// # 基本用法
//
// 创建一个简单的上下文构建器：
//
//	builder := context.NewGSSCBuilder()
//	ctx, err := builder.Build(context.Background(), &context.BuildInput{
//	    Query:              "今天天气怎么样？",
//	    SystemInstructions: "你是一个有帮助的助手。",
//	    History:            conversationHistory,
//	})
//
// # 高级用法
//
// 使用自定义设置配置构建器：
//
//	config := context.NewConfig(
//	    context.WithMaxTokens(16000),
//	    context.WithMinRelevance(0.5),
//	    context.WithScoringWeights(0.8, 0.2), // 80% 相关性, 20% 新近性
//	)
//
//	builder := context.NewGSSCBuilder(
//	    context.WithConfig(config),
//	    context.WithGatherer(customGatherer),
//	    context.WithStructurer(context.NewMinimalStructurer()),
//	)
//
// # 集成 Memory 和 RAG
//
// 添加记忆和 RAG 收集器：
//
//	memoryGatherer := context.NewMemoryGatherer(func(ctx context.Context, query string, limit int) ([]context.MemoryResult, error) {
//	    // 你的记忆检索逻辑
//	    return results, nil
//	}, 5)
//
//	ragGatherer := context.NewRAGGatherer(func(ctx context.Context, query string, topK int) ([]context.RAGResult, error) {
//	    // 你的 RAG 检索逻辑
//	    return results, nil
//	}, 5)
//
//	builder := context.NewGSSCBuilder(
//	    context.WithGatherer(context.NewCompositeGatherer([]context.Gatherer{
//	        context.NewInstructionsGatherer(),
//	        context.NewTaskGatherer(),
//	        context.NewHistoryGatherer(10),
//	        memoryGatherer,
//	        ragGatherer,
//	    }, true)), // 并行收集
//	)
//
// # 上下文结构
//
// 默认结构化器生成以下格式的输出：
//
//	[Role & Policies]     (P0: 系统指令，最高优先级)
//	<系统指令>
//
//	[Task]                (P1: 当前查询)
//	用户问题：<query>
//
//	[State]               (P1: 来自记忆的任务状态)
//	<任务状态信息>
//
//	[Evidence]            (P2: 来自记忆/RAG 的事实证据)
//	<检索到的证据>
//
//	[Context]             (P3: 对话历史，最低优先级)
//	<对话历史>
//
//	[Output]              (输出格式约束)
//	<输出模板>
//
// 当超出 Token 预算时，内容从最低优先级（P3）段落开始截断。
package context
