package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/easyops/helloagents-go/pkg/tools"
)

func TestNewFuncTool(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"test-tool",
		"Test tool description",
		tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"input": {Type: "string", Description: "Input value"},
			},
			Required: []string{"input"},
		},
		fn,
	)

	if tool.Name() != "test-tool" {
		t.Fatalf("expected name 'test-tool', got %s", tool.Name())
	}
	if tool.Description() != "Test tool description" {
		t.Fatalf("expected description 'Test tool description', got %s", tool.Description())
	}
}

func TestFuncTool_Parameters(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	params := tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"name":    {Type: "string", Description: "Name"},
			"age":     {Type: "integer", Description: "Age"},
			"enabled": {Type: "boolean", Description: "Enabled flag"},
		},
		Required: []string{"name"},
	}

	tool := tools.NewFuncTool("test", "desc", params, fn)

	schema := tool.Parameters()
	if schema.Type != "object" {
		t.Fatalf("expected type 'object', got %s", schema.Type)
	}
	if len(schema.Properties) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(schema.Properties))
	}
	if len(schema.Required) != 1 {
		t.Fatalf("expected 1 required, got %d", len(schema.Required))
	}
}

func TestFuncTool_Execute(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		name := args["name"].(string)
		return "Hello, " + name, nil
	}

	tool := tools.NewFuncTool(
		"greet",
		"Greet user",
		tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"name": {Type: "string"},
			},
			Required: []string{"name"},
		},
		fn,
	)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"name": "World",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Hello, World" {
		t.Fatalf("expected 'Hello, World', got %s", result)
	}
}

func TestFuncTool_ExecuteWithError(t *testing.T) {
	expectedErr := errors.New("execution failed")
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "", expectedErr
	}

	tool := tools.NewFuncTool(
		"failing-tool",
		"A tool that fails",
		tools.ParameterSchema{Type: "object"},
		fn,
	)

	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != expectedErr {
		t.Fatalf("expected specific error, got %v", err)
	}
}

func TestFuncTool_ExecuteWithContext(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "completed", nil
		}
	}

	tool := tools.NewFuncTool(
		"ctx-tool",
		"Context aware tool",
		tools.ParameterSchema{Type: "object"},
		fn,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := tool.Execute(ctx, map[string]interface{}{})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFuncTool_Validate(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"validate-tool",
		"Tool with validation",
		tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"required_param": {Type: "string"},
			},
			Required: []string{"required_param"},
		},
		fn,
	)

	// Test with missing required param
	err := tool.Validate(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing required param")
	}

	// Test with required param present
	err = tool.Validate(map[string]interface{}{"required_param": "value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFuncTool_WithValidator(t *testing.T) {
	customValidator := func(args map[string]interface{}) error {
		if val, ok := args["value"].(int); ok && val < 0 {
			return errors.New("value must be non-negative")
		}
		return nil
	}

	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"validated-tool",
		"Tool with custom validator",
		tools.ParameterSchema{Type: "object"},
		fn,
		tools.WithValidator(customValidator),
	)

	// Test invalid value
	err := tool.Validate(map[string]interface{}{"value": -1})
	if err == nil {
		t.Fatal("expected error for negative value")
	}

	// Test valid value
	err = tool.Validate(map[string]interface{}{"value": 5})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestFuncTool_ImplementsTool(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"test",
		"test",
		tools.ParameterSchema{Type: "object"},
		fn,
	)

	var _ tools.Tool = tool
	var _ tools.ToolWithValidation = tool
}

func TestNewSimpleTool(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return "Echo: " + input, nil
	}

	tool := tools.NewSimpleTool(
		"echo",
		"Echo input",
		"message",
		"Message to echo",
		fn,
	)

	if tool.Name() != "echo" {
		t.Fatalf("expected name 'echo', got %s", tool.Name())
	}
	if tool.Description() != "Echo input" {
		t.Fatalf("expected description 'Echo input', got %s", tool.Description())
	}
}

