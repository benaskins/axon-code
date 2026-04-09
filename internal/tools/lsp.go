package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/internal/sandbox"
)

// LSPTools holds gopls-based ToolDefs bound to a project directory.
type LSPTools struct {
	DiagnosticsTool tool.ToolDef
	DefinitionTool  tool.ToolDef
}

// NewLSPTools constructs LSPTools bound to projectDir.
func NewLSPTools(projectDir string, timeout time.Duration) LSPTools {
	return LSPTools{
		DiagnosticsTool: makeDiagnosticsTool(projectDir, timeout),
		DefinitionTool:  makeDefinitionTool(projectDir, timeout),
	}
}

func makeDiagnosticsTool(projectDir string, defaultTimeout time.Duration) tool.ToolDef {
	return tool.ToolDef{
		Name:        "lsp_diagnostics",
		Description: "Run gopls diagnostics on a Go file. Returns type errors, analysis warnings, and suggestions with file:line:col detail. More specific than 'go vet'. Use this after a failed build to understand exactly what is wrong.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]tool.PropertySchema{
				"path":     {Type: "string", Description: "Path to a .go file, relative to the project directory."},
				"severity": {Type: "string", Description: "Minimum severity: hint, info, warning, or error. Defaults to warning."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("lsp_diagnostics: missing required arg 'path'")
			}
			if _, err := sandbox.Resolve(projectDir, path); err != nil {
				return errResult("lsp_diagnostics: " + err.Error())
			}

			severity := "warning"
			if s, ok := stringArg(a, "severity"); ok {
				severity = s
			}

			timeout := defaultTimeout
			if secs, ok := a["timeout_seconds"]; ok {
				if v, ok := secs.(float64); ok {
					timeout = time.Duration(v * float64(time.Second))
				}
			}

			runCtx, cancel := context.WithTimeout(ctx.Ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(runCtx, "gopls", "check", fmt.Sprintf("-severity=%s", severity), path)
			cmd.Dir = projectDir

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			result := CmdResult{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
				} else {
					return errResult("lsp_diagnostics: " + err.Error())
				}
			}

			data, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return errResult("lsp_diagnostics: failed to marshal result: " + jsonErr.Error())
			}
			return tool.ToolResult{Content: string(data)}
		},
	}
}

func makeDefinitionTool(projectDir string, defaultTimeout time.Duration) tool.ToolDef {
	return tool.ToolDef{
		Name:        "lsp_definition",
		Description: "Look up the definition of a symbol at a given file position. Returns the declaration site, type signature, and documentation. Use this to discover API shapes of dependency packages when you encounter unknown types.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path", "line", "column"},
			Properties: map[string]tool.PropertySchema{
				"path":   {Type: "string", Description: "Path to a .go file, relative to the project directory."},
				"line":   {Type: "integer", Description: "Line number (1-based)."},
				"column": {Type: "integer", Description: "Column number (1-based, byte offset)."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("lsp_definition: missing required arg 'path'")
			}
			if _, err := sandbox.Resolve(projectDir, path); err != nil {
				return errResult("lsp_definition: " + err.Error())
			}

			line, ok := a["line"]
			if !ok {
				return errResult("lsp_definition: missing required arg 'line'")
			}
			col, ok := a["column"]
			if !ok {
				return errResult("lsp_definition: missing required arg 'column'")
			}

			lineNum := intArg(a, "line", 0)
			colNum := intArg(a, "column", 0)
			if lineNum <= 0 || colNum <= 0 {
				return errResult(fmt.Sprintf("lsp_definition: invalid position line=%v column=%v", line, col))
			}

			timeout := defaultTimeout
			if secs, ok := a["timeout_seconds"]; ok {
				if v, ok := secs.(float64); ok {
					timeout = time.Duration(v * float64(time.Second))
				}
			}

			runCtx, cancel := context.WithTimeout(ctx.Ctx, timeout)
			defer cancel()

			pos := fmt.Sprintf("%s:%d:%d", path, lineNum, colNum)
			cmd := exec.CommandContext(runCtx, "gopls", "definition", pos)
			cmd.Dir = projectDir

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			result := CmdResult{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
				} else {
					return errResult("lsp_definition: " + err.Error())
				}
			}

			data, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return errResult("lsp_definition: failed to marshal result: " + jsonErr.Error())
			}
			return tool.ToolResult{Content: string(data)}
		},
	}
}
