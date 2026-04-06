package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tool "github.com/benaskins/axon-tool"
)

// GoCmdTools holds the go_cmd ToolDef bound to a project directory.
type GoCmdTools struct {
	GoCmdTool tool.ToolDef
}

// NewGoCmdTool constructs GoCmdTools bound to projectDir.
func NewGoCmdTool(projectDir string, defaultTimeout time.Duration) GoCmdTools {
	return GoCmdTools{
		GoCmdTool: makeGoCmdTool(projectDir, defaultTimeout),
	}
}

// allowed go subcommands
var allowedGoSubcommands = map[string]bool{
	"build":   true,
	"test":    true,
	"vet":     true,
	"run":     true,
	"mod":     true,
	"fmt":     true,
	"install": true,
	"get":     true,
	"list":    true,
	"env":     true,
	"version": true,
	"clean":   true,
	"generate": true,
}

func makeGoCmdTool(projectDir string, defaultTimeout time.Duration) tool.ToolDef {
	subcommands := make([]string, 0, len(allowedGoSubcommands))
	for k := range allowedGoSubcommands {
		subcommands = append(subcommands, k)
	}

	return tool.ToolDef{
		Name:        "go_cmd",
		Description: fmt.Sprintf("Run a Go toolchain command in the project directory. Allowed subcommands: %s.", strings.Join(subcommands, ", ")),
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"args"},
			Properties: map[string]tool.PropertySchema{
				"args":            {Type: "string", Description: "Arguments to pass to the go command (e.g. 'build ./...' or 'test -v ./internal/...')."},
				"timeout_seconds": {Type: "number", Description: "Maximum seconds to wait. Defaults to the configured timeout."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			args, ok := stringArg(a, "args")
			if !ok {
				return errResult("go_cmd: missing required arg 'args'")
			}

			// Validate subcommand
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return errResult("go_cmd: empty args")
			}
			if !allowedGoSubcommands[parts[0]] {
				return errResult(fmt.Sprintf("go_cmd: subcommand %q not allowed", parts[0]))
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

			cmd := exec.CommandContext(runCtx, "go", parts...)
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
					result.ExitCode = -1
					if result.Stderr == "" {
						result.Stderr = err.Error()
					}
				}
			}

			data, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return errResult("go_cmd: failed to marshal result: " + jsonErr.Error())
			}
			return tool.ToolResult{Content: string(data)}
		},
	}
}
