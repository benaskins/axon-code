package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/internal/sandbox"
)

// SearchTools holds the grep and glob ToolDefs bound to a project directory.
type SearchTools struct {
	GrepTool tool.ToolDef
	GlobTool tool.ToolDef
}

// NewSearchTools constructs SearchTools bound to projectDir.
// Every path argument is resolved through sandbox.Resolve before any OS call.
func NewSearchTools(projectDir string) SearchTools {
	return SearchTools{
		GrepTool: makeGrepTool(projectDir),
		GlobTool: makeGlobTool(projectDir),
	}
}

func makeGrepTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "grep",
		Description: "Search for a regex pattern in files within the project directory. Returns matching lines with filename:linenum prefix.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"pattern"},
			Properties: map[string]tool.PropertySchema{
				"pattern":   {Type: "string", Description: "Regular expression pattern to search for."},
				"path":      {Type: "string", Description: "Directory to search (relative to project root). Defaults to '.'."},
				"file_type": {Type: "string", Description: "File extension filter (e.g. 'go', 'txt'). If omitted, all files are searched."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			pattern, ok := stringArg(a, "pattern")
			if !ok {
				return errResult("grep: missing required arg 'pattern'")
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return errResult("grep: invalid pattern: " + err.Error())
			}

			path, _ := stringArg(a, "path")
			if path == "" {
				path = "."
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("grep: " + err.Error())
			}

			fileType, _ := stringArg(a, "file_type")
			ext := ""
			if fileType != "" {
				ext = "." + strings.TrimPrefix(fileType, ".")
			}

			var sb strings.Builder
			_ = filepath.Walk(abs, func(fpath string, info os.FileInfo, werr error) error {
				if werr != nil || info.IsDir() {
					return nil
				}
				if ext != "" && filepath.Ext(fpath) != ext {
					return nil
				}
				rel, _ := filepath.Rel(projectDir, fpath)

				f, err := os.Open(fpath)
				if err != nil {
					return nil
				}
				defer f.Close()

				scanner := bufio.NewScanner(f)
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()
					if re.MatchString(line) {
						fmt.Fprintf(&sb, "%s:%d:%s\n", rel, lineNum, line)
					}
				}
				return nil
			})

			return tool.ToolResult{Content: sb.String()}
		},
	}
}

func makeGlobTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "glob",
		Description: "Find files matching a glob pattern within the project directory. Returns matching paths relative to the project root.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"pattern"},
			Properties: map[string]tool.PropertySchema{
				"pattern": {Type: "string", Description: "Glob pattern relative to project root (e.g. '*.go', 'src/*.go')."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			pattern, ok := stringArg(a, "pattern")
			if !ok {
				return errResult("glob: missing required arg 'pattern'")
			}

			// Reject traversal: any component that is ".." escapes the sandbox.
			for _, part := range strings.Split(filepath.ToSlash(pattern), "/") {
				if part == ".." {
					return errResult("glob: pattern escapes root: " + pattern)
				}
			}

			absPattern := filepath.Join(projectDir, pattern)
			matches, err := filepath.Glob(absPattern)
			if err != nil {
				return errResult("glob: " + err.Error())
			}

			var sb strings.Builder
			for _, m := range matches {
				rel, err := filepath.Rel(projectDir, m)
				if err != nil {
					continue
				}
				fmt.Fprintln(&sb, rel)
			}
			return tool.ToolResult{Content: sb.String()}
		},
	}
}
