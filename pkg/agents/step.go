package agents

import "time"

// ReasoningStep 表示一个推理步骤
type ReasoningStep struct {
	// Type 步骤类型（thought/action/observation）
	Type StepType `json:"type"`
	// Content 步骤内容
	Content string `json:"content"`
	// ToolName 工具名称（当 Type=action 时）
	ToolName string `json:"tool_name,omitempty"`
	// ToolArgs 工具参数（当 Type=action 时）
	ToolArgs map[string]interface{} `json:"tool_args,omitempty"`
	// ToolResult 工具结果（当 Type=observation 时）
	ToolResult string `json:"tool_result,omitempty"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// StepType 步骤类型
type StepType string

const (
	// StepTypeThought 思考
	StepTypeThought StepType = "thought"
	// StepTypeAction 行动（工具调用）
	StepTypeAction StepType = "action"
	// StepTypeObservation 观察（工具结果）
	StepTypeObservation StepType = "observation"
	// StepTypePlan 计划（PlanAndSolve）
	StepTypePlan StepType = "plan"
	// StepTypeReflection 反思（Reflection）
	StepTypeReflection StepType = "reflection"
)

// NewThoughtStep 创建思考步骤
func NewThoughtStep(content string) ReasoningStep {
	return ReasoningStep{
		Type:      StepTypeThought,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewActionStep 创建行动步骤
func NewActionStep(toolName string, toolArgs map[string]interface{}) ReasoningStep {
	return ReasoningStep{
		Type:      StepTypeAction,
		ToolName:  toolName,
		ToolArgs:  toolArgs,
		Timestamp: time.Now(),
	}
}

// NewObservationStep 创建观察步骤
func NewObservationStep(toolName, result string) ReasoningStep {
	return ReasoningStep{
		Type:       StepTypeObservation,
		ToolName:   toolName,
		ToolResult: result,
		Timestamp:  time.Now(),
	}
}

// NewPlanStep 创建计划步骤
func NewPlanStep(content string) ReasoningStep {
	return ReasoningStep{
		Type:      StepTypePlan,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewReflectionStep 创建反思步骤
func NewReflectionStep(content string) ReasoningStep {
	return ReasoningStep{
		Type:      StepTypeReflection,
		Content:   content,
		Timestamp: time.Now(),
	}
}
