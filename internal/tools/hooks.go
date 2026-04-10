package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FileHook runs after a file is successfully written or edited.
// Pattern is a glob matched against filepath.Base(relPath).
// Run receives the absolute path and project directory.
// Returns output to append to the tool result ("" = silent).
type FileHook struct {
	Pattern string
	Run     func(absPath, projectDir string) string
}

// CmdHook runs after go_cmd produces a non-zero exit code.
// Name is the key used in CmdResult.Hooks.
// Match decides whether this hook applies.
// Run returns output to add to the hooks map.
type CmdHook struct {
	Name  string
	Match func(subcommand string, exitCode int) bool
	Run   func(projectDir string) string
}

// HookRegistry holds file and command hooks.
type HookRegistry struct {
	FileHooks []FileHook
	CmdHooks  []CmdHook
}

// RunFileHooks runs all file hooks matching relPath and returns aggregated output.
// Nil-safe: returns "" if hr is nil.
func (hr *HookRegistry) RunFileHooks(relPath, absPath, projectDir string) string {
	if hr == nil {
		return ""
	}
	var parts []string
	base := filepath.Base(relPath)
	for _, h := range hr.FileHooks {
		matched, _ := filepath.Match(h.Pattern, base)
		if !matched {
			continue
		}
		out := safeRunFileHook(h, absPath, projectDir)
		if out != "" {
			parts = append(parts, out)
		}
	}
	return strings.Join(parts, "\n")
}

// RunCmdHooks runs all matching command hooks and returns a map of name->output.
// Nil-safe: returns nil if hr is nil or no hooks matched.
func (hr *HookRegistry) RunCmdHooks(subcommand string, exitCode int, projectDir string) map[string]string {
	if hr == nil {
		return nil
	}
	var result map[string]string
	for _, h := range hr.CmdHooks {
		if !h.Match(subcommand, exitCode) {
			continue
		}
		out := safeRunCmdHook(h, projectDir)
		if out != "" {
			if result == nil {
				result = make(map[string]string)
			}
			result[h.Name] = out
		}
	}
	return result
}

// NewDefaultHookRegistry creates a HookRegistry with the standard hooks:
// - *.go files: gofmt -w (silent on success, reports syntax errors)
// - go.mod: go mod tidy (silent on success, reports errors)
// - build/test/vet failure: gopls check ./... (if gopls on PATH)
func NewDefaultHookRegistry(projectDir string) *HookRegistry {
	hr := &HookRegistry{}

	// gofmt on *.go files
	hr.FileHooks = append(hr.FileHooks, FileHook{
		Pattern: "*.go",
		Run: func(absPath, projectDir string) string {
			cmd := exec.Command("gofmt", "-w", absPath)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return fmt.Sprintf("gofmt: %s", strings.TrimSpace(stderr.String()))
			}
			return ""
		},
	})

	// go mod tidy on go.mod
	hr.FileHooks = append(hr.FileHooks, FileHook{
		Pattern: "go.mod",
		Run: func(absPath, projectDir string) string {
			cmd := exec.Command("go", "mod", "tidy")
			cmd.Dir = projectDir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("go mod tidy: %s", strings.TrimSpace(string(out)))
			}
			return ""
		},
	})

	// gopls diagnostics on build/test/vet failure
	if _, err := exec.LookPath("gopls"); err == nil {
		hr.CmdHooks = append(hr.CmdHooks, CmdHook{
			Name: "diagnostics",
			Match: func(subcommand string, exitCode int) bool {
				return exitCode != 0 && (subcommand == "build" || subcommand == "test" || subcommand == "vet")
			},
			Run: func(projectDir string) string {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, "gopls", "check", "-severity=error", "./...")
				cmd.Dir = projectDir
				out, _ := cmd.CombinedOutput()
				return strings.TrimSpace(string(out))
			},
		})
	}

	return hr
}

func safeRunFileHook(h FileHook, absPath, projectDir string) (output string) {
	defer func() {
		if r := recover(); r != nil {
			output = fmt.Sprintf("hook panic: %v", r)
		}
	}()
	return h.Run(absPath, projectDir)
}

func safeRunCmdHook(h CmdHook, projectDir string) (output string) {
	defer func() {
		if r := recover(); r != nil {
			output = fmt.Sprintf("hook panic: %v", r)
		}
	}()
	return h.Run(projectDir)
}
