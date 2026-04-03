package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"time"

	tool "github.com/benaskins/axon-tool"
)

// BashResult is the JSON-serialisable output of the bash tool.
type BashResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// BashTools holds the bash ToolDef bound to a project directory.
type BashTools struct {
	BashTool tool.ToolDef
}

// NewBashTool constructs BashTools bound to projectDir with the given default timeout.
func NewBashTool(projectDir string, defaultTimeout time.Duration) BashTools {
	return BashTools{
		BashTool: makeBashTool(projectDir, defaultTimeout),
	}
}

func makeBashTool(projectDir string, defaultTimeout time.Duration) tool.ToolDef {
	return tool.ToolDef{
		Name:        "bash",
		Description: "Execute a shell command in the project directory. Returns stdout, stderr, and exit_code as JSON.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"command"},
			Properties: map[string]tool.PropertySchema{
				"command":         {Type: "string", Description: "Shell command to execute via sh -c."},
				"timeout_seconds": {Type: "number", Description: "Maximum seconds to wait. Defaults to the configured timeout."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			command, ok := stringArg(a, "command")
			if !ok {
				return errResult("bash: missing required arg 'command'")
			}

			timeout := defaultTimeout
			if secs, ok := a["timeout_seconds"]; ok {
				switch v := secs.(type) {
				case float64:
					timeout = time.Duration(v * float64(time.Second))
				case int:
					timeout = time.Duration(v) * time.Second
				}
			}

			runCtx, cancel := context.WithTimeout(ctx.Ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(runCtx, "sh", "-c", command)
			cmd.Dir = projectDir

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			result := BashResult{
				Stdout: stdout.String(),
				Stderr: stderr.String(),
			}
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitErr.ExitCode()
				} else {
					// context deadline exceeded or other non-exit error
					result.ExitCode = -1
					if result.Stderr == "" {
						result.Stderr = err.Error()
					}
				}
			}

			data, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return errResult("bash: failed to marshal result: " + jsonErr.Error())
			}
			return tool.ToolResult{Content: string(data)}
		},
	}
}
