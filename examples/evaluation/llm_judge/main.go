// Package main 展示 LLM Judge 评估使用示例
//
// LLM Judge 使用 LLM 作为评委对生成的数据进行多维度质量评估。
// 评估维度包括：正确性、清晰度、难度匹配、完整性。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/easyops/helloagents-go/pkg/evaluation"
	"github.com/easyops/helloagents-go/pkg/evaluation/benchmarks/datagen"
)

func main() {
	ctx := context.Background()

	// 1. 创建待评估数据集
	// 数据文件应为 JSONL 格式，每行一个 JSON 对象
	// 格式：{"question": "...", "answer": "...", "category": "..."}
	dataPath := "./generated_questions.jsonl"
	dataset := datagen.NewDataset(dataPath)

	// 2. 加载数据集
	if err := dataset.Load(ctx); err != nil {
		log.Fatalf("加载数据集失败: %v", err)
	}

	fmt.Printf("数据集: %s\n", dataset.Name())
	fmt.Printf("样本数: %d\n", dataset.Len())

	// 3. 可选：加载参考数据（用于对比评估）
	// refDataPath := "./reference_questions.jsonl"
	// refDataset := datagen.NewDataset(refDataPath)
	// refDataset.Load(ctx)

	// 4. 创建 LLM Judge 评估器（需要 LLM Provider）
	/*
		llmProvider := yourLLMProvider // 您的 LLM 提供商实现

		config := datagen.JudgeConfig{
			ReferenceSamples: refDataset.GetSamples(), // 可选
		}

		judge := datagen.NewLLMJudge(llmProvider, dataset, config)

		// 5. 执行评估
		result, err := judge.Evaluate(ctx,
			evaluation.WithMaxSamples(10),
			evaluation.WithProgressCallback(func(done, total int) {
				fmt.Printf("进度: %d/%d\n", done, total)
			}),
		)

		if err != nil {
			log.Fatalf("评估失败: %v", err)
		}

		// 6. 查看结果
		fmt.Printf("\n评估结果:\n")
		fmt.Printf("  平均分: %.2f\n", result.Metrics.AverageScore)
		fmt.Printf("  通过率: %.2f%%\n", result.Metrics.PassRate*100)
		fmt.Printf("  优秀率: %.2f%%\n", result.Metrics.ExcellentRate*100)

		fmt.Printf("\n各维度评分:\n")
		for dim, score := range result.Metrics.DimensionScores {
			fmt.Printf("  %s: %.2f\n", dim, score)
		}

		// 7. 导出报告
		exporter := datagen.NewExporter()
		if err := exporter.ExportJudgeReport(result, "./llm_judge_report.md"); err != nil {
			log.Fatalf("导出报告失败: %v", err)
		}

		if err := exporter.ExportJSON(result, "./llm_judge_result.json"); err != nil {
			log.Fatalf("导出 JSON 失败: %v", err)
		}
	*/

	fmt.Println("\n要执行完整评估，请取消注释上面的代码并提供您的 LLM Provider。")
}

// 演示评分维度
func demoDimensions() {
	fmt.Println("LLM Judge 评分维度 (1-5 分):")
	fmt.Println()

	dimensions := []struct {
		name        string
		description string
	}{
		{"正确性 (Correctness)", "题目和答案是否正确"},
		{"清晰度 (Clarity)", "题目描述是否清晰、无歧义"},
		{"难度匹配 (Difficulty Match)", "题目难度是否与标注一致"},
		{"完整性 (Completeness)", "题目信息是否完整"},
	}

	for _, d := range dimensions {
		fmt.Printf("  • %s: %s\n", d.name, d.description)
	}

	fmt.Println("\n评分标准:")
	fmt.Println("  1 分: 非常差")
	fmt.Println("  2 分: 差")
	fmt.Println("  3 分: 一般")
	fmt.Println("  4 分: 好")
	fmt.Println("  5 分: 非常好")

	fmt.Println("\n通过标准: 平均分 >= 3.0")
	fmt.Println("优秀标准: 平均分 >= 4.0")
}

// 演示数据格式
func demoDataFormat() {
	fmt.Println("输入数据格式 (JSONL):")
	fmt.Println()
	fmt.Println(`{"id": "q001", "question": "2+2等于多少?", "answer": "4", "category": "arithmetic"}`)
	fmt.Println(`{"id": "q002", "question": "地球是太阳系第几颗行星?", "answer": "第三颗", "category": "astronomy"}`)
}

// 使用占位符避免未使用导入
func init() {
	_ = evaluation.WithVerbose(true)
}
