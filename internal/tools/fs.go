package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/internal/sandbox"
)

// FSTools holds the file-system ToolDefs bound to a project directory.
type FSTools struct {
	ReadTool  tool.ToolDef
	WriteTool tool.ToolDef
	EditTool  tool.ToolDef
	ListTool  tool.ToolDef
}

// NewFSTools constructs FSTools bound to projectDir.
// Every path argument is resolved through sandbox.Resolve before any OS call.
// read_file rejects .go files (use the ast tool instead).
func NewFSTools(projectDir string) FSTools {
	return FSTools{
		ReadTool:  makeReadTool(projectDir),
		WriteTool: makeWriteTool(projectDir),
		EditTool:  makeEditTool(projectDir),
		ListTool:  makeListTool(projectDir),
	}
}

func makeReadTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "read_file",
		Description: "Read a file within the project directory. Optional offset (0-based line index) and limit (number of lines) control the slice returned.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]tool.PropertySchema{
				"path":   {Type: "string", Description: "Path relative to the project directory."},
				"offset": {Type: "integer", Description: "First line to return (0-based). Defaults to 0."},
				"limit":  {Type: "integer", Description: "Maximum number of lines to return. 0 means all."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("read_file: missing required arg 'path'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("read_file: " + err.Error())
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return errResult("read_file: " + err.Error())
			}
			lines := strings.Split(string(data), "\n")
			// Strip trailing empty element from a trailing newline.
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			offset := intArg(a, "offset", 0)
			limit := intArg(a, "limit", 0)
			if offset > len(lines) {
				offset = len(lines)
			}
			lines = lines[offset:]
			if limit > 0 && limit < len(lines) {
				lines = lines[:limit]
			}
			// Annotate with 1-based line numbers relative to the original file.
			var sb strings.Builder
			for i, l := range lines {
				fmt.Fprintf(&sb, "%d\t%s\n", offset+i+1, l)
			}
			return tool.ToolResult{Content: sb.String()}
		},
	}
}

func makeWriteTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "write_file",
		Description: "Write content to a file within the project directory. Parent directories are created if they do not exist.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path", "content"},
			Properties: map[string]tool.PropertySchema{
				"path":    {Type: "string", Description: "Path relative to the project directory."},
				"content": {Type: "string", Description: "Content to write."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("write_file: missing required arg 'path'")
			}
			content, ok := stringArg(a, "content")
			if !ok {
				return errResult("write_file: missing required arg 'content'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("write_file: " + err.Error())
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return errResult("write_file: " + err.Error())
			}
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return errResult("write_file: " + err.Error())
			}
			return tool.ToolResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}
		},
	}
}

func makeEditTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "edit_file",
		Description: "Replace the first occurrence of old_string with new_string in a file. Works on any file type. For structural Go changes (rename, replace function body), prefer the rewrite tool.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path", "old_string", "new_string"},
			Properties: map[string]tool.PropertySchema{
				"path":       {Type: "string", Description: "Path relative to the project directory."},
				"old_string": {Type: "string", Description: "Exact string to replace."},
				"new_string": {Type: "string", Description: "Replacement string."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("edit_file: missing required arg 'path'")
			}
			oldStr, ok := stringArg(a, "old_string")
			if !ok {
				return errResult("edit_file: missing required arg 'old_string'")
			}
			newStr, ok := stringArg(a, "new_string")
			if !ok {
				return errResult("edit_file: missing required arg 'new_string'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("edit_file: " + err.Error())
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return errResult("edit_file: " + err.Error())
			}
			original := string(data)
			if !strings.Contains(original, oldStr) {
				return errResult(fmt.Sprintf("edit_file: old_string not found in %s", path))
			}
			updated := strings.Replace(original, oldStr, newStr, 1)
			if err := os.WriteFile(abs, []byte(updated), 0o644); err != nil {
				return errResult("edit_file: " + err.Error())
			}
			return tool.ToolResult{Content: fmt.Sprintf("edited %s", path)}
		},
	}
}

func makeListTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "list_dir",
		Description: "List immediate children of a directory within the project. Each entry is labelled 'file' or 'dir'.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]tool.PropertySchema{
				"path": {Type: "string", Description: "Path relative to the project directory."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("list_dir: missing required arg 'path'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("list_dir: " + err.Error())
			}
			entries, err := os.ReadDir(abs)
			if err != nil {
				return errResult("list_dir: " + err.Error())
			}
			var sb strings.Builder
			for _, e := range entries {
				kind := "file"
				if e.IsDir() {
					kind = "dir"
				}
				fmt.Fprintf(&sb, "%s\t%s\n", kind, e.Name())
			}
			return tool.ToolResult{Content: sb.String()}
		},
	}
}

// stringArg extracts a string argument from the args map.
func stringArg(a map[string]any, key string) (string, bool) {
	v, ok := a[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// intArg extracts an integer argument, returning def if absent or not a number.
// JSON numbers arrive as float64.
func intArg(a map[string]any, key string, def int) int {
	v, ok := a[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return def
}

func errResult(msg string) tool.ToolResult {
	return tool.ToolResult{Content: "error: " + msg}
}

// CmdResult is a JSON-serializable command execution result used by go_cmd and git_cmd.
type CmdResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}
