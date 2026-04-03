package tools_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/benaskins/axon-code/internal/tools"
)

// TestGrepFindsPatternInNestedFiles verifies grep finds matching lines in nested files.
func TestGrepFindsPatternInNestedFiles(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir)
	st := tools.NewSearchTools(dir)

	fs.WriteTool.Execute(toolCtx(), args("path", "top.txt", "content", "hello world\nskip me\n"))
	fs.WriteTool.Execute(toolCtx(), args("path", "sub/nested.txt", "content", "another hello here\nnope\n"))

	result := st.GrepTool.Execute(toolCtx(), args("pattern", "hello"))
	if !strings.Contains(result.Content, "top.txt") {
		t.Errorf("expected top.txt in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "nested.txt") {
		t.Errorf("expected nested.txt in output, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "skip me") {
		t.Errorf("did not expect non-matching line 'skip me', got: %s", result.Content)
	}
	if strings.Contains(result.Content, "nope") {
		t.Errorf("did not expect non-matching line 'nope', got: %s", result.Content)
	}
}

// TestGrepFileTypeFilter verifies grep filters by file extension.
func TestGrepFileTypeFilter(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir)
	st := tools.NewSearchTools(dir)

	fs.WriteTool.Execute(toolCtx(), args("path", "main.go", "content", "package main // target\n"))
	fs.WriteTool.Execute(toolCtx(), args("path", "readme.txt", "content", "target is here too\n"))

	result := st.GrepTool.Execute(toolCtx(), args("pattern", "target", "file_type", "go"))
	if !strings.Contains(result.Content, "main.go") {
		t.Errorf("expected main.go in output, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "readme.txt") {
		t.Errorf("did not expect readme.txt when file_type=go, got: %s", result.Content)
	}
}

// TestGrepNoMatch verifies grep returns empty content when no lines match.
func TestGrepNoMatch(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir)
	st := tools.NewSearchTools(dir)

	fs.WriteTool.Execute(toolCtx(), args("path", "data.txt", "content", "nothing here\n"))

	result := st.GrepTool.Execute(toolCtx(), args("pattern", "XYZNOTFOUND"))
	if result.Content != "" {
		t.Errorf("expected empty output on no match, got: %s", result.Content)
	}
}

// TestGrepTraversalRejected verifies grep rejects path traversal.
func TestGrepTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	st := tools.NewSearchTools(dir)

	result := st.GrepTool.Execute(toolCtx(), args("pattern", "x", "path", filepath.Join("..", "escape")))
	if !strings.Contains(result.Content, "error") && !strings.Contains(result.Content, "escapes") {
		t.Errorf("expected traversal error, got: %s", result.Content)
	}
}

// TestGlobMatchesExpectedFiles verifies glob returns matching file paths.
func TestGlobMatchesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir)
	st := tools.NewSearchTools(dir)

	fs.WriteTool.Execute(toolCtx(), args("path", "a.go", "content", "a"))
	fs.WriteTool.Execute(toolCtx(), args("path", "b.go", "content", "b"))
	fs.WriteTool.Execute(toolCtx(), args("path", "c.txt", "content", "c"))

	result := st.GlobTool.Execute(toolCtx(), args("pattern", "*.go"))
	if !strings.Contains(result.Content, "a.go") {
		t.Errorf("expected a.go in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "b.go") {
		t.Errorf("expected b.go in output, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "c.txt") {
		t.Errorf("did not expect c.txt in *.go results, got: %s", result.Content)
	}
}

// TestGlobTraversalRejected verifies glob rejects patterns that escape the sandbox.
func TestGlobTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	st := tools.NewSearchTools(dir)

	result := st.GlobTool.Execute(toolCtx(), args("pattern", "../../*.go"))
	if !strings.Contains(result.Content, "error") && !strings.Contains(result.Content, "escapes") {
		t.Errorf("expected traversal error, got: %s", result.Content)
	}
}
