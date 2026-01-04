package gaia

import (
	"testing"
)

func TestEvaluator_ExtractAnswer(t *testing.T) {
	evaluator := &Evaluator{}

	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "FINAL ANSWER 格式",
			response: "经过分析，FINAL ANSWER: 42",
			want:     "42",
		},
		{
			name:     "Answer 格式",
			response: "The Answer: Beijing",
			want:     "Beijing",
		},
		{
			name:     "中文答案格式",
			response: "根据计算，答案：上海",
			want:     "上海",
		},
		{
			name:     "最后一行",
			response: "这是一些解释\n\n最终结果是 100",
			want:     "最终结果是 100",
		},
		{
			name:     "空响应",
			response: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluator.extractAnswer(tt.response)
			if got != tt.want {
				t.Errorf("extractAnswer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvaluator_EvaluateMatch(t *testing.T) {
	evaluator := &Evaluator{}

	tests := []struct {
		name        string
		predicted   string
		expected    string
		wantExact   bool
		wantPartial bool
	}{
		{
			name:        "精确匹配",
			predicted:   "Beijing",
			expected:    "Beijing",
			wantExact:   true,
			wantPartial: true,
		},
		{
			name:        "忽略大小写",
			predicted:   "beijing",
			expected:    "Beijing",
			wantExact:   true,
			wantPartial: true,
		},
		{
			name:        "包含关系",
			predicted:   "The answer is Beijing, China",
			expected:    "Beijing",
			wantExact:   false,
			wantPartial: true,
		},
		{
			name:        "词汇覆盖",
			predicted:   "machine learning is a subset of artificial intelligence",
			expected:    "machine learning artificial intelligence",
			wantExact:   false,
			wantPartial: true,
		},
		{
			name:        "完全不匹配",
			predicted:   "apple",
			expected:    "orange",
			wantExact:   false,
			wantPartial: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExact, gotPartial := evaluator.evaluateMatch(tt.predicted, tt.expected)
			if gotExact != tt.wantExact {
				t.Errorf("evaluateMatch() exactMatch = %v, want %v", gotExact, tt.wantExact)
			}
			if gotPartial != tt.wantPartial {
				t.Errorf("evaluateMatch() partialMatch = %v, want %v", gotPartial, tt.wantPartial)
			}
		})
	}
}

func TestNewDataset(t *testing.T) {
	dataset := NewDataset("/tmp/gaia", 1, "validation")

	if dataset == nil {
		t.Fatal("NewDataset() should return non-nil")
	}

	if dataset.level != 1 {
		t.Errorf("NewDataset() level = %d, want 1", dataset.level)
	}

	if dataset.split != "validation" {
		t.Errorf("NewDataset() split = %s, want validation", dataset.split)
	}
}

func TestDataset_Name(t *testing.T) {
	tests := []struct {
		level    int
		split    string
		wantName string
	}{
		{0, "validation", "GAIA_validation"},
		{1, "validation", "GAIA_validation_Level1"},
		{2, "test", "GAIA_test_Level2"},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			dataset := NewDataset("/tmp/gaia", tt.level, tt.split)
			got := dataset.Name()
			if got != tt.wantName {
				t.Errorf("Name() = %s, want %s", got, tt.wantName)
			}
		})
	}
}

func TestNewEvaluator(t *testing.T) {
	dataset := NewDataset("/tmp/gaia", 1, "validation")
	evaluator := NewEvaluator(dataset)

	if evaluator == nil {
		t.Error("NewEvaluator() should return non-nil")
	}

	name := evaluator.Name()
	if name != "GAIA_validation_Level1" {
		t.Errorf("Name() = %s, want GAIA_validation_Level1", name)
	}
}
