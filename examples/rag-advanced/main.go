package main

import (
	"context"
	"fmt"
	"os"

	"github.com/easyops/helloagents-go/pkg/rag"
)

// MockLLMProvider 模拟 LLM 提供者（用于演示）
type MockLLMProvider struct{}

func (p *MockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	// 模拟 MQE 查询扩展
	if containsSubstring(prompt, "查询扩展") || containsSubstring(prompt, "语义相关") {
		return `机器学习的基本概念是什么
什么是监督学习和非监督学习
深度学习与机器学习的区别`, nil
	}

	// 模拟 HyDE 假设文档生成
	return `机器学习是人工智能的一个分支，它使计算机系统能够从数据中学习和改进，
而无需显式编程。机器学习算法使用统计技术来发现数据中的模式，
并使用这些模式进行预测或决策。常见的机器学习类型包括监督学习、
非监督学习和强化学习。`, nil
}

// MockEmbedder 模拟嵌入器
type MockEmbedder struct{}

func (e *MockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		// 生成模拟向量
		embeddings[i] = make([]float32, 128)
		for j := range embeddings[i] {
			embeddings[i][j] = float32(i+j) * 0.01
		}
	}
	return embeddings, nil
}

// MockReranker 模拟重排序器
type MockReranker struct{}

func (r *MockReranker) Rerank(ctx context.Context, query string, results []rag.RetrievalResult) ([]rag.RetrievalResult, error) {
	// 简单地反转结果顺序作为模拟
	reranked := make([]rag.RetrievalResult, len(results))
	for i, result := range results {
		reranked[len(results)-1-i] = result
		reranked[len(results)-1-i].Score = float32(len(results)-i) * 0.1
	}
	return reranked, nil
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func main() {
	ctx := context.Background()

	// 创建组件
	store := rag.NewInMemoryVectorStore()
	embedder := &MockEmbedder{}
	llm := &MockLLMProvider{}
	reranker := &MockReranker{}

	// 添加一些测试文档
	docs := []rag.Document{
		{ID: "doc1", Content: "机器学习是人工智能的核心技术之一"},
		{ID: "doc2", Content: "深度学习使用神经网络进行学习"},
		{ID: "doc3", Content: "监督学习需要标注数据"},
	}

	chunker := rag.NewRecursiveCharacterChunker(100, 20)
	var chunks []rag.DocumentChunk
	for _, doc := range docs {
		docChunks := chunker.Chunk(doc)
		chunks = append(chunks, docChunks...)
	}

	// 为块生成嵌入并存储
	contents := make([]string, len(chunks))
	for i, chunk := range chunks {
		contents[i] = chunk.Content
	}
	embeddings, _ := embedder.Embed(ctx, contents)
	for i := range chunks {
		chunks[i].Vector = embeddings[i]
	}
	store.Add(ctx, chunks)

	// 创建检索器
	retriever := rag.NewVectorRetriever(store, embedder)

	fmt.Println("=== 高级 RAG 检索示例 ===")
	fmt.Println()

	// 1. 基础检索（无策略）
	fmt.Println("1. 基础检索:")
	results, err := retriever.Retrieve(ctx, "什么是机器学习?", 3)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		os.Exit(1)
	}
	printResults(results)

	// 2. 使用 MQE（多查询扩展）
	fmt.Println("\n2. MQE 检索 (多查询扩展):")
	results, err = retriever.RetrieveWithOptions(ctx, "什么是机器学习?", 3,
		rag.WithMQE(llm, 3),
	)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		os.Exit(1)
	}
	printResults(results)

	// 3. 使用 HyDE（假设文档嵌入）
	fmt.Println("\n3. HyDE 检索 (假设文档嵌入):")
	results, err = retriever.RetrieveWithOptions(ctx, "什么是机器学习?", 3,
		rag.WithHyDE(llm),
	)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		os.Exit(1)
	}
	printResults(results)

	// 4. 组合策略：MQE + HyDE + Rerank
	fmt.Println("\n4. 组合策略 (MQE + HyDE + Rerank):")
	results, err = retriever.RetrieveWithOptions(ctx, "什么是机器学习?", 3,
		rag.WithMQE(llm, 2),
		rag.WithHyDE(llm),
		rag.WithRerank(reranker),
		rag.WithRRFFusion(60),
	)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		os.Exit(1)
	}
	printResults(results)

	// 5. 使用自定义变换器
	fmt.Println("\n5. 自定义变换器配置:")
	mqeTransformer := rag.NewMultiQueryTransformer(llm,
		rag.WithNumQueries(2),
		rag.WithIncludeOriginal(true),
	)
	hydeTransformer := rag.NewHyDETransformer(llm,
		rag.WithHyDEMaxTokens(256),
	)
	results, err = retriever.RetrieveWithOptions(ctx, "什么是机器学习?", 3,
		rag.WithMQETransformer(mqeTransformer),
		rag.WithHyDETransformer(hydeTransformer),
	)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		os.Exit(1)
	}
	printResults(results)

	fmt.Println("\n=== 示例完成 ===")
}

func printResults(results []rag.RetrievalResult) {
	if len(results) == 0 {
		fmt.Println("   (无结果)")
		return
	}
	for i, r := range results {
		content := r.Chunk.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		fmt.Printf("   [%d] 分数: %.4f | 内容: %s\n", i+1, r.Score, content)
	}
}
