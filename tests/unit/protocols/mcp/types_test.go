package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func TestJSONRPCRequest(t *testing.T) {
	req := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      1,
		Method:  mcp.MethodListTools,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var parsed mcp.JSONRPCRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if parsed.JSONRPC != mcp.JSONRPCVersion {
		t.Errorf("JSONRPC = %v, want %v", parsed.JSONRPC, mcp.JSONRPCVersion)
	}
	if parsed.Method != mcp.MethodListTools {
		t.Errorf("Method = %v, want %v", parsed.Method, mcp.MethodListTools)
	}
}

func TestJSONRPCResponse(t *testing.T) {
	resp := mcp.JSONRPCResponse{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      1,
		Result:  json.RawMessage(`{"tools":[]}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed mcp.JSONRPCResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed.Error != nil {
		t.Errorf("Unexpected error in response")
	}
}

func TestJSONRPCError(t *testing.T) {
	resp := mcp.JSONRPCResponse{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      1,
		Error: &mcp.JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed mcp.JSONRPCResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed.Error == nil {
		t.Fatal("Expected error in response")
	}
	if parsed.Error.Code != -32600 {
		t.Errorf("Error code = %v, want %v", parsed.Error.Code, -32600)
	}
}

func TestToolInfo(t *testing.T) {
	tool := mcp.ToolInfo{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input value",
				},
			},
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("Name = %v, want %v", tool.Name, "test_tool")
	}
}

func TestResourceInfo(t *testing.T) {
	resource := mcp.ResourceInfo{
		URI:         "file:///test.txt",
		Name:        "test.txt",
		Description: "A test file",
		MimeType:    "text/plain",
	}

	if resource.URI != "file:///test.txt" {
		t.Errorf("URI = %v, want %v", resource.URI, "file:///test.txt")
	}
}

func TestPromptInfo(t *testing.T) {
	prompt := mcp.PromptInfo{
		Name:        "test_prompt",
		Description: "A test prompt",
		Arguments: []mcp.PromptArgument{
			{Name: "arg1", Description: "First argument", Required: true},
		},
	}

	if prompt.Name != "test_prompt" {
		t.Errorf("Name = %v, want %v", prompt.Name, "test_prompt")
	}
	if len(prompt.Arguments) != 1 {
		t.Errorf("Arguments count = %v, want %v", len(prompt.Arguments), 1)
	}
}

func TestCapabilities(t *testing.T) {
	caps := mcp.Capabilities{
		Tools:     &mcp.ToolsCapability{ListChanged: true},
		Resources: &mcp.ResourcesCapability{Subscribe: true, ListChanged: true},
		Prompts:   &mcp.PromptsCapability{ListChanged: false},
	}

	if caps.Tools == nil || !caps.Tools.ListChanged {
		t.Error("Tools capability not set correctly")
	}
	if caps.Resources == nil || !caps.Resources.Subscribe {
		t.Error("Resources capability not set correctly")
	}
}

func TestCallToolParams(t *testing.T) {
	params := mcp.CallToolParams{
		Name: "calculator",
		Arguments: map[string]interface{}{
			"expression": "2 + 2",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	var parsed mcp.CallToolParams
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal params: %v", err)
	}

	if parsed.Name != "calculator" {
		t.Errorf("Name = %v, want %v", parsed.Name, "calculator")
	}
}

func TestCallToolResult(t *testing.T) {
	result := mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: "Result: 4"},
		},
		IsError: false,
	}

	if result.IsError {
		t.Error("IsError should be false")
	}
	if len(result.Content) != 1 {
		t.Errorf("Content count = %v, want %v", len(result.Content), 1)
	}
	if result.Content[0].Text != "Result: 4" {
		t.Errorf("Content text = %v, want %v", result.Content[0].Text, "Result: 4")
	}
}
