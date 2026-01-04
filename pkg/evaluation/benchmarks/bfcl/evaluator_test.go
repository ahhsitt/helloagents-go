package bfcl

import (
	"context"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/core/config"
	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// MockAgent 用于测试的 Mock Agent
type MockAgent struct {
	name     string
	response string
}

func NewMockAgent(name, response string) *MockAgent {
	return &MockAgent{name: name, response: response}
}

func (m *MockAgent) Name() string {
	return m.name
}

func (m *MockAgent) Config() config.AgentConfig {
	return config.AgentConfig{}
}

func (m *MockAgent) Run(ctx context.Context, input agents.Input) (agents.Output, error) {
	return agents.Output{
		Response: m.response,
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockAgent) RunStream(ctx context.Context, input agents.Input) (<-chan agents.StreamChunk, <-chan error) {
	ch := make(chan agents.StreamChunk)
	errCh := make(chan error)
	go func() {
		ch <- agents.StreamChunk{Content: m.response, Done: true}
		close(ch)
		close(errCh)
	}()
	return ch, errCh
}

func TestEvaluator_ExtractFunctionCalls(t *testing.T) {
	evaluator := &Evaluator{}

	tests := []struct {
		name     string
		response string
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "JSON 数组",
			response: `[{"name": "get_weather", "arguments": {"city": "Beijing"}}]`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "单个对象",
			response: `{"name": "get_weather", "arguments": {"city": "Shanghai"}}`,
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "Markdown 代码块",
			response: "```json\n[{\"name\": \"search\", \"arguments\": {\"query\": \"test\"}}]\n```",
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "多个函数调用",
			response: `[{"name": "func1", "arguments": {}}, {"name": "func2", "arguments": {}}]`,
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "空响应",
			response: "",
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "无效 JSON",
			response: "这是一段普通文本，没有函数调用",
			wantLen:  0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := evaluator.extractFunctionCalls(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractFunctionCalls() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(calls) != tt.wantLen {
				t.Errorf("extractFunctionCalls() got %d calls, want %d", len(calls), tt.wantLen)
			}
		})
	}
}

func TestEvaluator_CompareValues(t *testing.T) {
	evaluator := &Evaluator{}

	tests := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{"相同字符串", "hello", "hello", true},
		{"忽略大小写", "Hello", "hello", true},
		{"相同数字", 42, 42, true},
		{"数字与字符串", 42, "42", true},
		{"浮点数", 3.14, 3.14, true},
		{"不同值", "a", "b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluator.compareValues(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareValues(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEvaluator_CompareFunctionCall(t *testing.T) {
	evaluator := &Evaluator{}

	tests := []struct {
		name      string
		predicted evaluation.FunctionCall
		expected  evaluation.FunctionCall
		wantScore float64
	}{
		{
			name:      "完全匹配",
			predicted: evaluation.FunctionCall{Name: "get_weather", Arguments: map[string]interface{}{"city": "Beijing"}},
			expected:  evaluation.FunctionCall{Name: "get_weather", Arguments: map[string]interface{}{"city": "Beijing"}},
			wantScore: 1.0,
		},
		{
			name:      "函数名不匹配",
			predicted: evaluation.FunctionCall{Name: "get_weather", Arguments: map[string]interface{}{}},
			expected:  evaluation.FunctionCall{Name: "search", Arguments: map[string]interface{}{}},
			wantScore: 0,
		},
		{
			name:      "部分参数匹配",
			predicted: evaluation.FunctionCall{Name: "func", Arguments: map[string]interface{}{"a": "1", "b": "wrong"}},
			expected:  evaluation.FunctionCall{Name: "func", Arguments: map[string]interface{}{"a": "1", "b": "2"}},
			wantScore: 0.5,
		},
		{
			name:      "无参数",
			predicted: evaluation.FunctionCall{Name: "func", Arguments: map[string]interface{}{}},
			expected:  evaluation.FunctionCall{Name: "func", Arguments: map[string]interface{}{}},
			wantScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluator.compareFunctionCall(tt.predicted, tt.expected)
			if got != tt.wantScore {
				t.Errorf("compareFunctionCall() = %v, want %v", got, tt.wantScore)
			}
		})
	}
}

func TestEvaluator_ParseGroundTruth(t *testing.T) {
	evaluator := &Evaluator{}

	// BFCL v4 格式
	gt := map[string]interface{}{
		"get_weather": map[string]interface{}{
			"city": []interface{}{"Beijing", "北京"},
		},
	}

	calls, err := evaluator.parseGroundTruth(gt)
	if err != nil {
		t.Errorf("parseGroundTruth() error = %v", err)
		return
	}

	if len(calls) != 1 {
		t.Errorf("parseGroundTruth() got %d calls, want 1", len(calls))
		return
	}

	if calls[0].Name != "get_weather" {
		t.Errorf("parseGroundTruth() got name %s, want get_weather", calls[0].Name)
	}

	// 验证参数取第一个可接受值
	if calls[0].Arguments["city"] != "Beijing" {
		t.Errorf("parseGroundTruth() got city %v, want Beijing", calls[0].Arguments["city"])
	}
}

func TestNewEvaluator(t *testing.T) {
	dataset := NewDataset("/tmp/bfcl", "simple_python")
	evaluator := NewEvaluator(dataset, ModeAST)

	if evaluator == nil {
		t.Fatal("NewEvaluator() should return non-nil")
	}

	if evaluator.mode != ModeAST {
		t.Errorf("NewEvaluator() mode = %v, want %v", evaluator.mode, ModeAST)
	}
}

func TestEvaluator_Name(t *testing.T) {
	dataset := NewDataset("/tmp/bfcl", "simple_python")
	evaluator := NewEvaluator(dataset, ModeAST)

	name := evaluator.Name()
	expected := "BFCL_simple_python_ast"
	if name != expected {
		t.Errorf("Name() = %s, want %s", name, expected)
	}
}
