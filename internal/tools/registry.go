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
//
// Tools that modify files (write_file, edit_file, rewrite) track their changes
// in a manifest. After the loop exits, DoneSignal.Files contains the list of
// modified paths.
func Build(projectDir string, cfg Config) ([]tool.ToolDef, *DoneSignal, error) {
	timeout := cfg.BashTimeout
	if timeout <= 0 {
		timeout = defaultBashTimeout
	}

	signal := &DoneSignal{}
	manifest := NewManifest()

	fs := NewFSTools(projectDir, manifest)
	search := NewSearchTools(projectDir)
	goast := NewGoASTTools(projectDir, manifest)
	gocmd := NewGoCmdTool(projectDir, timeout)
	gitcmd := NewGitCmdTool(projectDir, timeout)
	domain := NewDomainTools(projectDir)
	done := newDoneToolWithManifest(signal, manifest)

	tools := []tool.ToolDef{
		goast.InspectTool,
		goast.RewriteTool,
		fs.ReadTool,
		fs.WriteTool,
		fs.EditTool,
		fs.ListTool,
		search.GrepTool,
		search.GlobTool,
		gocmd.GoCmdTool,
		gitcmd.GitCmdTool,
		domain.InspectProjectTool,
		done,
	}

	return tools, signal, nil
}
