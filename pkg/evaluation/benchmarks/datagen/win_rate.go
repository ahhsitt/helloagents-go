package datagen

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// Win Rate 评估结果常量
const (
	winnerCandidate = "candidate"
	winnerReference = "reference"
	winnerTie       = "tie"
)

// WinRateConfig Win Rate 配置
type WinRateConfig struct {
	// RandomSeed 随机种子（用于位置随机化）
	RandomSeed int64
}

// WinRateEvaluator Win Rate 评估器
type WinRateEvaluator struct {
	// llmProvider LLM 提供商
	llmProvider llm.Provider

	// candidateDataset 候选数据集
	candidateDataset *Dataset

	// referenceDataset 参考数据集
	referenceDataset *Dataset

	// config 配置
	config WinRateConfig

	// rand 随机数生成器
	rand *rand.Rand
}

// NewWinRateEvaluator 创建 Win Rate 评估器
//
// 参数:
//   - llmProvider: LLM 服务提供商
//   - candidateDataset: 候选数据集（待评估）
//   - referenceDataset: 参考数据集（基准）
//   - config: 评估配置
func NewWinRateEvaluator(llmProvider llm.Provider, candidateDataset, referenceDataset *Dataset, config WinRateConfig) *WinRateEvaluator {
	seed := config.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &WinRateEvaluator{
		llmProvider:      llmProvider,
		candidateDataset: candidateDataset,
		referenceDataset: referenceDataset,
		config:           config,
		rand:             rand.New(rand.NewSource(seed)), //nolint:gosec // 位置随机化不需要加密安全的随机数
	}
}

// Name 返回评估器名称
func (w *WinRateEvaluator) Name() string {
	return "WinRate"
}

// Evaluate 执行完整评估
func (w *WinRateEvaluator) Evaluate(ctx context.Context, opts ...evaluation.EvalOption) (*evaluation.EvalResult, error) {
	config := evaluation.DefaultEvalConfig()
	config.ApplyOptions(opts...)

	// 确保数据集已加载
	if err := w.candidateDataset.Load(ctx); err != nil {
		return nil, fmt.Errorf("加载候选数据集失败: %w", err)
	}
	if err := w.referenceDataset.Load(ctx); err != nil {
		return nil, fmt.Errorf("加载参考数据集失败: %w", err)
	}

	startTime := time.Now()
	result := &evaluation.EvalResult{
		BenchmarkName:   w.Name(),
		AgentName:       w.llmProvider.Name(),
		DetailedResults: make([]*evaluation.SampleResult, 0),
		EvaluationTime:  startTime,
	}

	// 确定比较数量
	total := w.candidateDataset.Len()
	if w.referenceDataset.Len() < total {
		total = w.referenceDataset.Len()
	}
	if config.MaxSamples > 0 && config.MaxSamples < total {
		total = config.MaxSamples
	}
	result.TotalSamples = total

	// 统计胜负平
	wins, losses, ties := 0, 0, 0

	// 遍历样本进行对比
	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		candidateSample, err := w.candidateDataset.Get(i)
		if err != nil {
			continue
		}
		referenceSample, err := w.referenceDataset.Get(i)
		if err != nil {
			continue
		}

		// 应用超时
		evalCtx := ctx
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			evalCtx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		sampleResult, err := w.CompareSamples(evalCtx, candidateSample, referenceSample)
		if err != nil {
			sampleResult = &evaluation.SampleResult{
				SampleID: candidateSample.ID,
				Error:    err.Error(),
			}
		}

		result.DetailedResults = append(result.DetailedResults, sampleResult)

		// 统计胜负
		if compResult, ok := sampleResult.Predicted.(*evaluation.ComparisonResult); ok {
			switch compResult.ActualWinner {
			case winnerCandidate:
				wins++
				sampleResult.Success = true
			case winnerReference:
				losses++
			case winnerTie:
				ties++
			}
		}

		// 进度回调
		if config.ProgressCallback != nil {
			config.ProgressCallback(i+1, total)
		}
	}

	result.TotalDuration = time.Since(startTime)
	result.SuccessCount = wins

	// 计算汇总指标
	result.Metrics = w.computeMetrics(wins, losses, ties, total)

	return result, nil
}

