package tools_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/benaskins/axon-code/internal/tools"
)

func TestGoCmdVersion(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGoCmdTool(dir, 10*time.Second, nil)

	result := gc.GoCmdTool.Execute(toolCtx(), args("args", "version"))
	if strings.Contains(result.Content, "error") {
		t.Fatalf("go_cmd unexpectedly errored: %s", result.Content)
	}

	var out tools.CmdResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if !strings.Contains(out.Stdout, "go version") {
		t.Errorf("expected 'go version' in stdout, got: %q", out.Stdout)
	}
}

func TestGoCmdRejectsDisallowedSubcommand(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGoCmdTool(dir, 10*time.Second, nil)

	result := gc.GoCmdTool.Execute(toolCtx(), args("args", "install-something-bad"))
	if !strings.Contains(result.Content, "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %s", result.Content)
	}
}

func TestGoCmdRejectsEmptyArgs(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGoCmdTool(dir, 10*time.Second, nil)

	result := gc.GoCmdTool.Execute(toolCtx(), args("args", ""))
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for empty args, got: %s", result.Content)
	}
}

func TestGoCmdMissingArgs(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGoCmdTool(dir, 10*time.Second, nil)

	result := gc.GoCmdTool.Execute(toolCtx(), args())
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing args, got: %s", result.Content)
	}
}

func TestGoCmdCwdIsProjectDir(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGoCmdTool(dir, 10*time.Second, nil)

	result := gc.GoCmdTool.Execute(toolCtx(), args("args", "env GOPATH"))
	if strings.Contains(result.Content, "error:") {
		t.Fatalf("go_cmd unexpectedly errored: %s", result.Content)
	}

	var out tools.CmdResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got: %d; stderr=%q", out.ExitCode, out.Stderr)
	}
}
