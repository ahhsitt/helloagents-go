package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/easyops/helloagents-go/pkg/tools"
)

// Terminal 终端工具
//
// 允许执行 shell 命令。出于安全考虑，默认禁用危险命令。
type Terminal struct {
	// allowedCommands 允许执行的命令白名单（为空则允许所有）
	allowedCommands []string
	// blockedCommands 禁止执行的命令黑名单
	blockedCommands []string
	// workDir 工作目录
	workDir string
	// timeout 命令执行超时时间
	timeout time.Duration
}

// TerminalOption Terminal 配置选项
type TerminalOption func(*Terminal)

// NewTerminal 创建终端工具
func NewTerminal(opts ...TerminalOption) *Terminal {
	t := &Terminal{
		blockedCommands: []string{
			"rm -rf /",
			"rm -rf /*",
			":(){ :|:& };:",
			"dd if=/dev/zero",
			"mkfs",
			"shutdown",
			"reboot",
			"halt",
			"poweroff",
		},
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithAllowedCommands 设置允许的命令白名单
func WithAllowedCommands(commands []string) TerminalOption {
	return func(t *Terminal) {
		t.allowedCommands = commands
	}
}

// WithBlockedCommands 追加禁止的命令
func WithBlockedCommands(commands []string) TerminalOption {
	return func(t *Terminal) {
		t.blockedCommands = append(t.blockedCommands, commands...)
	}
}

// WithWorkDir 设置工作目录
func WithWorkDir(dir string) TerminalOption {
	return func(t *Terminal) {
		t.workDir = dir
	}
}

// WithTerminalTimeout 设置命令执行超时
func WithTerminalTimeout(d time.Duration) TerminalOption {
	return func(t *Terminal) {
		t.timeout = d
	}
}

// Name 返回工具名称
func (t *Terminal) Name() string {
	return "terminal"
}

// Description 返回工具描述
func (t *Terminal) Description() string {
	return "Execute shell commands. Use this tool to run system commands and scripts. Some dangerous commands are blocked for security."
}

// Parameters 返回参数 Schema
func (t *Terminal) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"command": {
				Type:        "string",
				Description: "The shell command to execute",
			},
		},
		Required: []string{"command"},
	}
}

// Execute 执行命令
func (t *Terminal) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	cmdRaw, ok := args["command"]
	if !ok {
		return "", fmt.Errorf("missing required parameter: command")
	}

	command, ok := cmdRaw.(string)
	if !ok {
		return "", fmt.Errorf("command must be a string")
	}

	// 检查命令是否允许
	if err := t.checkCommand(command); err != nil {
		return "", err
	}

	// 应用超时
	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if t.workDir != "" {
		cmd.Dir = t.workDir
	}

	// 执行命令
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// 组合输出
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR: " + stderr.String()
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %v", t.timeout)
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// checkCommand 检查命令是否允许执行
func (t *Terminal) checkCommand(command string) error {
	cmdLower := strings.ToLower(command)

	// 检查白名单
	if len(t.allowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range t.allowedCommands {
			if strings.HasPrefix(cmdLower, strings.ToLower(allowedCmd)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command not in allowed list: %s", command)
		}
	}

	// 检查黑名单
	for _, blockedCmd := range t.blockedCommands {
		if strings.Contains(cmdLower, strings.ToLower(blockedCmd)) {
			return fmt.Errorf("command is blocked for security: %s", command)
		}
	}

	return nil
}

// Validate 验证参数
func (t *Terminal) Validate(args map[string]interface{}) error {
	cmd, ok := args["command"]
	if !ok {
		return fmt.Errorf("missing required parameter: command")
	}
	if _, ok := cmd.(string); !ok {
		return fmt.Errorf("command must be a string")
	}
	return nil
}

// compile-time interface check
var _ tools.Tool = (*Terminal)(nil)
var _ tools.ToolWithValidation = (*Terminal)(nil)
