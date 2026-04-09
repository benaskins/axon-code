package tools_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benaskins/axon-code/internal/tools"
)

func skipIfNoGopls(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not on PATH")
	}
}

// setupGoModule creates a minimal Go module in dir with the given files.
func setupGoModule(t *testing.T, dir string, files map[string]string) {
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

const testTimeout = 30 * time.Second

// --- lsp_diagnostics ---

func TestLSPDiagnostics_MissingPath(t *testing.T) {
	dir := t.TempDir()
	lsp := tools.NewLSPTools(dir, testTimeout)
	result := lsp.DiagnosticsTool.Execute(toolCtx(), args())
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing path, got: %s", result.Content)
	}
}

func TestLSPDiagnostics_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	lsp := tools.NewLSPTools(dir, testTimeout)
	result := lsp.DiagnosticsTool.Execute(toolCtx(), args("path", "../../etc/passwd"))
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected sandbox error, got: %s", result.Content)
	}
}

// --- lsp_definition ---

func TestLSPDefinition_MissingArgs(t *testing.T) {
	dir := t.TempDir()
	lsp := tools.NewLSPTools(dir, testTimeout)

	// missing all args
	result := lsp.DefinitionTool.Execute(toolCtx(), args())
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing path, got: %s", result.Content)
	}

	// missing line and column
	result = lsp.DefinitionTool.Execute(toolCtx(), args("path", "main.go"))
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing line, got: %s", result.Content)
	}

	// missing column
	result = lsp.DefinitionTool.Execute(toolCtx(), args("path", "main.go", "line", 5.0))
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected error for missing column, got: %s", result.Content)
	}
}

func TestLSPDefinition_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	lsp := tools.NewLSPTools(dir, testTimeout)
	result := lsp.DefinitionTool.Execute(toolCtx(), args("path", "../../etc/passwd", "line", 1.0, "column", 1.0))
	if !strings.Contains(result.Content, "error") {
		t.Errorf("expected sandbox error, got: %s", result.Content)
	}
}

// --- Integration tests (require gopls) ---

func TestLSPDiagnostics_CleanFile(t *testing.T) {
	skipIfNoGopls(t)
	dir := t.TempDir()
	setupGoModule(t, dir, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})

	lsp := tools.NewLSPTools(dir, testTimeout)
	result := lsp.DiagnosticsTool.Execute(toolCtx(), args("path", "main.go"))
	if strings.Contains(result.Content, "error:") {
		t.Errorf("expected no tool error for clean file, got: %s", result.Content)
	}
	// Clean file should have exit code 0 and no diagnostics
	if strings.Contains(result.Content, "cannot") {
		t.Errorf("clean file should have no type errors, got: %s", result.Content)
	}
}

func TestLSPDiagnostics_TypeError(t *testing.T) {
	skipIfNoGopls(t)
	dir := t.TempDir()
	setupGoModule(t, dir, map[string]string{
		"main.go": `package main

func main() {
	var x int = "hello"
	_ = x
}
`,
	})

	lsp := tools.NewLSPTools(dir, testTimeout)
	result := lsp.DiagnosticsTool.Execute(toolCtx(), args("path", "main.go"))
	if strings.Contains(result.Content, "error:") {
		t.Errorf("expected CmdResult (not tool error) for type error, got: %s", result.Content)
	}
	// gopls should report the type mismatch
	if !strings.Contains(result.Content, "cannot") {
		t.Errorf("expected type error diagnostic containing 'cannot', got: %s", result.Content)
	}
}

func TestLSPDefinition_FindsFunction(t *testing.T) {
	skipIfNoGopls(t)
	dir := t.TempDir()
	setupGoModule(t, dir, map[string]string{
		"main.go": `package main

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }

func main() { _ = Add(1, 2) }
`,
	})

	lsp := tools.NewLSPTools(dir, testTimeout)
	// Line 6: "func main() { _ = Add(1, 2) }" -- "A" in Add is at column 19
	result := lsp.DefinitionTool.Execute(toolCtx(), args("path", "main.go", "line", 6.0, "column", 19.0))
	if strings.Contains(result.Content, "error:") {
		t.Errorf("expected CmdResult (not tool error), got: %s", result.Content)
	}
	// Should find the definition and show the function signature or doc
	if !strings.Contains(result.Content, "Add") {
		t.Errorf("expected definition output to mention 'Add', got: %s", result.Content)
	}
}

func TestLSPDefinition_InvalidPosition(t *testing.T) {
	skipIfNoGopls(t)
	dir := t.TempDir()
	setupGoModule(t, dir, map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
	})

	lsp := tools.NewLSPTools(dir, testTimeout)
	// Line 1, column 1 is on "package" keyword -- not an identifier
	result := lsp.DefinitionTool.Execute(toolCtx(), args("path", "main.go", "line", 1.0, "column", 1.0))
	// Should return a CmdResult with gopls error, not a tool-level crash
	if strings.HasPrefix(result.Content, "error:") {
		t.Errorf("expected CmdResult (not tool error) for invalid position, got: %s", result.Content)
	}
}
