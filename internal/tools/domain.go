package tools

import (
	"fmt"
	"strings"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/inspect"
	"github.com/benaskins/axon-code/internal/sandbox"
)

// DomainTools holds project-level domain tools.
type DomainTools struct {
	InspectProjectTool tool.ToolDef
}

// NewDomainTools constructs domain tools bound to projectDir.
func NewDomainTools(projectDir string) DomainTools {
	return DomainTools{
		InspectProjectTool: makeInspectProjectTool(projectDir),
	}
}

// makeInspectProjectTool returns information about a Go project's packages,
// exported interfaces, and exported types.
func makeInspectProjectTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "inspect_project",
		Description: "Inspect a Go project to list packages, extract exported interfaces and types. Returns structured information about the project's codebase.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]tool.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Path relative to the project directory to inspect.",
				},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			path, ok := stringArg(args, "path")
			if !ok {
				return errResult("inspect_project: missing required arg 'path'")
			}

			absDir, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult(fmt.Sprintf("inspect_project: %v", err))
			}

			var result strings.Builder

			pkgs, err := inspect.ListPackagesInDir(absDir, "./...")
			if err != nil {
				return errResult(fmt.Sprintf("inspect_project: listing packages: %v", err))
			}

			result.WriteString("## Packages\n\n")
			for _, pkg := range pkgs {
				result.WriteString(fmt.Sprintf("- **%s** (%s)\n", pkg.ImportPath, pkg.Dir))
				result.WriteString(fmt.Sprintf("  - Files: %d\n", len(pkg.Files)))
			}

			result.WriteString("\n## Interfaces and Types\n\n")
			for _, pkg := range pkgs {
				ifaces, err := inspect.ExtractInterfaces(pkg.Dir)
				if err != nil {
					continue
				}
				if len(ifaces) > 0 {
					result.WriteString(fmt.Sprintf("### %s\n\n", pkg.ImportPath))
					for _, iface := range ifaces {
						result.WriteString(fmt.Sprintf("- **%s**\n", iface.Name))
						for _, m := range iface.Methods {
							result.WriteString(fmt.Sprintf("  - `%s`: %s\n", m.Name, m.Signature))
						}
					}
					result.WriteString("\n")
				}

				types, err := inspect.ExtractTypes(pkg.Dir)
				if err != nil {
					continue
				}
				if len(types) > 0 {
					result.WriteString(fmt.Sprintf("### Types in %s\n\n", pkg.ImportPath))
					for _, t := range types {
						result.WriteString(fmt.Sprintf("- **%s** (%s)\n", t.Name, t.TypeKind))
					}
					result.WriteString("\n")
				}
			}

			return tool.ToolResult{Content: result.String()}
		},
	}
}

