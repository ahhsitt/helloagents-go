package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/easyops/helloagents-go/pkg/tools/builtin"
)

func TestTerminal_WhitelistEnforcement(t *testing.T) {
	terminal := builtin.NewTerminal()

	tests := []struct {
		name        string
		command     string
		shouldAllow bool
	}{
		{"allowed ls", "ls", true},
		{"allowed cat", "cat /etc/hosts", true},
		{"allowed pwd", "pwd", true},
		{"allowed grep", "grep test file.txt", true},
		{"denied rm", "rm file.txt", false},
		{"denied python", "python script.py", false},
		{"denied curl", "curl http://example.com", false},
		{"denied wget", "wget http://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := terminal.Execute(context.Background(), map[string]interface{}{
				"command": tt.command,
			})

			if tt.shouldAllow && err != nil && strings.Contains(err.Error(), "not allowed") {
				t.Errorf("expected command to be allowed, got error: %v", err)
			}
			if !tt.shouldAllow && (err == nil || !strings.Contains(err.Error(), "not allowed")) {
				t.Errorf("expected command to be denied, got: %v", err)
			}
		})
	}
}

func TestTerminal_DangerousPatternDetection(t *testing.T) {
	terminal := builtin.NewTerminal()

	tests := []struct {
		name    string
		command string
		pattern string
	}{
		{"command chaining semicolon", "ls; rm -rf /", ";"},
		{"command chaining and", "ls && rm -rf /", "&&"},
		{"command chaining or", "ls || rm -rf /", "||"},
		{"command substitution backtick", "echo `whoami`", "`"},
		{"command substitution dollar", "echo $(whoami)", "$("},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := terminal.Execute(context.Background(), map[string]interface{}{
				"command": tt.command,
			})

			if err == nil {
				t.Error("expected error for dangerous pattern")
			}
			if !strings.Contains(err.Error(), "dangerous pattern") {
				t.Errorf("expected dangerous pattern error, got: %v", err)
			}
		})
	}
}

func TestTerminal_PipeControl(t *testing.T) {
	// Strict mode - no pipe
	strictTerminal := builtin.NewTerminal()

	_, err := strictTerminal.Execute(context.Background(), map[string]interface{}{
		"command": "ls | wc -l",
	})
	if err == nil || !strings.Contains(err.Error(), "pipe operations not allowed") {
		t.Errorf("strict mode should deny pipe, got: %v", err)
	}

	// Standard mode - allow pipe
	standardTerminal := builtin.NewTerminal(builtin.WithSecurityMode(builtin.ModeStandard))

	_, err = standardTerminal.Execute(context.Background(), map[string]interface{}{
		"command": "ls | wc -l",
	})
	// May fail for other reasons (file not found etc), but should not fail for pipe
	if err != nil && strings.Contains(err.Error(), "pipe operations not allowed") {
		t.Errorf("standard mode should allow pipe, got: %v", err)
	}
}

func TestTerminal_RedirectControl(t *testing.T) {
	// Strict mode - no redirect
	strictTerminal := builtin.NewTerminal()

	_, err := strictTerminal.Execute(context.Background(), map[string]interface{}{
		"command": "echo test > file.txt",
	})
	if err == nil || !strings.Contains(err.Error(), "redirect operations not allowed") {
		t.Errorf("strict mode should deny redirect, got: %v", err)
	}

	// Relaxed mode - allow redirect
	relaxedTerminal := builtin.NewTerminal(builtin.WithSecurityMode(builtin.ModeRelaxed))

	// Should not fail for redirect policy (may fail for other reasons)
	_, err = relaxedTerminal.Execute(context.Background(), map[string]interface{}{
		"command": "echo test > /dev/null",
	})
	if err != nil && strings.Contains(err.Error(), "redirect operations not allowed") {
		t.Errorf("relaxed mode should allow redirect, got: %v", err)
	}
}

func TestTerminal_WorkspaceSandbox(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "terminal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	terminal := builtin.NewTerminal(builtin.WithWorkspace(tmpDir))

	// Test valid path within workspace
	_, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "ls .",
	})
	if err != nil {
		t.Errorf("ls in workspace should work, got: %v", err)
	}

	// Test path traversal detection
	_, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cat ../../etc/passwd",
	})
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("path traversal should be detected, got: %v", err)
	}
}

