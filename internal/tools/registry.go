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

	fs := NewFSTools(projectDir, cfg.GoAST)
	search := NewSearchTools(projectDir)
	gocmd := NewGoCmdTool(projectDir, timeout)
	gitcmd := NewGitCmdTool(projectDir, timeout)
	done := NewDoneTool(signal)

	var tools []tool.ToolDef

	if cfg.GoAST {
		// AST-aware tool set: ast + rewrite for Go files, read_file for
		// non-Go files (config, markdown, etc), write_file for new files.
		// No bash: use go_cmd and git_cmd for toolchain operations.
		goast := NewGoASTTools(projectDir)
		tools = []tool.ToolDef{
			goast.InspectTool,
			goast.RewriteTool,
			fs.ReadTool,
			fs.WriteTool,
			fs.ListTool,
			search.GrepTool,
			search.GlobTool,
			gocmd.GoCmdTool,
			gitcmd.GitCmdTool,
			done,
		}
	} else {
		tools = []tool.ToolDef{
			fs.ReadTool,
			fs.WriteTool,
			fs.EditTool,
			fs.ListTool,
			search.GrepTool,
			search.GlobTool,
			gocmd.GoCmdTool,
			gitcmd.GitCmdTool,
			done,
		}
	}

	return tools, signal, nil
}
