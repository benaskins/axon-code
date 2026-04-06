package tools_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/benaskins/axon-code/internal/tools"
)

func TestGitCmdStatus(t *testing.T) {
	dir := t.TempDir()
	// Init a git repo in the temp dir
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	gc := tools.NewGitCmdTool(dir, 10*time.Second)

	result := gc.GitCmdTool.Execute(toolCtx(), args("args", "status"))
	if strings.Contains(result.Content, "error:") {
		t.Fatalf("git_cmd unexpectedly errored: %s", result.Content)
	}

	var out tools.CmdResult
	if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s (%v)", result.Content, err)
	}
	if out.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got: %d; stderr=%q", out.ExitCode, out.Stderr)
	}
}

func TestGitCmdRejectsDisallowedSubcommand(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGitCmdTool(dir, 10*time.Second)

	result := gc.GitCmdTool.Execute(toolCtx(), args("args", "push origin main"))
	if !strings.Contains(result.Content, "not allowed") {
		t.Errorf("expected 'not allowed' error, got: %s", result.Content)
	}
}

func TestGitCmdRejectsDestructiveCommands(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGitCmdTool(dir, 10*time.Second)

	for _, sub := range []string{"push", "reset", "checkout", "rebase", "merge", "pull"} {
		result := gc.GitCmdTool.Execute(toolCtx(), args("args", sub))
		if !strings.Contains(result.Content, "not allowed") {
			t.Errorf("expected %q to be rejected, got: %s", sub, result.Content)
		}
	}
}

func TestGitCmdMissingArgs(t *testing.T) {
	dir := t.TempDir()
	gc := tools.NewGitCmdTool(dir, 10*time.Second)

	result := gc.GitCmdTool.Execute(toolCtx(), args())
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing args, got: %s", result.Content)
	}
}
