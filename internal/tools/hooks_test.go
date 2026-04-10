package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tool "github.com/benaskins/axon-tool"
)

func hooksSkipIfNoGopls(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not on PATH")
	}
}

func hooksSetupGoModule(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	gomod := "module testmod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// --- Unit tests: HookRegistry ---

func TestRunFileHooks_PatternMatching(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "*.go", Run: func(absPath, projectDir string) string { return "go-hook" }},
		},
	}
	out := hr.RunFileHooks("main.go", "/tmp/main.go", "/tmp")
	if out != "go-hook" {
		t.Errorf("expected 'go-hook', got %q", out)
	}
	out = hr.RunFileHooks("internal/parse/parse.go", "/tmp/internal/parse/parse.go", "/tmp")
	if out != "go-hook" {
		t.Errorf("expected 'go-hook' for nested path, got %q", out)
	}
	out = hr.RunFileHooks("readme.md", "/tmp/readme.md", "/tmp")
	if out != "" {
		t.Errorf("expected empty for non-matching file, got %q", out)
	}
}

func TestRunFileHooks_SilentOnEmpty(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "*.go", Run: func(absPath, projectDir string) string { return "" }},
		},
	}
	out := hr.RunFileHooks("main.go", "/tmp/main.go", "/tmp")
	if out != "" {
		t.Errorf("expected empty for silent hook, got %q", out)
	}
}

func TestRunFileHooks_MultipleHooksAggregated(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "*.go", Run: func(absPath, projectDir string) string { return "hook-a" }},
			{Pattern: "*.go", Run: func(absPath, projectDir string) string { return "hook-b" }},
		},
	}
	out := hr.RunFileHooks("main.go", "/tmp/main.go", "/tmp")
	if out != "hook-a\nhook-b" {
		t.Errorf("expected aggregated output, got %q", out)
	}
}

func TestRunCmdHooks_MatchesSubcommandAndExitCode(t *testing.T) {
	hr := &HookRegistry{
		CmdHooks: []CmdHook{
			{
				Name:  "diag",
				Match: func(sub string, exit int) bool { return exit != 0 && sub == "build" },
				Run:   func(projectDir string) string { return "build-diag" },
			},
		},
	}
	// Should match
	m := hr.RunCmdHooks("build", 1, "/tmp")
	if m == nil || m["diag"] != "build-diag" {
		t.Errorf("expected diag hook output, got %v", m)
	}
	// Wrong subcommand
	m = hr.RunCmdHooks("version", 1, "/tmp")
	if m != nil {
		t.Errorf("expected nil for non-matching subcommand, got %v", m)
	}
	// Exit 0
	m = hr.RunCmdHooks("build", 0, "/tmp")
	if m != nil {
		t.Errorf("expected nil for exit 0, got %v", m)
	}
}

func TestHookPanicRecovery_FileHook(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "*.go", Run: func(absPath, projectDir string) string { panic("boom") }},
		},
	}
	out := hr.RunFileHooks("main.go", "/tmp/main.go", "/tmp")
	if out == "" {
		t.Error("expected panic recovery message, got empty")
	}
}

func TestHookPanicRecovery_CmdHook(t *testing.T) {
	hr := &HookRegistry{
		CmdHooks: []CmdHook{
			{
				Name:  "bad",
				Match: func(string, int) bool { return true },
				Run:   func(string) string { panic("boom") },
			},
		},
	}
	m := hr.RunCmdHooks("build", 1, "/tmp")
	if m == nil || m["bad"] == "" {
		t.Error("expected panic recovery message in hooks map")
	}
}

func TestNilRegistrySafe(t *testing.T) {
	var hr *HookRegistry
	out := hr.RunFileHooks("main.go", "/tmp/main.go", "/tmp")
	if out != "" {
		t.Errorf("expected empty from nil registry, got %q", out)
	}
	m := hr.RunCmdHooks("build", 1, "/tmp")
	if m != nil {
		t.Errorf("expected nil from nil registry, got %v", m)
	}
}

func TestFileHook_GoModPattern(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "go.mod", Run: func(absPath, projectDir string) string { return "tidy" }},
		},
	}
	out := hr.RunFileHooks("go.mod", "/tmp/go.mod", "/tmp")
	if out != "tidy" {
		t.Errorf("expected 'tidy', got %q", out)
	}
	// Shouldn't match go.sum
	out = hr.RunFileHooks("go.sum", "/tmp/go.sum", "/tmp")
	if out != "" {
		t.Errorf("expected empty for go.sum, got %q", out)
	}
}

// --- Integration tests: default hooks with real commands ---

func TestDefaultGofmtHook_FormatsFile(t *testing.T) {
	dir := t.TempDir()
	// Write a badly formatted Go file
	badGo := "package main\n\nfunc main(){\nfmt.Println( \"hello\" )\n}\n"
	abs := filepath.Join(dir, "main.go")
	if err := os.WriteFile(abs, []byte(badGo), 0644); err != nil {
		t.Fatal(err)
	}

	hr := NewDefaultHookRegistry(dir)
	out := hr.RunFileHooks("main.go", abs, dir)
	// gofmt should succeed silently
	if out != "" {
		t.Errorf("expected silent gofmt on valid Go, got %q", out)
	}
	// Verify the file was actually formatted
	formatted, _ := os.ReadFile(abs)
	if strings.Contains(string(formatted), "Println( ") {
		t.Error("file was not reformatted by gofmt")
	}
}

func TestDefaultGofmtHook_ReportsSyntaxError(t *testing.T) {
	dir := t.TempDir()
	badGo := "package main\n\nfunc main() {\n\tif true {\n}\n" // missing closing brace
	abs := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(abs, []byte(badGo), 0644); err != nil {
		t.Fatal(err)
	}

	hr := NewDefaultHookRegistry(dir)
	out := hr.RunFileHooks("bad.go", abs, dir)
	if out == "" {
		t.Error("expected gofmt error for syntax error, got empty")
	}
}

