package code

import (
	tool "github.com/benaskins/axon-tool"

	internaltools "github.com/benaskins/axon-code/internal/tools"
)

// DoneSignal is populated by the done tool when the agent signals completion.
type DoneSignal = internaltools.DoneSignal

// ToolsConfig configures tool building.
type ToolsConfig struct {
	// GoAST enables AST-aware tools (ast, rewrite) for Go source files.
	GoAST bool
}

// BuildTools returns the standard coding tool set bound to projectDir,
// plus a DoneSignal the caller can inspect after the loop exits.
// Tools are returned as a map keyed by tool name.
func BuildTools(projectDir string, cfg ToolsConfig) (map[string]tool.ToolDef, *DoneSignal, error) {
	defs, signal, err := internaltools.Build(projectDir, internaltools.Config{
		GoAST: cfg.GoAST,
	})
	if err != nil {
		return nil, nil, err
	}

	m := make(map[string]tool.ToolDef, len(defs))
	for _, td := range defs {
		m[td.Name] = td
	}
	return m, signal, nil
}