// CompareSamples 比较两个样本
func (w *WinRateEvaluator) CompareSamples(ctx context.Context, candidate, reference evaluation.Sample) (*evaluation.SampleResult, error) {
	startTime := time.Now()

	result := &evaluation.SampleResult{
		SampleID: candidate.ID,
		Details:  make(map[string]interface{}),
	}

	// 随机决定位置
	swapped := w.rand.Float32() < 0.5

	var problemA, problemB evaluation.Sample
	if swapped {
		problemA, problemB = reference, candidate
	} else {
		problemA, problemB = candidate, reference
	}

	// 构建对比提示
	prompt := w.buildComparePrompt(problemA, problemB)

	// 调用 LLM
	req := llm.Request{
		Messages: []message.Message{
			message.NewSystemMessage(w.getSystemPrompt()),
			message.NewUserMessage(prompt),
		},
	}

	resp, err := w.llmProvider.Generate(ctx, req)
	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	result.AgentResponse = resp.Content
	result.ExecutionTime = time.Since(startTime)

	// 解析结果
	compResult := w.parseCompareResponse(resp.Content, candidate.ID, reference.ID, swapped)
	result.Predicted = compResult

	result.Details["winner"] = compResult.Winner
	result.Details["actual_winner"] = compResult.ActualWinner
	result.Details["reason"] = compResult.Reason
	result.Details["swapped"] = swapped

	return result, nil
}

// getSystemPrompt 获取系统提示
func (w *WinRateEvaluator) getSystemPrompt() string {
	return `你是一个专业的题目质量评估专家。请比较两道题目，选择质量更好的一道。

评估标准：
1. 题目表述清晰度
2. 题目难度适中性
3. 答案准确性
4. 教育价值

请以以下格式回复：
Winner: [A/B/Tie]
Reason: <选择理由>`
}

// buildComparePrompt 构建对比提示
func (w *WinRateEvaluator) buildComparePrompt(problemA, problemB evaluation.Sample) string {
	prompt := "## 题目 A\n\n"
	prompt += fmt.Sprintf("**问题**: %s\n", problemA.Input)
	if answer, ok := problemA.Expected.(string); ok && answer != "" {
		prompt += fmt.Sprintf("**答案**: %s\n", answer)
	}

	prompt += "\n---\n\n## 题目 B\n\n"
	prompt += fmt.Sprintf("**问题**: %s\n", problemB.Input)
	if answer, ok := problemB.Expected.(string); ok && answer != "" {
		prompt += fmt.Sprintf("**答案**: %s\n", answer)
	}

	prompt += "\n请比较以上两道题目，选择质量更好的一道。"

	return prompt
}

// parseCompareResponse 解析对比响应
func (w *WinRateEvaluator) parseCompareResponse(response, candidateID, referenceID string, swapped bool) *evaluation.ComparisonResult {
	result := &evaluation.ComparisonResult{
		ProblemAID: candidateID,
		ProblemBID: referenceID,
	}

	// 提取 Winner
	winnerPattern := regexp.MustCompile(`(?i)Winner:\s*([ABTie]+)`)
	matches := winnerPattern.FindStringSubmatch(response)
	if len(matches) > 1 {
		result.Winner = strings.TrimSpace(strings.ToUpper(matches[1]))
	}

	// 提取 Reason
	reasonPattern := regexp.MustCompile(`(?i)Reason:\s*(.+?)(?:\n|$)`)
	reasonMatches := reasonPattern.FindStringSubmatch(response)
	if len(reasonMatches) > 1 {
		result.Reason = strings.TrimSpace(reasonMatches[1])
	}

	// 处理 Tie 情况
	if strings.Contains(strings.ToLower(result.Winner), "tie") {
		result.Winner = "Tie"
		result.ActualWinner = winnerTie
		return result
	}

	// 映射回实际胜者
	if result.Winner == "A" {
		if swapped {
			result.ActualWinner = winnerReference
		} else {
			result.ActualWinner = winnerCandidate
		}
	} else if result.Winner == "B" {
		if swapped {
			result.ActualWinner = winnerCandidate
		} else {
			result.ActualWinner = winnerReference
		}
	} else {
		result.ActualWinner = winnerTie
	}

	return result
}

// computeMetrics 计算汇总指标
func (w *WinRateEvaluator) computeMetrics(wins, losses, ties, total int) *evaluation.MetricsSummary {
	summary := &evaluation.MetricsSummary{
		Extra: make(map[string]interface{}),
	}

	if total == 0 {
		return summary
	}

	summary.WinRate = float64(wins) / float64(total)
	summary.LossRate = float64(losses) / float64(total)
	summary.TieRate = float64(ties) / float64(total)
	summary.Accuracy = summary.WinRate

	summary.Extra["total_comparisons"] = total
	summary.Extra["wins"] = wins
	summary.Extra["losses"] = losses
	summary.Extra["ties"] = ties

	return summary
}
