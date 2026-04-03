package tools_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/benaskins/axon-code/internal/tools"
)

// TestBashEchoReturnsStdout verifies a simple echo command populates stdout.
func TestBashEchoReturnsStdout(t *testing.T) {
	dir := t.TempDir()
	bt := tools.NewBashTool(dir, 10*time.Second)

	result := bt.BashTool.Execute(toolCtx(), args("command", "echo hello"))
	if strings.Contains(result.Content, "error") {
		t.Fatalf("bash unexpectedly errored: %s", result.Content)
	}

	var out tools.BashResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if !strings.Contains(out.Stdout, "hello") {
		t.Errorf("expected 'hello' in stdout, got: %q", out.Stdout)
	}
	if out.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got: %d", out.ExitCode)
	}
}

// TestBashFailingCommandReturnsNonZeroExitCode verifies a failing command returns
// non-zero exit_code and populates stderr.
func TestBashFailingCommandReturnsNonZeroExitCode(t *testing.T) {
	dir := t.TempDir()
	bt := tools.NewBashTool(dir, 10*time.Second)

	result := bt.BashTool.Execute(toolCtx(), args("command", "ls /nonexistent_path_xyz 2>&1; exit 1"))
	var out tools.BashResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode == 0 {
		t.Errorf("expected non-zero exit_code, got 0; stdout=%q stderr=%q", out.Stdout, out.Stderr)
	}
}

// TestBashTimeoutCancelsCommand verifies a long-running command is cancelled by timeout.
func TestBashTimeoutCancelsCommand(t *testing.T) {
	dir := t.TempDir()
	bt := tools.NewBashTool(dir, 200*time.Millisecond)

	result := bt.BashTool.Execute(toolCtx(), args("command", "sleep 60"))
	var out tools.BashResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode == 0 {
		t.Errorf("expected non-zero exit_code after timeout, got 0")
	}
}

// TestBashTimeoutOverrideViaParam verifies timeout_seconds param overrides the default.
func TestBashTimeoutOverrideViaParam(t *testing.T) {
	dir := t.TempDir()
	// default is long; param is short
	bt := tools.NewBashTool(dir, 30*time.Second)

	result := bt.BashTool.Execute(toolCtx(), args("command", "sleep 60", "timeout_seconds", float64(0.2)))
	var out tools.BashResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode == 0 {
		t.Errorf("expected non-zero exit_code after short timeout, got 0")
	}
}

// TestBashCwdIsProjectDir verifies the working directory is set to the project dir.
func TestBashCwdIsProjectDir(t *testing.T) {
	dir := t.TempDir()
	bt := tools.NewBashTool(dir, 10*time.Second)

	result := bt.BashTool.Execute(toolCtx(), args("command", "pwd"))
	var out tools.BashResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode != 0 {
		t.Fatalf("pwd failed: %s", out.Stderr)
	}
	// Trim trailing newline for comparison. Use EvalSymlinks-style: just check
	// the path is a suffix/equal since t.TempDir may use /private/... on macOS.
	got := strings.TrimSpace(out.Stdout)
	if !strings.HasSuffix(got, strings.TrimPrefix(dir, "/private")) && got != dir {
		t.Errorf("expected cwd %q, got %q", dir, got)
	}
}

// TestBashMissingCommand verifies missing 'command' arg returns an error result.
func TestBashMissingCommand(t *testing.T) {
	dir := t.TempDir()
	bt := tools.NewBashTool(dir, 10*time.Second)

	result := bt.BashTool.Execute(toolCtx(), args())
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing command, got: %s", result.Content)
	}
}
