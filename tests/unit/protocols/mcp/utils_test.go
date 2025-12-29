package mcp_test

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func TestCreateContext(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)

	if ctx.Messages == nil {
		t.Error("Messages should not be nil")
	}
	if ctx.Tools == nil {
		t.Error("Tools should not be nil")
	}
	if ctx.Resources == nil {
		t.Error("Resources should not be nil")
	}
	if ctx.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestCreateContextWithValues(t *testing.T) {
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello"},
	}
	tools := []map[string]interface{}{
		{"name": "calc", "description": "Calculator"},
	}
	resources := []map[string]interface{}{
		{"uri": "file:///test.txt"},
	}
	metadata := map[string]interface{}{
		"version": "1.0",
	}

	ctx := mcp.CreateContext(messages, tools, resources, metadata)

	if len(ctx.Messages) != 1 {
		t.Errorf("Messages count = %v, want %v", len(ctx.Messages), 1)
	}
	if len(ctx.Tools) != 1 {
		t.Errorf("Tools count = %v, want %v", len(ctx.Tools), 1)
	}
	if len(ctx.Resources) != 1 {
		t.Errorf("Resources count = %v, want %v", len(ctx.Resources), 1)
	}
	if ctx.Metadata["version"] != "1.0" {
		t.Errorf("Metadata version = %v, want %v", ctx.Metadata["version"], "1.0")
	}
}

func TestParseContext(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "valid json string",
			data: `{"messages":[],"tools":[],"resources":[],"metadata":{}}`,
		},
		{
			name: "valid json bytes",
			data: []byte(`{"messages":[],"tools":[]}`),
		},
		{
			name:    "invalid json",
			data:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "invalid type",
			data:    123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := mcp.ParseContext(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ctx == nil {
				t.Error("ParseContext() returned nil context")
			}
		})
	}
}

func TestContextAddMessage(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)
	ctx.AddMessage("user", "Hello")

	if len(ctx.Messages) != 1 {
		t.Errorf("Messages count = %v, want %v", len(ctx.Messages), 1)
	}
	if ctx.Messages[0]["role"] != "user" {
		t.Errorf("Message role = %v, want %v", ctx.Messages[0]["role"], "user")
	}
	if ctx.Messages[0]["content"] != "Hello" {
		t.Errorf("Message content = %v, want %v", ctx.Messages[0]["content"], "Hello")
	}
}

func TestContextAddTool(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)
	ctx.AddTool("calculator", "Performs calculations")

	if len(ctx.Tools) != 1 {
		t.Errorf("Tools count = %v, want %v", len(ctx.Tools), 1)
	}
	if ctx.Tools[0]["name"] != "calculator" {
		t.Errorf("Tool name = %v, want %v", ctx.Tools[0]["name"], "calculator")
	}
}

func TestContextAddResource(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)
	ctx.AddResource("file:///test.txt", "test.txt")

	if len(ctx.Resources) != 1 {
		t.Errorf("Resources count = %v, want %v", len(ctx.Resources), 1)
	}
	if ctx.Resources[0]["uri"] != "file:///test.txt" {
		t.Errorf("Resource URI = %v, want %v", ctx.Resources[0]["uri"], "file:///test.txt")
	}
}

func TestContextMetadata(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)
	ctx.SetMetadata("version", "1.0")

	value, ok := ctx.GetMetadata("version")
	if !ok {
		t.Error("GetMetadata() should return true for existing key")
	}
	if value != "1.0" {
		t.Errorf("GetMetadata() = %v, want %v", value, "1.0")
	}

	_, ok = ctx.GetMetadata("nonexistent")
	if ok {
		t.Error("GetMetadata() should return false for nonexistent key")
	}
}

func TestContextToJSON(t *testing.T) {
	ctx := mcp.CreateContext(nil, nil, nil, nil)
	ctx.AddMessage("user", "Hello")

	jsonStr, err := ctx.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Parse back
	parsed, err := mcp.ParseContext(jsonStr)
	if err != nil {
		t.Fatalf("ParseContext() error = %v", err)
	}

	if len(parsed.Messages) != 1 {
		t.Errorf("Messages count = %v, want %v", len(parsed.Messages), 1)
	}
}

func TestCreateErrorResponse(t *testing.T) {
	resp := mcp.CreateErrorResponse("Something went wrong", "ERROR_001", nil)

	if resp.Error.Message != "Something went wrong" {
		t.Errorf("Error message = %v, want %v", resp.Error.Message, "Something went wrong")
	}
	if resp.Error.Code != "ERROR_001" {
		t.Errorf("Error code = %v, want %v", resp.Error.Code, "ERROR_001")
	}
}

func TestCreateErrorResponseDefaultCode(t *testing.T) {
	resp := mcp.CreateErrorResponse("Something went wrong", "", nil)

	if resp.Error.Code != "UNKNOWN_ERROR" {
		t.Errorf("Error code = %v, want %v", resp.Error.Code, "UNKNOWN_ERROR")
	}
}

func TestCreateSuccessResponse(t *testing.T) {
	data := map[string]interface{}{"result": 42}
	resp := mcp.CreateSuccessResponse(data, nil)

	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestCreateSuccessResponseWithMetadata(t *testing.T) {
	data := map[string]interface{}{"result": 42}
	metadata := map[string]interface{}{"timestamp": "2024-01-01"}
	resp := mcp.CreateSuccessResponse(data, metadata)

	if resp.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if resp.Metadata["timestamp"] != "2024-01-01" {
		t.Errorf("Metadata timestamp = %v, want %v", resp.Metadata["timestamp"], "2024-01-01")
	}
}
