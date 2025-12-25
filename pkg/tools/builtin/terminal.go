package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/easyops/helloagents-go/pkg/tools"
)

// SecurityMode 定义终端工具的安全模式预设
type SecurityMode int

const (
	// ModeStrict 严格模式（默认）
	// 只允许只读命令，不允许管道和重定向
	ModeStrict SecurityMode = iota

	// ModeStandard 标准模式
	// 允许只读命令和安全的管道操作
	ModeStandard

	// ModeRelaxed 宽松模式
	// 允许读写命令，但仍禁止危险系统命令
	ModeRelaxed

	// ModeCustom 自定义模式
	// 完全由用户配置白名单和规则
	ModeCustom
)

// CommandCategory 定义命令的安全分类
type CommandCategory int

const (
	// CategoryReadOnly 只读命令（最安全）
	CategoryReadOnly CommandCategory = iota

	// CategoryReadWrite 读写命令（需要授权）
	CategoryReadWrite

	// CategoryExecute 执行命令（高风险）
	CategoryExecute
)

// 默认配置常量
const (
	DefaultMaxOutputSize = 10 * 1024 * 1024 // 10MB
	DefaultTimeout       = 30 * time.Second
)

// 只读命令白名单（CategoryReadOnly）
var readOnlyCommands = map[string]bool{
	// 文件列表与信息
	"ls": true, "dir": true, "tree": true,
	// 文件内容查看
	"cat": true, "head": true, "tail": true, "less": true, "more": true,
	// 文件搜索
	"find": true, "grep": true, "egrep": true, "fgrep": true, "rg": true,
	// 文本处理
	"wc": true, "sort": true, "uniq": true, "cut": true, "awk": true, "sed": true,
	// 目录操作
	"pwd": true, "cd": true,
	// 文件信息
	"file": true, "stat": true, "du": true, "df": true,
	// 其他
	"echo": true, "which": true, "whereis": true, "whoami": true,
	"date": true, "env": true, "printenv": true,
}

// 读写命令白名单（CategoryReadWrite）
var readWriteCommands = map[string]bool{
	"cp": true, "mv": true, "mkdir": true, "touch": true,
	"rm": true, "rmdir": true,
	"chmod": true, "chown": true,
	"ln": true,
}

// 执行命令白名单（CategoryExecute）- 预留给未来扩展
// nolint:unused
var executeCommands = map[string]bool{
	"python": true, "python3": true,
	"node": true, "npm": true, "npx": true,
	"go":   true,
	"bash": true, "sh": true, "zsh": true,
	"make": true,
	"git":  true,
}

// 危险模式列表（用于命令注入检测）
var dangerousPatterns = []string{
	";",   // 命令分隔
	"&&",  // 命令链接
	"||",  // 条件执行
	"`",   // 命令替换
	"$(",  // 命令替换
	"$((", // 算术扩展
}

// 需要单独配置的模式（管道和重定向）
var pipePattern = "|"
var redirectPatterns = []string{">", "<", ">>", "<<"}

// Terminal 终端工具
//
// 为 Agent 提供安全的命令行执行能力。
//
// 安全特性：
//   - 命令白名单（默认只允许只读命令）
//   - 工作目录沙箱（限制在指定目录内操作）
//   - 路径遍历防护（防止 ../ 逃逸）
//   - 命令注入检测（检测危险 shell 模式）
//   - 输出大小限制（防止内存溢出）
//   - 超时控制
//
// 使用示例：
//
//	// 严格模式（默认）- 只允许只读命令
//	terminal := NewTerminal(WithWorkspace("./project"))
//
//	// 标准模式 - 允许管道操作
//	terminal := NewTerminal(
//	    WithSecurityMode(ModeStandard),
//	    WithWorkspace("./project"),
//	)
//
//	// 宽松模式 - 允许读写操作
//	terminal := NewTerminal(
//	    WithSecurityMode(ModeRelaxed),
//	    WithWorkspace("./project"),
//	)
type Terminal struct {
	// securityMode 安全模式
	securityMode SecurityMode
	// allowedCommands 允许执行的命令白名单
	allowedCommands map[string]bool
	// workspace 沙箱工作目录（为空则不限制）
	workspace string
	// currentDir 当前工作目录
	currentDir string
	// timeout 命令执行超时时间
	timeout time.Duration
	// maxOutputSize 最大输出大小
	maxOutputSize int64
	// allowPipe 是否允许管道操作
	allowPipe bool
	// allowRedirect 是否允许重定向
	allowRedirect bool
	// allowCd 是否允许 cd 命令
	allowCd bool
	// logger 日志记录器
	logger *slog.Logger
}

