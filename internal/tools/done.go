package tools

import (
	"fmt"
	"time"

	tool "github.com/benaskins/axon-tool"
)

// DoneSignal is populated by the done tool when the agent signals completion.
// The caller inspects Done, Summary, and Files after the loop exits.
type DoneSignal struct {
	Done    bool
	Summary string
	Files   []string // paths modified during this session (relative to project dir)
}

// Manifest tracks files modified by tool execution.
// It is shared across write_file, edit_file, and rewrite tools.
type Manifest struct {
	files map[string]bool
}

// NewManifest creates an empty file manifest.
func NewManifest() *Manifest {
	return &Manifest{files: make(map[string]bool)}
}

// Track records a file path as modified.
func (m *Manifest) Track(path string) {
	m.files[path] = true
}

// Files returns the deduplicated list of modified file paths.
func (m *Manifest) Files() []string {
	out := make([]string, 0, len(m.files))
	for f := range m.files {
		out = append(out, f)
	}
	return out
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

// newDoneToolWithManifest returns a done tool that copies the manifest into the signal.
func newDoneToolWithManifest(signal *DoneSignal, manifest *Manifest) tool.ToolDef {
	td := NewDoneTool(signal)
	origExec := td.Execute
	td.Execute = func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
		signal.Files = manifest.Files()
		return origExec(ctx, a)
	}
	return td
}

// Config carries tool-level configuration for the registry builder.
type Config struct {
	// BashTimeout is the default execution timeout for the bash tool.
	// If zero, a 30-second default is applied.
	BashTimeout time.Duration
}
