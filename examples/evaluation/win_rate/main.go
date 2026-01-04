// Package main 展示 Win Rate 评估使用示例
//
// Win Rate 通过成对对比来计算胜率，评估生成数据相对于参考数据的质量。
// 支持位置随机化以消除位置偏差。
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

	// 1. 创建候选数据集（待评估）
	candidatePath := "./generated_questions.jsonl"
	candidateDataset := datagen.NewDataset(candidatePath)

	// 2. 创建参考数据集（基准）
	referencePath := "./reference_questions.jsonl"
	referenceDataset := datagen.NewDataset(referencePath)

	// 3. 加载数据集
	if err := candidateDataset.Load(ctx); err != nil {
		log.Fatalf("加载候选数据集失败: %v", err)
	}

	if err := referenceDataset.Load(ctx); err != nil {
		log.Fatalf("加载参考数据集失败: %v", err)
	}

	fmt.Printf("候选数据集: %d 个样本\n", candidateDataset.Len())
	fmt.Printf("参考数据集: %d 个样本\n", referenceDataset.Len())

	// 4. 创建 Win Rate 评估器（需要 LLM Provider）
	/*
		llmProvider := yourLLMProvider // 您的 LLM 提供商实现

		config := datagen.WinRateConfig{
			RandomSeed: 42, // 可选，用于位置随机化的随机种子
		}

		evaluator := datagen.NewWinRateEvaluator(
			llmProvider,
			candidateDataset,
			referenceDataset,
			config,
		)

		// 5. 执行评估
		result, err := evaluator.Evaluate(ctx,
			evaluation.WithMaxSamples(20), // 对比前 20 对
			evaluation.WithProgressCallback(func(done, total int) {
				fmt.Printf("进度: %d/%d\n", done, total)
			}),
		)

		if err != nil {
			log.Fatalf("评估失败: %v", err)
		}

		// 6. 查看结果
		fmt.Printf("\n胜率统计:\n")
		fmt.Printf("  胜: %d (%.2f%%)\n",
			result.Metrics.Extra["wins"],
			result.Metrics.WinRate*100)
		fmt.Printf("  负: %d (%.2f%%)\n",
			result.Metrics.Extra["losses"],
			result.Metrics.LossRate*100)
		fmt.Printf("  平: %d (%.2f%%)\n",
			result.Metrics.Extra["ties"],
			result.Metrics.TieRate*100)

		// 7. 结论
		if result.Metrics.WinRate > 0.6 {
			fmt.Println("\n结论: 候选数据集显著优于参考数据集")
		} else if result.Metrics.WinRate > 0.4 {
			fmt.Println("\n结论: 候选数据集与参考数据集质量相当")
		} else {
			fmt.Println("\n结论: 候选数据集不及参考数据集")
		}

		// 8. 导出报告
		exporter := datagen.NewExporter()
		if err := exporter.ExportWinRateReport(result, "./win_rate_report.md"); err != nil {
			log.Fatalf("导出报告失败: %v", err)
		}

		if err := exporter.ExportJSON(result, "./win_rate_result.json"); err != nil {
			log.Fatalf("导出 JSON 失败: %v", err)
		}
	*/

	fmt.Println("\n要执行完整评估，请取消注释上面的代码并提供您的 LLM Provider。")
}

// 演示位置随机化
func demoPositionRandomization() {
	fmt.Println("位置随机化说明:")
	fmt.Println()
	fmt.Println("为消除 LLM 的位置偏差（倾向于选择 A 或 B），")
	fmt.Println("Win Rate 评估会随机交换两个问题的位置。")
	fmt.Println()

	fmt.Println("例如：")
	fmt.Println("  原始顺序: [候选问题] vs [参考问题]")
	fmt.Println("  随机交换: [参考问题] vs [候选问题]")
	fmt.Println()

	fmt.Println("评估器会自动处理位置映射，确保正确统计胜负。")
}

// 演示评估标准
func demoEvaluationCriteria() {
	fmt.Println("LLM 评委评估标准:")
	fmt.Println()

	criteria := []string{
		"题目表述清晰度",
		"题目难度适中性",
		"答案准确性",
		"教育价值",
	}

	for i, c := range criteria {
		fmt.Printf("  %d. %s\n", i+1, c)
	}

	fmt.Println("\n评估结果:")
	fmt.Println("  Winner: A  - 问题 A 更好")
	fmt.Println("  Winner: B  - 问题 B 更好")
	fmt.Println("  Winner: Tie - 两者质量相当")
}

// 演示与 AIME 真题对比
func demoAIMEComparison() {
	fmt.Println("与 AIME 真题对比:")
	fmt.Println()
	fmt.Println("AIME (American Invitational Mathematics Examination) 是")
	fmt.Println("美国高中数学竞赛，题目质量被广泛认可。")
	fmt.Println()
	fmt.Println("使用 AIME 真题作为参考数据集，可以评估生成的数学题")
	fmt.Println("与真实竞赛题目的质量差距。")
	fmt.Println()

	fmt.Println("胜率解读:")
	fmt.Println("  > 50%: 生成题目质量超越 AIME (非常优秀)")
	fmt.Println("  40-50%: 接近 AIME 质量 (优秀)")
	fmt.Println("  30-40%: 有一定差距 (良好)")
	fmt.Println("  < 30%: 差距较大 (需改进)")
}

// 使用占位符避免未使用导入
func init() {
	_ = evaluation.WithVerbose(true)
}