// TerminalOption Terminal 配置选项
type TerminalOption func(*Terminal)

// NewTerminal 创建终端工具
//
// 默认使用严格模式（ModeStrict），只允许只读命令。
// 可通过 Option 函数自定义安全配置。
func NewTerminal(opts ...TerminalOption) *Terminal {
	t := &Terminal{
		securityMode:    ModeStrict,
		allowedCommands: make(map[string]bool),
		timeout:         DefaultTimeout,
		maxOutputSize:   DefaultMaxOutputSize,
		allowPipe:       false,
		allowRedirect:   false,
		allowCd:         true,
		logger:          slog.Default(),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(t)
	}

	// 根据安全模式设置默认白名单
	t.applySecurityMode()

	// 初始化当前目录
	if t.workspace != "" {
		t.currentDir = t.workspace
	}

	return t
}

// applySecurityMode 根据安全模式应用默认配置
func (t *Terminal) applySecurityMode() {
	// 如果用户已设置自定义白名单，保持不变
	if len(t.allowedCommands) > 0 {
		return
	}

	switch t.securityMode {
	case ModeStrict:
		// 只允许只读命令
		for cmd := range readOnlyCommands {
			t.allowedCommands[cmd] = true
		}
		t.allowPipe = false
		t.allowRedirect = false

	case ModeStandard:
		// 只读命令 + 管道
		for cmd := range readOnlyCommands {
			t.allowedCommands[cmd] = true
		}
		t.allowPipe = true
		t.allowRedirect = false

	case ModeRelaxed:
		// 只读 + 读写命令
		for cmd := range readOnlyCommands {
			t.allowedCommands[cmd] = true
		}
		for cmd := range readWriteCommands {
			t.allowedCommands[cmd] = true
		}
		t.allowPipe = true
		t.allowRedirect = true

	case ModeCustom:
		// 用户完全自定义，不设置默认白名单
	}
}

// WithSecurityMode 设置安全模式
//
// 可用模式：
//   - ModeStrict: 严格模式（默认），只允许只读命令
//   - ModeStandard: 标准模式，允许只读命令和管道
//   - ModeRelaxed: 宽松模式，允许读写命令
//   - ModeCustom: 自定义模式，完全由用户配置
func WithSecurityMode(mode SecurityMode) TerminalOption {
	return func(t *Terminal) {
		t.securityMode = mode
	}
}

// WithWorkspace 设置工作目录沙箱
//
// 设置后，所有命令只能在该目录及其子目录内执行。
// 路径遍历尝试（如 ../）将被拒绝。
func WithWorkspace(path string) TerminalOption {
	return func(t *Terminal) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			t.workspace = path
		} else {
			// 解析符号链接，确保一致性比较
			realPath, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				t.workspace = absPath
			} else {
				t.workspace = realPath
			}
		}
		t.currentDir = t.workspace
	}
}

// WithAllowedCommands 设置允许的命令白名单
//
// 设置后将覆盖安全模式的默认白名单。
func WithAllowedCommands(commands []string) TerminalOption {
	return func(t *Terminal) {
		t.allowedCommands = make(map[string]bool)
		for _, cmd := range commands {
			t.allowedCommands[cmd] = true
		}
	}
}

// WithAllowPipe 设置是否允许管道操作
func WithAllowPipe(allow bool) TerminalOption {
	return func(t *Terminal) {
		t.allowPipe = allow
	}
}

// WithAllowRedirect 设置是否允许重定向操作
func WithAllowRedirect(allow bool) TerminalOption {
	return func(t *Terminal) {
		t.allowRedirect = allow
	}
}

// WithMaxOutputSize 设置最大输出大小（字节）
func WithMaxOutputSize(size int64) TerminalOption {
	return func(t *Terminal) {
		t.maxOutputSize = size
	}
}

