package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/inspect"
	"github.com/benaskins/axon-code/internal/sandbox"
)

// DomainTools holds project-level domain tools.
type DomainTools struct {
	SummariseProjectTool tool.ToolDef
	InspectProjectTool   tool.ToolDef
}

// NewDomainTools constructs domain tools bound to projectDir.
func NewDomainTools(projectDir string) DomainTools {
	return DomainTools{
		SummariseProjectTool: makeSummariseProjectTool(projectDir),
		InspectProjectTool:   makeInspectProjectTool(projectDir),
	}
}

// makeSummariseProjectTool gives the agent a complete project snapshot in one call:
// directory tree, go.mod, CLAUDE.md, package list, git status, and build/test status.
// This replaces the 6-8 orientation tool calls the agent would otherwise make.
func makeSummariseProjectTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "summarise_project",
		Description: "Get a complete project snapshot: directory tree, go.mod, CLAUDE.md, packages, git status, and build/test results. Call this FIRST before any other tool to understand the project.",
		Parameters: tool.ParameterSchema{
			Type:       "object",
			Properties: map[string]tool.PropertySchema{},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			var b strings.Builder

			// 1. Directory tree (2 levels deep)
			fmt.Fprintf(&b, "## Directory Tree\n\n")
			tree, err := buildTree(projectDir, "", 2)
			if err != nil {
				fmt.Fprintf(&b, "error: %v\n", err)
			} else {
				b.WriteString(tree)
			}
			b.WriteString("\n")

			// 2. go.mod
			fmt.Fprintf(&b, "## go.mod\n\n")
			modPath := filepath.Join(projectDir, "go.mod")
			modBytes, err := os.ReadFile(modPath)
			if err != nil {
				fmt.Fprintf(&b, "not found\n")
			} else {
				fmt.Fprintf(&b, "```\n%s```\n", string(modBytes))
			}
			b.WriteString("\n")

			// 3. CLAUDE.md (project instructions)
			fmt.Fprintf(&b, "## CLAUDE.md\n\n")
			claudePath := filepath.Join(projectDir, "CLAUDE.md")
			claudeBytes, err := os.ReadFile(claudePath)
			if err != nil {
				fmt.Fprintf(&b, "not found\n")
			} else {
				content := string(claudeBytes)
				if len(content) > 2000 {
					content = content[:2000] + "\n... (truncated)"
				}
				b.WriteString(content)
			}
			b.WriteString("\n\n")

			// 4. Package list
			fmt.Fprintf(&b, "## Packages\n\n")
			pkgs, err := inspect.ListPackagesInDir(projectDir, "./...")
			if err != nil {
				fmt.Fprintf(&b, "error: %v\n", err)
			} else {
				for _, pkg := range pkgs {
					fmt.Fprintf(&b, "- %s (%d files)\n", pkg.ImportPath, len(pkg.Files))
				}
			}
			b.WriteString("\n")

			// 5. Git status
			fmt.Fprintf(&b, "## Git Status\n\n")
			gitStatus := exec.Command("git", "status", "--short")
			gitStatus.Dir = projectDir
			statusOut, err := gitStatus.CombinedOutput()
			if err != nil {
				fmt.Fprintf(&b, "not a git repo or error: %v\n", err)
			} else if len(statusOut) == 0 {
				b.WriteString("clean\n")
			} else {
				fmt.Fprintf(&b, "```\n%s```\n", string(statusOut))
			}
			b.WriteString("\n")

			// 6. Build + test status
			fmt.Fprintf(&b, "## Build & Test\n\n")
			buildCmd := exec.Command("go", "build", "./...")
			buildCmd.Dir = projectDir
			buildOut, err := buildCmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(&b, "BUILD FAILED:\n```\n%s```\n", string(buildOut))
			} else {
				b.WriteString("build: PASS\n")
			}

			testCmd := exec.Command("go", "test", "./...")
			testCmd.Dir = projectDir
			testOut, err := testCmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(&b, "TEST FAILED:\n```\n%s```\n", string(testOut))
			} else {
				fmt.Fprintf(&b, "test: PASS\n```\n%s```\n", string(testOut))
			}

			return tool.ToolResult{Content: b.String()}
		},
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

// buildTree returns a directory listing with indentation, up to maxDepth levels.
func buildTree(dir, prefix string, maxDepth int) (string, error) {
	if maxDepth <= 0 {
		return "", nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if name == ".git" || name == "vendor" || name == "node_modules" {
			continue
		}
		if e.IsDir() {
			fmt.Fprintf(&b, "%s%s/\n", prefix, name)
			sub, _ := buildTree(filepath.Join(dir, name), prefix+"  ", maxDepth-1)
			b.WriteString(sub)
		} else {
			fmt.Fprintf(&b, "%s%s\n", prefix, name)
		}
	}
	return b.String(), nil
}