func TestSimpleTool_Parameters(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	tool := tools.NewSimpleTool(
		"test",
		"Test tool",
		"query",
		"Query string",
		fn,
	)

	params := tool.Parameters()
	if params.Type != "object" {
		t.Fatalf("expected type 'object', got %s", params.Type)
	}
	if _, ok := params.Properties["query"]; !ok {
		t.Fatal("expected 'query' property")
	}
	if len(params.Required) != 1 || params.Required[0] != "query" {
		t.Fatal("expected 'query' to be required")
	}
}

func TestSimpleTool_Execute(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return "Processed: " + input, nil
	}

	tool := tools.NewSimpleTool(
		"process",
		"Process input",
		"data",
		"Data to process",
		fn,
	)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"data": "test input",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "Processed: test input" {
		t.Fatalf("expected 'Processed: test input', got %s", result)
	}
}

func TestSimpleTool_ExecuteMissingParam(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	tool := tools.NewSimpleTool(
		"test",
		"Test",
		"input",
		"Input",
		fn,
	)

	_, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err == nil {
		t.Fatal("expected error for missing param")
	}
}

func TestSimpleTool_ExecuteWrongType(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	tool := tools.NewSimpleTool(
		"test",
		"Test",
		"input",
		"Input",
		fn,
	)

	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"input": 123, // Wrong type - should be string
	})

	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestSimpleTool_ImplementsTool(t *testing.T) {
	fn := func(ctx context.Context, input string) (string, error) {
		return input, nil
	}

	tool := tools.NewSimpleTool("test", "test", "input", "input", fn)

	var _ tools.Tool = tool
}

func TestToDefinition(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"my-tool",
		"My tool description",
		tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"param": {Type: "string", Description: "A param"},
			},
		},
		fn,
	)

	def := tools.ToDefinition(tool)

	if def.Name != "my-tool" {
		t.Fatalf("expected name 'my-tool', got %s", def.Name)
	}
	if def.Description != "My tool description" {
		t.Fatalf("expected description 'My tool description', got %s", def.Description)
	}
	if def.Parameters.Type != "object" {
		t.Fatalf("expected parameters type 'object', got %s", def.Parameters.Type)
	}
}

func TestToLLMToolDefinition(t *testing.T) {
	fn := func(ctx context.Context, args map[string]interface{}) (string, error) {
		return "result", nil
	}

	tool := tools.NewFuncTool(
		"llm-tool",
		"LLM tool",
		tools.ParameterSchema{
			Type: "object",
			Properties: map[string]tools.PropertySchema{
				"query": {Type: "string"},
			},
			Required: []string{"query"},
		},
		fn,
	)

	llmDef := tools.ToLLMToolDefinition(tool)

	if llmDef["name"] != "llm-tool" {
		t.Fatalf("expected name 'llm-tool', got %v", llmDef["name"])
	}
	if llmDef["description"] != "LLM tool" {
		t.Fatalf("expected description 'LLM tool', got %v", llmDef["description"])
	}
	if llmDef["parameters"] == nil {
		t.Fatal("expected parameters to be non-nil")
	}
}

func TestToolResult(t *testing.T) {
	result := tools.NewToolResult("my-tool", "success result")

	if result.Name != "my-tool" {
		t.Fatalf("expected name 'my-tool', got %s", result.Name)
	}
	if !result.Success {
		t.Fatal("expected success to be true")
	}
	if result.Result != "success result" {
		t.Fatalf("expected result 'success result', got %s", result.Result)
	}
}

func TestToolError(t *testing.T) {
	err := errors.New("something went wrong")
	result := tools.NewToolError("failing-tool", err)

	if result.Name != "failing-tool" {
		t.Fatalf("expected name 'failing-tool', got %s", result.Name)
	}
	if result.Success {
		t.Fatal("expected success to be false")
	}
	if result.Error != "something went wrong" {
		t.Fatalf("expected error 'something went wrong', got %s", result.Error)
	}
}