// WithAllowCd 设置是否允许 cd 命令
func WithAllowCd(allow bool) TerminalOption {
	return func(t *Terminal) {
		t.allowCd = allow
	}
}

// WithTerminalTimeout 设置命令执行超时
func WithTerminalTimeout(d time.Duration) TerminalOption {
	return func(t *Terminal) {
		t.timeout = d
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *slog.Logger) TerminalOption {
	return func(t *Terminal) {
		t.logger = logger
	}
}

// Name 返回工具名称
func (t *Terminal) Name() string {
	return "terminal"
}

// Description 返回工具描述
func (t *Terminal) Description() string {
	mode := "strict"
	switch t.securityMode {
	case ModeStandard:
		mode = "standard"
	case ModeRelaxed:
		mode = "relaxed"
	case ModeCustom:
		mode = "custom"
	}
	return fmt.Sprintf("Execute shell commands in %s security mode. Allowed commands are whitelisted for safety.", mode)
}

// Parameters 返回参数 Schema
func (t *Terminal) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"command": {
				Type:        "string",
				Description: "The shell command to execute (must be in the allowed whitelist)",
			},
		},
		Required: []string{"command"},
	}
}

// Execute 执行命令
func (t *Terminal) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	startTime := time.Now()

	cmdRaw, ok := args["command"]
	if !ok {
		return "", fmt.Errorf("missing required parameter: command")
	}

	command, ok := cmdRaw.(string)
	if !ok {
		return "", fmt.Errorf("command must be a string")
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// 解析基础命令
	baseCmd := t.parseBaseCommand(command)

	// 特殊处理 cd 命令
	if baseCmd == "cd" {
		return t.handleCd(command)
	}

	// 检查命令是否允许
	if err := t.checkCommand(command, baseCmd); err != nil {
		t.logSecurityEvent(command, err.Error())
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

	// 设置工作目录
	if t.currentDir != "" {
		cmd.Dir = t.currentDir
	}

	// 执行命令并限制输出
	output, err := t.executeWithLimits(ctx, cmd)

	// 记录执行日志
	duration := time.Since(startTime)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	t.logExecution(command, t.currentDir, duration, exitCode, len(output))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %v", t.timeout)
		}
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// parseBaseCommand 解析命令字符串，提取基础命令
func (t *Terminal) parseBaseCommand(command string) string {
	// 去除前导空格
	command = strings.TrimSpace(command)

	// 提取第一个单词作为基础命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	baseCmd := parts[0]

	// 处理路径形式的命令（如 /usr/bin/ls）
	baseCmd = filepath.Base(baseCmd)

	return baseCmd
}

// checkCommand 检查命令是否允许执行
func (t *Terminal) checkCommand(command, baseCmd string) error {
	// 先检查危险模式（优先级最高）
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	// 检查白名单（ModeCustom 且无白名单时跳过）
	if len(t.allowedCommands) > 0 && !t.allowedCommands[baseCmd] {
		allowedList := make([]string, 0, len(t.allowedCommands))
		for cmd := range t.allowedCommands {
			allowedList = append(allowedList, cmd)
		}
		return fmt.Errorf("command not allowed: %s (allowed: %s)", baseCmd, strings.Join(allowedList, ", "))
	}

	// 检查管道
	if !t.allowPipe && strings.Contains(command, pipePattern) {
		return fmt.Errorf("pipe operations not allowed in current security mode")
	}

	// 检查重定向
	if !t.allowRedirect {
		for _, pattern := range redirectPatterns {
			if strings.Contains(command, pattern) {
				return fmt.Errorf("redirect operations not allowed in current security mode")
			}
		}
	}

	// 检查路径安全（如果设置了 workspace）
	if t.workspace != "" {
		if err := t.checkPathSecurity(command); err != nil {
			return err
		}
	}

	return nil
}

// checkPathSecurity 检查命令中的路径是否在沙箱内
func (t *Terminal) checkPathSecurity(command string) error {
	parts := strings.Fields(command)

	for _, part := range parts[1:] { // 跳过命令本身
		// 跳过选项参数
		if strings.HasPrefix(part, "-") {
			continue
		}

		// 检查是否包含路径遍历
		if strings.Contains(part, "..") {
			// 解析完整路径
			var fullPath string
			if filepath.IsAbs(part) {
				fullPath = filepath.Clean(part)
			} else {
				fullPath = filepath.Clean(filepath.Join(t.currentDir, part))
			}

			// 检查是否在 workspace 内
			if !strings.HasPrefix(fullPath, t.workspace) {
				return fmt.Errorf("path traversal detected: access denied outside workspace")
			}
		}
	}

	return nil
}

// handleCd 处理 cd 命令
func (t *Terminal) handleCd(command string) (string, error) {
	if !t.allowCd {
		return "", fmt.Errorf("cd command is disabled")
	}

	parts := strings.Fields(command)

	// cd 无参数，返回当前目录
	if len(parts) < 2 {
		return fmt.Sprintf("Current directory: %s", t.currentDir), nil
	}

	targetDir := parts[1]
	var newDir string

	// 处理特殊路径
	switch targetDir {
	case "~":
		if t.workspace != "" {
			newDir = t.workspace
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("cannot get home directory: %w", err)
			}
			newDir = home
		}
	case "-":
		return "", fmt.Errorf("cd - is not supported")
	default:
		if filepath.IsAbs(targetDir) {
			newDir = filepath.Clean(targetDir)
		} else {
			newDir = filepath.Clean(filepath.Join(t.currentDir, targetDir))
		}
	}

	// 解析符号链接
	realPath, err := filepath.EvalSymlinks(newDir)
	if err != nil {
		// 目录可能不存在，使用原路径检查
		realPath = newDir
	}

	// 检查是否在 workspace 内
	if t.workspace != "" && !strings.HasPrefix(realPath, t.workspace) {
		return "", fmt.Errorf("cannot navigate outside workspace: %s", t.workspace)
	}

	// 检查目录是否存在
	info, err := os.Stat(newDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory does not exist: %s", newDir)
		}
		return "", fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", newDir)
	}

	// 更新当前目录
	t.currentDir = newDir

	return fmt.Sprintf("Changed to: %s", t.currentDir), nil
}

