package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// IsScratchPath returns true if relPath targets a scratch file or directory.
// Scratch paths are blocked for write operations to prevent the model from
// creating standalone exploration programs instead of writing proper tests.
//
// Blocked patterns:
//   - tmp/ directory and anything under it
//   - tmp_*.go files at any level (standalone scratch programs)
func IsScratchPath(relPath string) bool {
	clean := filepath.Clean(relPath)
	// Block tmp/ directory.
	if clean == "tmp" || strings.HasPrefix(clean, "tmp"+string(filepath.Separator)) {
		return true
	}
	// Block tmp_*.go files (standalone scratch programs).
	base := filepath.Base(clean)
	if strings.HasPrefix(base, "tmp_") && strings.HasSuffix(base, ".go") {
		return true
	}
	return false
}

// Resolve resolves relPath relative to root, returning the absolute path.
// It returns an error if relPath is empty, is absolute, or escapes root via traversal.
func Resolve(root, relPath string) (string, error) {
	if relPath == "" {
		return "", errors.New("sandbox: path must not be empty")
	}
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("sandbox: absolute path not allowed: %s", relPath)
	}

	joined := filepath.Join(root, relPath)
	abs := filepath.Clean(joined)

	// Ensure the resolved path is within root.
	// Add a trailing separator to root to prevent /projects/myapp-evil from matching /projects/myapp.
	rootWithSep := filepath.Clean(root) + string(filepath.Separator)
	if abs != filepath.Clean(root) && !strings.HasPrefix(abs, rootWithSep) {
		return "", fmt.Errorf("sandbox: path escapes root: %s", relPath)
	}

	return abs, nil
}
