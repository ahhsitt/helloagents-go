package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Transport 传输层接口
//
// 传输层负责将 JSON-RPC 请求发送到 MCP 服务器，并返回响应。
// 不同的传输实现支持不同的通信方式。
type Transport interface {
	// Send 发送请求并返回响应
	Send(ctx context.Context, request []byte) ([]byte, error)
	// Close 关闭传输连接
	Close() error
}

// StdioTransport 标准输入输出传输
//
// 通过启动子进程，使用标准输入/输出与 MCP 服务器通信。
// 这是最常见的本地 MCP 服务器连接方式。
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	mu        sync.Mutex
	closed    atomic.Bool
	closeOnce sync.Once
}

// StdioTransportConfig Stdio 传输配置
type StdioTransportConfig struct {
	// Command 要执行的命令
	Command string
	// Args 命令参数
	Args []string
	// Env 环境变量（会与系统环境变量合并）
	Env map[string]string
	// Dir 工作目录
	Dir string
}

// NewStdioTransport 创建 Stdio 传输
func NewStdioTransport(config StdioTransportConfig) (*StdioTransport, error) {
	// #nosec G204 - Command is provided by trusted configuration, not user input
	cmd := exec.Command(config.Command, config.Args...)

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if config.Dir != "" {
		cmd.Dir = config.Dir
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// 将 stderr 重定向到 os.Stderr 以便调试
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line size

	return &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: scanner,
	}, nil
}

// Send 发送请求并返回响应
func (t *StdioTransport) Send(ctx context.Context, request []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed.Load() {
		return nil, fmt.Errorf("transport is closed")
	}

	// 写入请求（以换行符结尾）
	if _, err := t.stdin.Write(append(request, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// 在上下文中读取响应
	responseCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		if t.scanner.Scan() {
			responseCh <- t.scanner.Bytes()
		} else {
			if err := t.scanner.Err(); err != nil {
				errCh <- fmt.Errorf("failed to read response: %w", err)
			} else {
				errCh <- fmt.Errorf("unexpected end of output")
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case response := <-responseCh:
		// 复制响应数据，因为 scanner 的缓冲区会被重用
		result := make([]byte, len(response))
		copy(result, response)
		return result, nil
	}
}

// Close 关闭传输
func (t *StdioTransport) Close() error {
	var closeErr error

	t.closeOnce.Do(func() {
		t.closed.Store(true)

		// 关闭 stdin，这会通知子进程结束
		if err := t.stdin.Close(); err != nil {
			closeErr = fmt.Errorf("failed to close stdin: %w", err)
		}

		// 等待进程结束
		if err := t.cmd.Wait(); err != nil {
			// 进程可能因为 stdin 关闭而正常退出
			// 只有非预期的错误才需要报告
			if closeErr == nil && err.Error() != "signal: killed" {
				closeErr = fmt.Errorf("process exited with error: %w", err)
			}
		}
	})

	return closeErr
}

// HTTPTransport HTTP 传输
//
// 通过 HTTP POST 请求与远程 MCP 服务器通信。
type HTTPTransport struct {
	url     string
	client  *http.Client
	headers map[string]string
}

// HTTPTransportConfig HTTP 传输配置
type HTTPTransportConfig struct {
	// URL 服务器 URL
	URL string
	// Headers 自定义请求头
	Headers map[string]string
	// Client 自定义 HTTP 客户端（可选）
	Client *http.Client
}

// NewHTTPTransport 创建 HTTP 传输
func NewHTTPTransport(config HTTPTransportConfig) *HTTPTransport {
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPTransport{
		url:     config.URL,
		client:  client,
		headers: config.Headers,
	}
}

// Send 发送 HTTP 请求
func (t *HTTPTransport) Send(ctx context.Context, request []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(request))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

// Close 关闭 HTTP 传输（无操作）
func (t *HTTPTransport) Close() error {
	return nil
}

// MemoryTransport 内存传输（用于测试）
//
// 直接调用处理函数，不涉及网络或进程通信。
type MemoryTransport struct {
	handler func(request []byte) ([]byte, error)
}

// NewMemoryTransport 创建内存传输
func NewMemoryTransport(handler func(request []byte) ([]byte, error)) *MemoryTransport {
	return &MemoryTransport{handler: handler}
}

// Send 直接调用处理函数
func (t *MemoryTransport) Send(ctx context.Context, request []byte) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return t.handler(request)
	}
}

// Close 关闭内存传输（无操作）
func (t *MemoryTransport) Close() error {
	return nil
}

// NewRequest 创建 JSON-RPC 请求
func NewRequest(id interface{}, method string, params interface{}) ([]byte, error) {
	var paramsRaw json.RawMessage
	if params != nil {
		var err error
		paramsRaw, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsRaw,
	}

	return json.Marshal(req)
}

// ParseResponse 解析 JSON-RPC 响应
func ParseResponse(data []byte) (*JSONRPCResponse, error) {
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}
