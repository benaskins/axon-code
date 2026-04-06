package tools

import (
	"time"

	tool "github.com/benaskins/axon-tool"
)

const defaultBashTimeout = 30 * time.Second

// Build instantiates all tools bound to projectDir and returns them as a slice
// alongside a DoneSignal that the caller can inspect after the loop exits.
//
// The standard tool set is AST-first: ast and rewrite for Go source files,
// read_file for non-Go files (config, markdown, YAML), write_file for creating
// or overwriting files. All Go toolchain and git operations go through go_cmd
// and git_cmd respectively.
func Build(projectDir string, cfg Config) ([]tool.ToolDef, *DoneSignal, error) {
	timeout := cfg.BashTimeout
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}

	signal := &DoneSignal{}

	fs := NewFSTools(projectDir)
	search := NewSearchTools(projectDir)
	goast := NewGoASTTools(projectDir)
	gocmd := NewGoCmdTool(projectDir, timeout)
	gitcmd := NewGitCmdTool(projectDir, timeout)
	domain := NewDomainTools(projectDir)
	done := NewDoneTool(signal)

	tools := []tool.ToolDef{
		goast.InspectTool,
		goast.RewriteTool,
		fs.ReadTool,
		fs.WriteTool,
		fs.ListTool,
		search.GrepTool,
		search.GlobTool,
		gocmd.GoCmdTool,
		gitcmd.GitCmdTool,
		domain.SummariseProjectTool,
		domain.InspectProjectTool,
		done,
	}

	return tools, signal, nil
}
