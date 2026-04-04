package tools

import (
	"fmt"
	"time"

	tool "github.com/benaskins/axon-tool"
)

// DoneSignal is populated by the done tool when the agent signals completion.
// The caller inspects Done and Summary after the loop exits.
type DoneSignal struct {
	Done    bool
	Summary string
}

// NewDoneTool returns a ToolDef for the done tool bound to signal.
// When Execute is called the summary is stored on signal and Done is set to true.
func NewDoneTool(signal *DoneSignal) tool.ToolDef {
	return tool.ToolDef{
		Name:        "done",
		Description: "Signal that the task is complete. Call this when all requested work has been finished.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"summary"},
			Properties: map[string]tool.PropertySchema{
				"summary": {Type: "string", Description: "A brief description of what was accomplished."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			summary, ok := stringArg(a, "summary")
			if !ok {
				return errResult("done: missing required arg 'summary'")
			}
			signal.Done = true
			signal.Summary = summary
			return tool.ToolResult{Content: fmt.Sprintf("done: %s", summary)}
		},
	}
}

// Config carries tool-level configuration for the registry builder.
type Config struct {
	// BashTimeout is the default execution timeout for the bash tool.
	// If zero, a 30-second default is applied.
	BashTimeout time.Duration

	// GoAST enables AST-aware tools (ast, rewrite) for Go source files.
	// When true, these tools are added alongside the standard file tools.
	GoAST bool
}