// executeWithLimits 执行命令并限制输出大小
func (t *Terminal) executeWithLimits(ctx context.Context, cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer

	// 使用 LimitedWriter 限制输出
	stdoutLimited := &limitedWriter{w: &stdout, limit: t.maxOutputSize}
	stderrLimited := &limitedWriter{w: &stderr, limit: t.maxOutputSize}

	cmd.Stdout = stdoutLimited
	cmd.Stderr = stderrLimited

	err := cmd.Run()

	// 组合输出
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "[STDERR]\n" + stderr.String()
	}

	// 添加截断警告
	if stdoutLimited.truncated || stderrLimited.truncated {
		output += fmt.Sprintf("\n\n[WARNING] Output truncated (exceeded %d bytes limit)", t.maxOutputSize)
	}

	if output == "" && err == nil {
		output = "(command completed with no output)"
	}

	return strings.TrimSpace(output), err
}

// limitedWriter 限制写入大小的 Writer
type limitedWriter struct {
	w         io.Writer
	limit     int64
	written   int64
	truncated bool
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.written >= lw.limit {
		lw.truncated = true
		return len(p), nil // 静默丢弃
	}

	remaining := lw.limit - lw.written
	if int64(len(p)) > remaining {
		p = p[:remaining]
		lw.truncated = true
	}

	n, err := lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}

// logExecution 记录命令执行日志
func (t *Terminal) logExecution(command, cwd string, duration time.Duration, exitCode, outputSize int) {
	t.logger.Info("terminal command executed",
		slog.String("command", command),
		slog.String("cwd", cwd),
		slog.Duration("duration", duration),
		slog.Int("exit_code", exitCode),
		slog.Int("output_size", outputSize),
	)
}

// logSecurityEvent 记录安全事件
func (t *Terminal) logSecurityEvent(command, reason string) {
	t.logger.Warn("terminal command rejected",
		slog.String("command", command),
		slog.String("reason", reason),
	)
}

// CurrentDir 返回当前工作目录
func (t *Terminal) CurrentDir() string {
	return t.currentDir
}

// ResetDir 重置到 workspace 根目录
func (t *Terminal) ResetDir() {
	if t.workspace != "" {
		t.currentDir = t.workspace
	}
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
