// Package inspect provides deterministic codebase inspection functionality.
package inspect

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ModFile represents a parsed go.mod file.
type ModFile struct {
	Module    string
	GoVersion string
	Requires  []Require
	Replaces  []Replace
}

// Require represents a require directive in go.mod.
type Require struct {
	Path     string
	Version  string
	Indirect bool
}

// Replace represents a replace directive in go.mod.
type Replace struct {
	OldPath    string
	OldVersion string
	NewPath    string
	NewVersion string
}

// ParseGoMod parses a go.mod file and returns its contents as a ModFile struct.
// The path is validated to ensure it's within the base directory.
// #nosec G304 - Path is validated via validatePath to ensure it's within baseDir
func ParseGoMod(path string) (*ModFile, error) {
	// Validate path to prevent directory traversal
	absPath, err := validatePath(".", path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	return ParseModContent(string(content))
}

// ParseModContent parses go.mod content from a string.
func ParseModContent(content string) (*ModFile, error) {
	mod := &ModFile{
		Requires: make([]Require, 0),
		Replaces: make([]Replace, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	inRequireBlock := false
	inReplaceBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Module declaration
		if strings.HasPrefix(line, "module ") {
			mod.Module = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			continue
		}

		// Go version
		if strings.HasPrefix(line, "go ") {
			mod.GoVersion = strings.TrimSpace(strings.TrimPrefix(line, "go "))
			continue
		}

		// Start of require block
		if line == "require (" {
			inRequireBlock = true
			continue
		}

		// End of require block
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// Start of replace block
		if line == "replace (" {
			inReplaceBlock = true
			continue
		}

		// End of replace block
		if inReplaceBlock && line == ")" {
			inReplaceBlock = false
			continue
		}

		// Parse require directive
		if inRequireBlock || strings.HasPrefix(line, "require ") {
			reqLine := line
			if inRequireBlock {
				reqLine = strings.TrimSpace(line)
			} else {
				reqLine = strings.TrimSpace(strings.TrimPrefix(line, "require "))
			}

			req, ok := parseRequireDirective(reqLine)
			if ok {
				mod.Requires = append(mod.Requires, req)
			}
			continue
		}

		// Parse replace directive
		if inReplaceBlock || strings.HasPrefix(line, "replace ") {
			repLine := line
			if inReplaceBlock {
				repLine = strings.TrimSpace(line)
			} else {
				repLine = strings.TrimSpace(strings.TrimPrefix(line, "replace "))
			}

			rep, ok := parseReplaceDirective(repLine)
			if ok {
				mod.Replaces = append(mod.Replaces, rep)
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning go.mod: %w", err)
	}

	return mod, nil
}

func parseRequireDirective(line string) (Require, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return Require{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return Require{}, false
	}

	req := Require{
		Path:     parts[0],
		Version:  parts[1],
		Indirect: false,
	}

	// Check for indirect comment
	for i, part := range parts {
		if part == "//" && i+1 < len(parts) && parts[i+1] == "indirect" {
			req.Indirect = true
			break
		}
	}

	return req, true
}

func parseReplaceDirective(line string) (Replace, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return Replace{}, false
	}

	// Format: old/path => new/path or old/path => new/path v1.0.0
	// Or with version: old/path v1.0.0 => new/path v2.0.0
	parts := strings.Split(line, " => ")
	if len(parts) != 2 {
		return Replace{}, false
	}

	oldParts := strings.Fields(parts[0])
	newParts := strings.Fields(parts[1])

	rep := Replace{
		OldPath: oldParts[0],
		NewPath: newParts[0],
	}

	if len(oldParts) > 1 {
		rep.OldVersion = oldParts[1]
	}
	if len(newParts) > 1 {
		rep.NewVersion = newParts[1]
	}

	return rep, true
}