func TestTerminal_CdCommand(t *testing.T) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "terminal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	terminal := builtin.NewTerminal(builtin.WithWorkspace(tmpDir))

	// Test cd to subdirectory
	output, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd subdir",
	})
	if err != nil {
		t.Errorf("cd to subdir should work, got: %v", err)
	}
	if !strings.Contains(output, "Changed to") {
		t.Errorf("expected 'Changed to' in output, got: %s", output)
	}

	// Verify current directory changed
	if terminal.CurrentDir() != subDir {
		t.Errorf("expected currentDir to be %s, got: %s", subDir, terminal.CurrentDir())
	}

	// Test cd back with ..
	output, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd ..",
	})
	if err != nil {
		t.Errorf("cd .. should work, got: %v", err)
	}
	if terminal.CurrentDir() != tmpDir {
		t.Errorf("expected currentDir to be %s, got: %s", tmpDir, terminal.CurrentDir())
	}

	// Test cd escape attempt
	_, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd ../..",
	})
	if err == nil || !strings.Contains(err.Error(), "cannot navigate outside workspace") {
		t.Errorf("cd escape should be blocked, got: %v", err)
	}

	// Test cd without args
	output, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd",
	})
	if err != nil {
		t.Errorf("cd without args should work, got: %v", err)
	}
	if !strings.Contains(output, "Current directory") {
		t.Errorf("expected 'Current directory' in output, got: %s", output)
	}
}

func TestTerminal_CdDisabled(t *testing.T) {
	terminal := builtin.NewTerminal(builtin.WithAllowCd(false))

	_, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd /tmp",
	})
	if err == nil || !strings.Contains(err.Error(), "cd command is disabled") {
		t.Errorf("cd should be disabled, got: %v", err)
	}
}

func TestTerminal_Timeout(t *testing.T) {
	terminal := builtin.NewTerminal(
		builtin.WithTerminalTimeout(100*time.Millisecond),
		builtin.WithSecurityMode(builtin.ModeStandard), // allow sleep
		builtin.WithAllowedCommands([]string{"sleep"}),
	)

	_, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "sleep 10",
	})

	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestTerminal_OutputSizeLimit(t *testing.T) {
	terminal := builtin.NewTerminal(
		builtin.WithMaxOutputSize(100), // Very small limit
	)

	// Generate output larger than limit
	output, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "ls -la /",
	})

	// Command may or may not error, but output should be truncated
	if err == nil && len(output) > 0 {
		if !strings.Contains(output, "[WARNING]") && len(output) > 200 {
			t.Errorf("expected output to be truncated or have warning, got %d bytes", len(output))
		}
	}
}

func TestTerminal_SecurityModes(t *testing.T) {
	tests := []struct {
		name     string
		mode     builtin.SecurityMode
		command  string
		allowed  bool
	}{
		{"strict allows ls", builtin.ModeStrict, "ls", true},
		{"strict denies rm", builtin.ModeStrict, "rm file", false},
		{"relaxed allows rm", builtin.ModeRelaxed, "rm file", true},
		{"custom allows all when no whitelist", builtin.ModeCustom, "ls", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminal := builtin.NewTerminal(builtin.WithSecurityMode(tt.mode))

			_, err := terminal.Execute(context.Background(), map[string]interface{}{
				"command": tt.command,
			})

			isAllowed := err == nil || !strings.Contains(err.Error(), "not allowed")

			if tt.allowed && !isAllowed {
				t.Errorf("expected command to be allowed, got: %v", err)
			}
			if !tt.allowed && isAllowed {
				t.Errorf("expected command to be denied")
			}
		})
	}
}

func TestTerminal_CustomWhitelist(t *testing.T) {
	terminal := builtin.NewTerminal(
		builtin.WithAllowedCommands([]string{"custom-cmd", "another-cmd"}),
	)

	// Standard commands should be denied
	_, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "ls",
	})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("ls should be denied with custom whitelist, got: %v", err)
	}
}

func TestTerminal_ResetDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "terminal-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	terminal := builtin.NewTerminal(builtin.WithWorkspace(tmpDir))

	// Change directory
	_, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "cd subdir",
	})
	if err != nil {
		t.Fatalf("cd failed: %v", err)
	}

	// Reset
	terminal.ResetDir()

	if terminal.CurrentDir() != tmpDir {
		t.Errorf("ResetDir should reset to workspace, got: %s", terminal.CurrentDir())
	}
}

func TestTerminal_Validate(t *testing.T) {
	terminal := builtin.NewTerminal()

	// Missing command
	err := terminal.Validate(map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing command")
	}

	// Wrong type
	err = terminal.Validate(map[string]interface{}{
		"command": 123,
	})
	if err == nil {
		t.Error("expected error for wrong type")
	}

	// Valid
	err = terminal.Validate(map[string]interface{}{
		"command": "ls",
	})
	if err != nil {
		t.Errorf("expected no error for valid input, got: %v", err)
	}
}

func TestTerminal_EmptyCommand(t *testing.T) {
	terminal := builtin.NewTerminal()

	_, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "",
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected empty command error, got: %v", err)
	}

	_, err = terminal.Execute(context.Background(), map[string]interface{}{
		"command": "   ",
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected empty command error for whitespace, got: %v", err)
	}
}

func TestTerminal_PathCommand(t *testing.T) {
	terminal := builtin.NewTerminal()

	// Full path to command should work (base name is checked)
	_, err := terminal.Execute(context.Background(), map[string]interface{}{
		"command": "/bin/ls",
	})
	if err != nil && strings.Contains(err.Error(), "not allowed") {
		t.Errorf("/bin/ls should be allowed (base name is ls), got: %v", err)
	}
}
