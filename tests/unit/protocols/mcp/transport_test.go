package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/easyops/helloagents-go/pkg/protocols/mcp"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name    string
		id      interface{}
		method  string
		params  interface{}
		wantErr bool
	}{
		{
			name:   "simple request",
			id:     1,
			method: "test/method",
			params: nil,
		},
		{
			name:   "request with params",
			id:     2,
			method: "test/method",
			params: map[string]string{"key": "value"},
		},
		{
			name:   "request with string id",
			id:     "req-1",
			method: "test/method",
			params: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := mcp.NewRequest(tt.id, tt.method, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var req mcp.JSONRPCRequest
				if err := json.Unmarshal(data, &req); err != nil {
					t.Errorf("Failed to unmarshal request: %v", err)
					return
				}

				if req.JSONRPC != mcp.JSONRPCVersion {
					t.Errorf("JSONRPC version = %v, want %v", req.JSONRPC, mcp.JSONRPCVersion)
				}
				if req.Method != tt.method {
					t.Errorf("Method = %v, want %v", req.Method, tt.method)
				}
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantErr  bool
		hasError bool
	}{
		{
			name: "success response",
			data: `{"jsonrpc":"2.0","id":1,"result":{"key":"value"}}`,
		},
		{
			name:     "error response",
			data:     `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`,
			hasError: true,
		},
		{
			name:    "invalid json",
			data:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := mcp.ParseResponse([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if (resp.Error != nil) != tt.hasError {
					t.Errorf("Response has error = %v, want %v", resp.Error != nil, tt.hasError)
				}
			}
		})
	}
}

func TestMemoryTransport(t *testing.T) {
	handler := func(request []byte) ([]byte, error) {
		return []byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`), nil
	}

	transport := mcp.NewMemoryTransport(handler)

	ctx := context.Background()
	response, err := transport.Send(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"test"}`))
	if err != nil {
		t.Errorf("Send() error = %v", err)
		return
	}

	resp, err := mcp.ParseResponse(response)
	if err != nil {
		t.Errorf("ParseResponse() error = %v", err)
		return
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error in response: %v", resp.Error)
	}
}

func TestHTTPTransportConfig(t *testing.T) {
	config := mcp.HTTPTransportConfig{
		URL: "http://localhost:8080",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	transport := mcp.NewHTTPTransport(config)
	_ = transport // Just verify it compiles
}
