// Package main 展示 GAIA 评估使用示例
//
// GAIA (General AI Assistants) 用于评估智能体的通用 AI 助手能力。
// 数据集包含三个难度级别：Level 1（简单）、Level 2（中等）、Level 3（困难）。
//
// 使用前请先下载 GAIA 数据集：
//
//	huggingface-cli download gaia-benchmark/GAIA --local-dir ./gaia_data
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/easyops/helloagents-go/pkg/evaluation"
	"github.com/easyops/helloagents-go/pkg/evaluation/benchmarks/gaia"
)

func main() {
	ctx := context.Background()

	// 1. 创建数据集
	dataDir := "./gaia_data"
	level := 1         // 1=简单, 2=中等, 3=困难, 0=全部
	split := "validation" // validation 或 test

	dataset := gaia.NewDataset(dataDir, level, split)

	// 2. 加载数据集
	if err := dataset.Load(ctx); err != nil {
		log.Fatalf("加载数据集失败: %v", err)
	}

	fmt.Printf("数据集: %s\n", dataset.Name())
	fmt.Printf("样本数: %d\n", dataset.Len())

	// 3. 查看级别分布
	dist := dataset.GetLevelDistribution()
	fmt.Printf("级别分布:\n")
	for level, count := range dist {
		fmt.Printf("  Level %d: %d 个样本\n", level, count)
	}

	// 4. 查看第一个样本
	sample, err := dataset.Get(0)
	if err != nil {
		log.Fatalf("获取样本失败: %v", err)
	}

	fmt.Printf("\n样本示例:\n")
	fmt.Printf("  ID: %s\n", sample.ID)
	fmt.Printf("  Level: %d\n", sample.Level)
	fmt.Printf("  问题: %s\n", truncate(sample.Input, 100))
	if expected, ok := sample.Expected.(string); ok {
		fmt.Printf("  期望答案: %s\n", expected)
	}

	// 5. 创建评估器
	evaluator := gaia.NewEvaluator(dataset)
	fmt.Printf("\n评估器: %s\n", evaluator.Name())

	// 6. 执行评估（需要实际的 Agent）
	/*
		agent := yourAgent // 您的智能体实现

		result, err := evaluator.Evaluate(ctx, agent,
			evaluation.WithMaxSamples(10),
			evaluation.WithProgressCallback(func(done, total int) {
				fmt.Printf("进度: %d/%d\n", done, total)
			}),
		)

		if err != nil {
			log.Fatalf("评估失败: %v", err)
		}

		// 7. 查看结果
		fmt.Printf("\n评估结果:\n")
		fmt.Printf("  总样本数: %d\n", result.TotalSamples)
		fmt.Printf("  成功数: %d\n", result.SuccessCount)
		fmt.Printf("  准确率: %.2f%%\n", result.OverallAccuracy*100)

		// 级别指标
		for level, lm := range result.LevelMetrics {
			fmt.Printf("  Level %d: %.2f%% 精确匹配率\n", level, lm.ExactMatchRate*100)
		}

		// 8. 导出结果
		exporter := gaia.NewExporter()
		if err := exporter.Export(result, "./gaia_submission.jsonl"); err != nil {
			log.Fatalf("导出失败: %v", err)
		}

		if err := exporter.ExportMarkdownReport(result, "./gaia_report.md"); err != nil {
			log.Fatalf("导出报告失败: %v", err)
		}
	*/

	fmt.Println("\n要执行完整评估，请取消注释上面的代码并提供您的智能体实现。")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// 演示答案标准化
func demoAnswerNormalization() {
	// GAIA 评估会自动标准化答案：
	// - 转为小写
	// - 移除前导冠词 (the, a, an)
	// - 移除尾随标点
	// - 移除货币符号和百分号
	// - 移除数字中的逗号分隔符

	examples := []struct {
		input    string
		expected string
	}{
		{"The answer", "answer"},
		{"$1,000,000", "1000000"},
		{"50%", "50"},
		{"Beijing.", "beijing"},
	}

	fmt.Println("答案标准化示例:")
	for _, ex := range examples {
		fmt.Printf("  %q -> %q\n", ex.input, ex.expected)
	}
}

// 演示使用评估工具
func demoEvaluationTool() {
	// 如果想使用一键评估工具，可以：
	/*
		tool := evaluation.NewGAIAEvaluationTool(dataDir, outputDir, agent)
		result, err := tool.Execute(ctx, map[string]interface{}{
			"level":       1,
			"split":       "validation",
			"max_samples": 10,
		})
	*/
	_ = evaluation.WithVerbose(true) // 占位符
}
