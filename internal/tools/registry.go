package tools

import (
	"time"

	tool "github.com/benaskins/axon-tool"
)

const defaultBashTimeout = 30 * time.Second

// Build instantiates all tools bound to projectDir and returns them as a slice
// alongside a DoneSignal that the caller can inspect after the loop exits.
func Build(projectDir string, cfg Config) ([]tool.ToolDef, *DoneSignal, error) {
	timeout := cfg.BashTimeout
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}

	signal := &DoneSignal{}

	fs := NewFSTools(projectDir)
	search := NewSearchTools(projectDir)
	bash := NewBashTool(projectDir, timeout)
	done := NewDoneTool(signal)

	tools := []tool.ToolDef{
		fs.ReadTool,
		fs.WriteTool,
		fs.EditTool,
		fs.ListTool,
		search.GrepTool,
		search.GlobTool,
		bash.BashTool,
		done,
	}

	return tools, signal, nil
}
