package sandbox

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// IsScratchPath returns true if relPath targets a top-level tmp/ directory.
// Scratch paths are blocked for write operations to prevent the model from
// creating standalone exploration programs instead of writing proper tests.
func IsScratchPath(relPath string) bool {
	clean := filepath.Clean(relPath)
	return clean == "tmp" || strings.HasPrefix(clean, "tmp"+string(filepath.Separator))
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
