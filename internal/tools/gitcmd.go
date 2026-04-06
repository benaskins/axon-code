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

// GitCmdTools holds the git_cmd ToolDef bound to a project directory.
type GitCmdTools struct {
	GitCmdTool tool.ToolDef
}

// NewGitCmdTool constructs GitCmdTools bound to projectDir.
func NewGitCmdTool(projectDir string, defaultTimeout time.Duration) GitCmdTools {
	return GitCmdTools{
		GitCmdTool: makeGitCmdTool(projectDir, defaultTimeout),
	}
}

// allowed git subcommands
var allowedGitSubcommands = map[string]bool{
	"add":    true,
	"commit": true,
	"status": true,
	"diff":   true,
	"log":    true,
	"branch": true,
	"show":   true,
	"rev-parse": true,
	"ls-files":  true,
}

func makeGitCmdTool(projectDir string, defaultTimeout time.Duration) tool.ToolDef {
	subcommands := make([]string, 0, len(allowedGitSubcommands))
	for k := range allowedGitSubcommands {
		subcommands = append(subcommands, k)
	}

	return tool.ToolDef{
		Name:        "git_cmd",
		Description: fmt.Sprintf("Run a git command in the project directory. Allowed subcommands: %s.", strings.Join(subcommands, ", ")),
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"args"},
			Properties: map[string]tool.PropertySchema{
				"args":            {Type: "string", Description: "Arguments to pass to git (e.g. 'add .' or 'commit -m \"feat: add parser\"')."},
				"timeout_seconds": {Type: "number", Description: "Maximum seconds to wait. Defaults to the configured timeout."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			args, ok := stringArg(a, "args")
			if !ok {
				return errResult("git_cmd: missing required arg 'args'")
			}

			// Validate subcommand
			parts := strings.Fields(args)
			if len(parts) == 0 {
				return errResult("git_cmd: empty args")
			}
			if !allowedGitSubcommands[parts[0]] {
				return errResult(fmt.Sprintf("git_cmd: subcommand %q not allowed", parts[0]))
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

			cmd := exec.CommandContext(runCtx, "git", parts...)
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
					result.ExitCode = -1
					if result.Stderr == "" {
						result.Stderr = err.Error()
					}
				}
			}

			data, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				return errResult("git_cmd: failed to marshal result: " + jsonErr.Error())
			}
			return tool.ToolResult{Content: string(data)}
		},
	}
}
