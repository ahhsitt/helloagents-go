// Package main 展示 BFCL 评估使用示例
//
// BFCL (Berkeley Function Calling Leaderboard) 用于评估智能体的函数调用能力。
//
// 使用前请先下载 BFCL 数据集：
//
//	git clone --depth 1 https://github.com/ShishirPatil/gorilla.git temp_gorilla
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/easyops/helloagents-go/pkg/evaluation"
	"github.com/easyops/helloagents-go/pkg/evaluation/benchmarks/bfcl"
)

func main() {
	ctx := context.Background()

	// 1. 创建数据集
	// 数据目录应包含 BFCL v4 格式的数据文件
	dataDir := "./temp_gorilla/berkeley-function-call-leaderboard/bfcl_eval/data"
	category := "simple_python" // 可选：simple_python, multiple, parallel 等

	dataset := bfcl.NewDataset(dataDir, category)

	// 2. 加载数据集
	if err := dataset.Load(ctx); err != nil {
		log.Fatalf("加载数据集失败: %v", err)
	}

	fmt.Printf("数据集: %s\n", dataset.Name())
	fmt.Printf("样本数: %d\n", dataset.Len())

	// 3. 查看第一个样本
	sample, err := dataset.Get(0)
	if err != nil {
		log.Fatalf("获取样本失败: %v", err)
	}

	fmt.Printf("\n样本示例:\n")
	fmt.Printf("  ID: %s\n", sample.ID)
	fmt.Printf("  输入: %s\n", sample.Input[:min(100, len(sample.Input))]+"...")
	fmt.Printf("  工具数: %d\n", len(sample.Tools))

	// 4. 创建评估器
	evaluator := bfcl.NewEvaluator(dataset, bfcl.ModeAST)
	fmt.Printf("\n评估器: %s\n", evaluator.Name())

	// 5. 执行评估（需要实际的 Agent）
	// 下面的代码需要一个实现了 agents.Agent 接口的智能体
	/*
		agent := yourAgent // 您的智能体实现

		result, err := evaluator.Evaluate(ctx, agent,
			evaluation.WithMaxSamples(10),          // 只评估前 10 个样本
			evaluation.WithTimeout(5*time.Minute), // 每个样本 5 分钟超时
			evaluation.WithProgressCallback(func(done, total int) {
				fmt.Printf("进度: %d/%d\n", done, total)
			}),
		)

		if err != nil {
			log.Fatalf("评估失败: %v", err)
		}

		fmt.Printf("\n评估结果:\n")
		fmt.Printf("  总样本数: %d\n", result.TotalSamples)
		fmt.Printf("  成功数: %d\n", result.SuccessCount)
		fmt.Printf("  准确率: %.2f%%\n", result.OverallAccuracy*100)

		// 6. 导出结果
		exporter := bfcl.NewExporter(true)
		if err := exporter.Export(result, "./bfcl_results.jsonl"); err != nil {
			log.Fatalf("导出失败: %v", err)
		}

		if err := exporter.ExportMarkdownReport(result, "./bfcl_report.md"); err != nil {
			log.Fatalf("导出报告失败: %v", err)
		}
	*/

	fmt.Println("\n要执行完整评估，请取消注释上面的代码并提供您的智能体实现。")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 演示评估选项的使用
func demoOptions() {
	// WithMaxSamples - 限制评估样本数
	_ = evaluation.WithMaxSamples(100)

	// WithTimeout - 设置单个样本超时
	// _ = evaluation.WithTimeout(2 * time.Minute)

	// WithProgressCallback - 进度回调
	_ = evaluation.WithProgressCallback(func(done, total int) {
		fmt.Printf("进度: %d/%d (%.1f%%)\n", done, total, float64(done)/float64(total)*100)
	})

	// WithVerbose - 详细输出
	_ = evaluation.WithVerbose(true)

	// WithOutputDir - 设置输出目录
	_ = evaluation.WithOutputDir("./evaluation_results")
}
