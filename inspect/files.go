// Package inspect provides deterministic codebase inspection functionality.
package inspect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath ensures that a path is within the allowed base directory.
// This prevents directory traversal attacks by resolving the path and checking
// that it starts with the base directory prefix.
// If the path is absolute, it validates that it's within the base directory.
// If the path is relative, it joins it with the base directory first.
func validatePath(baseDir, path string) (string, error) {
	var fullPath string

	// Check if path is absolute
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		// Join base directory with the requested path
		fullPath = filepath.Join(baseDir, path)
	}

	// Resolve to absolute path (handles .., symlinks, etc.)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Resolve base directory to absolute path
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}

	// Ensure the resolved path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("path escapes base directory: %s", path)
	}

	return absPath, nil
}

// ReadSourceFile reads a Go source file and returns its contents.
// The path is validated to ensure it's within the base directory.
// #nosec G304 - Path is validated via validatePath to ensure it's within baseDir
func ReadSourceFile(baseDir, path string) (string, error) {
	// Validate path to prevent directory traversal
	absPath, err := validatePath(baseDir, path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(content), nil
}

// ReadSourceFiles reads multiple Go source files and returns their contents.
// Paths are validated to ensure they're within the base directory.
func ReadSourceFiles(baseDir string, paths []string) (map[string]string, error) {
	contents := make(map[string]string)

	for _, path := range paths {
		content, err := ReadSourceFile(baseDir, path)
		if err != nil {
			return nil, err
		}
		contents[path] = content
	}

	return contents, nil
}

// GetSourceFiles returns a list of Go source files in a directory (excluding test files).
func GetSourceFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip hidden directories and vendor
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	return files, nil
}

// GetAllSourceFiles returns all Go source files (including test files) in a directory.
func GetAllSourceFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	return files, nil
}
