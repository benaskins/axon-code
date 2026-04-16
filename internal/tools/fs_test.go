package tools_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/internal/tools"
)

func toolCtx() *tool.ToolContext {
	return &tool.ToolContext{Ctx: context.Background()}
}

func args(pairs ...any) map[string]any {
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i].(string)] = pairs[i+1]
	}
	return m
}

// TestWriteReadRoundTrip verifies write_file + read_file produce the same content.
func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	writeResult := fs.WriteTool.Execute(toolCtx(), args("path", "hello.txt", "content", "line1\nline2\nline3\n"))
	if strings.Contains(writeResult.Content, "error") {
		t.Fatalf("write_file failed: %s", writeResult.Content)
	}

	readResult := fs.ReadTool.Execute(toolCtx(), args("path", "hello.txt"))
	if !strings.Contains(readResult.Content, "line1") {
		t.Errorf("expected line1 in output, got: %s", readResult.Content)
	}
	if !strings.Contains(readResult.Content, "line3") {
		t.Errorf("expected line3 in output, got: %s", readResult.Content)
	}
}

// TestReadWithOffsetAndLimit verifies partial reads.
func TestReadWithOffsetAndLimit(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	content := "alpha\nbeta\ngamma\ndelta\nepsilon\n"
	fs.WriteTool.Execute(toolCtx(), args("path", "lines.txt", "content", content))

	// offset=1, limit=2 => lines 2 and 3 (beta, gamma)
	result := fs.ReadTool.Execute(toolCtx(), args("path", "lines.txt", "offset", float64(1), "limit", float64(2)))
	if !strings.Contains(result.Content, "beta") {
		t.Errorf("expected beta, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "gamma") {
		t.Errorf("expected gamma, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "alpha") {
		t.Errorf("did not expect alpha (before offset), got: %s", result.Content)
	}
	if strings.Contains(result.Content, "delta") {
		t.Errorf("did not expect delta (after limit), got: %s", result.Content)
	}
}

// TestWriteCreatesParentDirs verifies write_file creates parent dirs.
func TestWriteCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	result := fs.WriteTool.Execute(toolCtx(), args("path", "sub/nested/file.txt", "content", "hello"))
	if strings.Contains(result.Content, "error") {
		t.Fatalf("write_file failed: %s", result.Content)
	}

	readResult := fs.ReadTool.Execute(toolCtx(), args("path", "sub/nested/file.txt"))
	if !strings.Contains(readResult.Content, "hello") {
		t.Errorf("expected hello, got: %s", readResult.Content)
	}
}

// TestReadGoFiles verifies read_file can read .go files.
func TestReadGoFiles(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	fs.WriteTool.Execute(toolCtx(), args("path", "main.go", "content", "package main\n"))

	result := fs.ReadTool.Execute(toolCtx(), args("path", "main.go"))
	if !strings.Contains(result.Content, "package main") {
		t.Errorf("expected file content, got: %s", result.Content)
	}
}

// TestListDir verifies list_dir returns immediate children with types.
func TestListDir(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	fs.WriteTool.Execute(toolCtx(), args("path", "afile.txt", "content", "x"))
	fs.WriteTool.Execute(toolCtx(), args("path", "subdir/child.txt", "content", "y"))

	result := fs.ListTool.Execute(toolCtx(), args("path", "."))
	if !strings.Contains(result.Content, "afile.txt") {
		t.Errorf("expected afile.txt, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "subdir") {
		t.Errorf("expected subdir, got: %s", result.Content)
	}
	// child.txt should NOT appear (not immediate child of root)
	if strings.Contains(result.Content, "child.txt") {
		t.Errorf("did not expect child.txt in root listing, got: %s", result.Content)
	}
}

// TestListDirTypeLabels verifies list_dir labels files and dirs.
func TestListDirTypeLabels(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	fs.WriteTool.Execute(toolCtx(), args("path", "readme.md", "content", "doc"))
	fs.WriteTool.Execute(toolCtx(), args("path", "pkg/main.go", "content", "pkg"))

	result := fs.ListTool.Execute(toolCtx(), args("path", "."))
	if !strings.Contains(result.Content, "dir") || !strings.Contains(result.Content, "file") {
		t.Errorf("expected both dir and file labels, got: %s", result.Content)
	}
}

// TestScratchPathRejected verifies write_file and edit_file reject tmp/ paths.
func TestScratchPathRejected(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	// write_file to tmp/ should be rejected
	wr := fs.WriteTool.Execute(toolCtx(), args("path", "tmp/explore.go", "content", "package main"))
	if !strings.Contains(wr.Content, "error") {
		t.Errorf("write_file tmp/explore.go should be rejected, got: %s", wr.Content)
	}
	if !strings.Contains(wr.Content, "scratch") {
		t.Errorf("expected scratch dir message, got: %s", wr.Content)
	}

	// write_file to tmp/nested/ should also be rejected
	wr2 := fs.WriteTool.Execute(toolCtx(), args("path", "tmp/nested/file.go", "content", "package foo"))
	if !strings.Contains(wr2.Content, "error") {
		t.Errorf("write_file tmp/nested/file.go should be rejected, got: %s", wr2.Content)
	}

	// edit_file to tmp/ should be rejected (even if file existed)
	er := fs.EditTool.Execute(toolCtx(), args("path", "tmp/explore.go", "old_string", "x", "new_string", "y"))
	if !strings.Contains(er.Content, "error") {
		t.Errorf("edit_file tmp/explore.go should be rejected, got: %s", er.Content)
	}

	// read_file from tmp/ should still work (not blocked)
	// First create a file manually to read
	fs2 := tools.NewFSTools(dir, tools.NewManifest(), nil)
	_ = fs2 // read is tested separately; the point is write/edit are blocked
}

// TestTraversalRejected verifies tools reject path traversal.
func TestTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	fs := tools.NewFSTools(dir, tools.NewManifest(), nil)

	escapePath := filepath.Join("..", "escape.txt")

	for _, tc := range []struct {
		name   string
		result tool.ToolResult
	}{
		{"read_file", fs.ReadTool.Execute(toolCtx(), args("path", escapePath))},
		{"write_file", fs.WriteTool.Execute(toolCtx(), args("path", escapePath, "content", "x"))},
		{"list_dir", fs.ListTool.Execute(toolCtx(), args("path", escapePath))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.result.Content, "error") && !strings.Contains(tc.result.Content, "escapes") {
				t.Errorf("%s: expected traversal error, got: %s", tc.name, tc.result.Content)
			}
		})
	}
}