func TestDefaultGoModTidyHook_Silent(t *testing.T) {
	dir := t.TempDir()
	gomod := "module testmod\n\ngo 1.21\n"
	abs := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(abs, []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}
	// Need a .go file for go mod tidy to be happy
	mainGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	hr := NewDefaultHookRegistry(dir)
	out := hr.RunFileHooks("go.mod", abs, dir)
	if out != "" {
		t.Errorf("expected silent go mod tidy, got %q", out)
	}
}

// --- Integration tests: hooks through tool execution ---

func TestWriteFile_RunsGofmtHook(t *testing.T) {
	dir := t.TempDir()
	hooks := NewDefaultHookRegistry(dir)
	fs := NewFSTools(dir, NewManifest(), hooks)

	// Write invalid Go (bad syntax) -- gofmt should report error
	badGo := "package main\n\nfunc main() {\n\tif true {\n}\n"
	result := fs.WriteTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{"path": "bad.go", "content": badGo},
	)
	if !strings.Contains(result.Content, "gofmt") {
		t.Errorf("expected gofmt error in result, got: %s", result.Content)
	}
}

func TestWriteFile_GofmtSilentOnValid(t *testing.T) {
	dir := t.TempDir()
	hooks := NewDefaultHookRegistry(dir)
	fs := NewFSTools(dir, NewManifest(), hooks)

	goodGo := "package main\n\nfunc main() {}\n"
	result := fs.WriteTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{"path": "good.go", "content": goodGo},
	)
	// Should just have "wrote N bytes" with no hook output
	if strings.Contains(result.Content, "gofmt") {
		t.Errorf("expected no gofmt output for valid Go, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "wrote") {
		t.Errorf("expected 'wrote' in result, got: %s", result.Content)
	}
}

func TestWriteFile_NonGoFileSkipsHook(t *testing.T) {
	dir := t.TempDir()
	hooks := NewDefaultHookRegistry(dir)
	fs := NewFSTools(dir, NewManifest(), hooks)

	result := fs.WriteTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{"path": "readme.md", "content": "# Hello"},
	)
	if strings.Contains(result.Content, "gofmt") {
		t.Errorf("expected no gofmt for .md file, got: %s", result.Content)
	}
}

func TestEditFile_RunsGofmtHook(t *testing.T) {
	dir := t.TempDir()
	hooks := NewDefaultHookRegistry(dir)
	fs := NewFSTools(dir, NewManifest(), hooks)

	// First write a valid file
	validGo := "package main\n\nfunc main() { println(\"hello\") }\n"
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(validGo), 0644)

	// Edit it to be invalid
	result := fs.EditTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{
			"path":       "main.go",
			"old_string": "println(\"hello\")",
			"new_string": "if true {",
		},
	)
	if !strings.Contains(result.Content, "gofmt") {
		t.Errorf("expected gofmt error after invalid edit, got: %s", result.Content)
	}
}

func TestGoCmdFailure_IncludesHooks(t *testing.T) {
	hooksSkipIfNoGopls(t)
	dir := t.TempDir()
	hooksSetupGoModule(t, dir, map[string]string{
		"main.go": "package main\n\nfunc main() {\n\tvar x int = \"hello\"\n\t_ = x\n}\n",
	})

	hooks := NewDefaultHookRegistry(dir)
	gc := NewGoCmdTool(dir, 30*time.Second, hooks)

	result := gc.GoCmdTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{"args": "build ./..."},
	)

	// Parse the CmdResult JSON
	var cr CmdResult
	if err := json.Unmarshal([]byte(result.Content), &cr); err != nil {
		t.Fatalf("failed to parse CmdResult: %v", err)
	}
	if cr.ExitCode == 0 {
		t.Fatal("expected non-zero exit code for type error")
	}
	if cr.Hooks == nil || cr.Hooks["diagnostics"] == "" {
		t.Errorf("expected diagnostics hook output, got hooks=%v", cr.Hooks)
	}
}

func TestGoCmdSuccess_NoHooks(t *testing.T) {
	dir := t.TempDir()
	hooks := NewDefaultHookRegistry(dir)
	gc := NewGoCmdTool(dir, 10*time.Second, hooks)

	result := gc.GoCmdTool.Execute(
		&tool.ToolContext{Ctx: context.Background()},
		map[string]any{"args": "version"},
	)

	var cr CmdResult
	if err := json.Unmarshal([]byte(result.Content), &cr); err != nil {
		t.Fatalf("failed to parse CmdResult: %v", err)
	}
	if cr.Hooks != nil {
		t.Errorf("expected no hooks on success, got %v", cr.Hooks)
	}
}

func TestCmdResult_HooksOmitEmpty(t *testing.T) {
	cr := CmdResult{Stdout: "ok", ExitCode: 0}
	data, _ := json.Marshal(cr)
	if strings.Contains(string(data), "hooks") {
		t.Errorf("expected hooks omitted from JSON when nil, got: %s", data)
	}
}

// Ensure filepath.Base is used for matching (not full path)
func TestFileHook_MatchesBaseName(t *testing.T) {
	hr := &HookRegistry{
		FileHooks: []FileHook{
			{Pattern: "*.go", Run: func(absPath, projectDir string) string {
				// Verify absPath is passed correctly
				if filepath.Base(absPath) == "parse.go" {
					return "ok"
				}
				return "wrong-path"
			}},
		},
	}
	out := hr.RunFileHooks("internal/parse/parse.go", "/proj/internal/parse/parse.go", "/proj")
	if out != "ok" {
		t.Errorf("expected 'ok', got %q", out)
	}
}
