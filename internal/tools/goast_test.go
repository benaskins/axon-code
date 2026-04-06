package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tool "github.com/benaskins/axon-tool"
)

const testGoFile = `package calc

import "errors"

// Add returns the sum of a and b.
func Add(a, b int) int {
	return a - b
}

// Subtract returns a minus b.
func Subtract(a, b int) int {
	return a - b
}

// Divide returns a divided by b.
func Divide(a, b int) int {
	return a / b
}

var _ = errors.New
`

func setupGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(testGoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func execTool(t *testing.T, td tool.ToolDef, args map[string]any) string {
	t.Helper()
	result := td.Execute(&tool.ToolContext{}, args)
	if strings.HasPrefix(result.Content, "error:") {
		t.Fatalf("tool error: %s", result.Content)
	}
	return result.Content
}

func TestInspect(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	out := execTool(t, tools.InspectTool, map[string]any{"path": "calc.go"})

	// Should contain package name.
	if !strings.Contains(out, "package calc") {
		t.Errorf("missing package name in output:\n%s", out)
	}
	// Should list functions with line numbers.
	if !strings.Contains(out, "func Add(a, b int) int") {
		t.Errorf("missing Add function in output:\n%s", out)
	}
	if !strings.Contains(out, "func Subtract(a, b int) int") {
		t.Errorf("missing Subtract function in output:\n%s", out)
	}
	if !strings.Contains(out, "func Divide(a, b int) int") {
		t.Errorf("missing Divide function in output:\n%s", out)
	}
	// Should contain line numbers in [start:end] format.
	if !strings.Contains(out, "[") {
		t.Errorf("missing line numbers in output:\n%s", out)
	}
}

func TestInspect_NonGoFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("key: value"), 0o644)
	tools := NewGoASTTools(dir, NewManifest())

	result := tools.InspectTool.Execute(&tool.ToolContext{}, map[string]any{"path": "config.yaml"})
	if !strings.Contains(result.Content, "not a .go file") {
		t.Errorf("expected error for non-Go file, got: %s", result.Content)
	}
}

func TestRename(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	execTool(t, tools.RewriteTool, map[string]any{
		"path":      "calc.go",
		"operation": "rename",
		"target":    "Add",
		"name":      "Sum",
	})

	// Read back and verify.
	data, _ := os.ReadFile(filepath.Join(dir, "calc.go"))
	src := string(data)

	if strings.Contains(src, "func Add(") {
		t.Error("old function name 'Add' still present")
	}
	if !strings.Contains(src, "func Sum(") {
		t.Error("new function name 'Sum' not found")
	}
}

func TestReplaceBody(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	execTool(t, tools.RewriteTool, map[string]any{
		"path":      "calc.go",
		"operation": "replace_body",
		"target":    "Add",
		"code":      "return a + b",
	})

	data, _ := os.ReadFile(filepath.Join(dir, "calc.go"))
	src := string(data)

	if strings.Contains(src, "a - b") && strings.Contains(src, "func Add") {
		// The Add function should no longer have a - b
		// But Subtract still has a - b, so check specifically in Add context
	}
	if !strings.Contains(src, "a + b") {
		t.Error("expected 'a + b' in rewritten Add function")
	}
}

func TestReplaceReturn(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	execTool(t, tools.RewriteTool, map[string]any{
		"path":      "calc.go",
		"operation": "replace_return",
		"target":    "Add",
		"code":      "a + b",
	})

	data, _ := os.ReadFile(filepath.Join(dir, "calc.go"))
	src := string(data)

	if !strings.Contains(src, "a + b") {
		t.Error("expected 'a + b' in rewritten return")
	}
	// Subtract should still have a - b.
	if !strings.Contains(src, "return a - b") {
		t.Error("Subtract was modified unexpectedly")
	}
}

func TestChangeSignature(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	execTool(t, tools.RewriteTool, map[string]any{
		"path":      "calc.go",
		"operation": "change_signature",
		"target":    "Divide",
		"code":      "(a, b int) (int, error)",
	})

	data, _ := os.ReadFile(filepath.Join(dir, "calc.go"))
	src := string(data)

	if !strings.Contains(src, "(int, error)") {
		t.Errorf("expected '(int, error)' return type, got:\n%s", src)
	}
}

func TestRename_NotFound(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	result := tools.RewriteTool.Execute(&tool.ToolContext{}, map[string]any{
		"path":      "calc.go",
		"operation": "rename",
		"target":    "NonExistent",
		"name":      "Whatever",
	})
	if !strings.Contains(result.Content, "not found") {
		t.Errorf("expected not found error, got: %s", result.Content)
	}
}

func TestReplaceBody_InvalidCode(t *testing.T) {
	dir := setupGoProject(t)
	tools := NewGoASTTools(dir, NewManifest())

	result := tools.RewriteTool.Execute(&tool.ToolContext{}, map[string]any{
		"path":      "calc.go",
		"operation": "replace_body",
		"target":    "Add",
		"code":      "this is not valid go {{{",
	})
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for invalid code, got: %s", result.Content)
	}
}
