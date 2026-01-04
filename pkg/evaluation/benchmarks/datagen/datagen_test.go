package datagen

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/evaluation"
)

func TestLLMJudge_ParseJudgeResponse(t *testing.T) {
	judge := &LLMJudge{}

	tests := []struct {
		name      string
		response  string
		wantScore float64
	}{
		{
			name: "标准 JSON",
			response: `{
				"correctness": 4.5,
				"clarity": 4.0,
				"difficulty_match": 3.5,
				"completeness": 4.0,
				"comments": "Good quality"
			}`,
			wantScore: 4.0, // (4.5 + 4.0 + 3.5 + 4.0) / 4
		},
		{
			name:      "Markdown 代码块",
			response:  "```json\n{\"correctness\": 5, \"clarity\": 5, \"difficulty_match\": 5, \"completeness\": 5}\n```",
			wantScore: 5.0,
		},
		{
			name:      "无效响应使用默认值",
			response:  "无法解析的响应",
			wantScore: 3.0, // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := judge.parseJudgeResponse(tt.response)
			if score.TotalScore != tt.wantScore {
				t.Errorf("parseJudgeResponse() TotalScore = %v, want %v", score.TotalScore, tt.wantScore)
			}
		})
	}
}

func TestLLMJudge_ComputeMetrics(t *testing.T) {
	judge := &LLMJudge{}

	results := []*evaluation.SampleResult{
		{
			SampleID: "test_001",
			Success:  true,
			Score:    4.5,
			Details: map[string]interface{}{
				"correctness":      4.5,
				"clarity":          4.5,
				"difficulty_match": 4.5,
				"completeness":     4.5,
			},
		},
		{
			SampleID: "test_002",
			Success:  false,
			Score:    2.5,
			Details: map[string]interface{}{
				"correctness":      2.5,
				"clarity":          2.5,
				"difficulty_match": 2.5,
				"completeness":     2.5,
			},
		},
	}

	summary := judge.computeMetrics(results)

	if summary.AverageScore != 3.5 {
		t.Errorf("computeMetrics() AverageScore = %v, want 3.5", summary.AverageScore)
	}

	if summary.PassRate != 0.5 {
		t.Errorf("computeMetrics() PassRate = %v, want 0.5", summary.PassRate)
	}

	if summary.DimensionScores["correctness"] != 3.5 {
		t.Errorf("computeMetrics() correctness = %v, want 3.5", summary.DimensionScores["correctness"])
	}
}

func TestWinRateEvaluator_ParseCompareResponse(t *testing.T) {
	evaluator := &WinRateEvaluator{}

	tests := []struct {
		name       string
		response   string
		swapped    bool
		wantWinner string
		wantActual string
	}{
		{
			name:       "A 胜出未交换",
			response:   "Winner: A\nReason: Better clarity",
			swapped:    false,
			wantWinner: "A",
			wantActual: "candidate",
		},
		{
			name:       "A 胜出已交换",
			response:   "Winner: A\nReason: Better clarity",
			swapped:    true,
			wantWinner: "A",
			wantActual: "reference",
		},
		{
			name:       "B 胜出未交换",
			response:   "Winner: B\nReason: More complete",
			swapped:    false,
			wantWinner: "B",
			wantActual: "reference",
		},
		{
			name:       "平局",
			response:   "Winner: Tie\nReason: Both are equally good",
			swapped:    false,
			wantWinner: "Tie",
			wantActual: "tie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.parseCompareResponse(tt.response, "candidate_1", "reference_1", tt.swapped)
			if result.Winner != tt.wantWinner {
				t.Errorf("parseCompareResponse() Winner = %v, want %v", result.Winner, tt.wantWinner)
			}
			if result.ActualWinner != tt.wantActual {
				t.Errorf("parseCompareResponse() ActualWinner = %v, want %v", result.ActualWinner, tt.wantActual)
			}
		})
	}
}

func TestWinRateEvaluator_ComputeMetrics(t *testing.T) {
	evaluator := &WinRateEvaluator{}

	summary := evaluator.computeMetrics(6, 3, 1, 10)

	if summary.WinRate != 0.6 {
		t.Errorf("computeMetrics() WinRate = %v, want 0.6", summary.WinRate)
	}

	if summary.LossRate != 0.3 {
		t.Errorf("computeMetrics() LossRate = %v, want 0.3", summary.LossRate)
	}

	if summary.TieRate != 0.1 {
		t.Errorf("computeMetrics() TieRate = %v, want 0.1", summary.TieRate)
	}
}

func TestNewDataset(t *testing.T) {
	dataset := NewDataset("/tmp/data.jsonl")

	if dataset == nil {
		t.Fatal("NewDataset() should return non-nil")
	}

	if dataset.dataPath != "/tmp/data.jsonl" {
		t.Errorf("NewDataset() dataPath = %s, want /tmp/data.jsonl", dataset.dataPath)
	}
}
