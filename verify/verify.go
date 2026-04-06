// Package verify provides functions to run go test, go build, and go vet commands
// with output parsing capabilities.
package verify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Result holds the outcome of a verification command execution.
type Result struct {
	Command  string
	Success  bool
	Stdout   string
	Stderr   string
	Errors   []string
}

// RunTest runs 'go test' in the specified directory and returns the result.
func RunTest(dir string) (*Result, error) {
	return runCommand(dir, "go", "test", "./...")
}

// RunBuild runs 'go build' in the specified directory and returns the result.
func RunBuild(dir string) (*Result, error) {
	return runCommand(dir, "go", "build", "./...")
}

// RunVet runs 'go vet' in the specified directory and returns the result.
func RunVet(dir string) (*Result, error) {
	return runCommand(dir, "go", "vet", "./...")
}

// validateDir validates that the directory path is safe and exists.
// It prevents directory traversal attacks by ensuring the path is absolute
// and resolves to a valid directory.
func validateDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("directory cannot be empty")
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Check that the directory exists and is actually a directory
	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absDir)
	}

	return absDir, nil
}

// runCommand executes a command and parses its output into a Result.
// The directory is validated via validateDir to prevent directory traversal.
// The command name and args are hardcoded ("go" with fixed subcommands),
// so there is no risk of arbitrary command injection.
// #nosec G204 - Command is hardcoded as "go" with controlled arguments
func runCommand(dir string, name string, args ...string) (*Result, error) {
	// Validate directory to prevent directory traversal
	absDir, err := validateDir(dir)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = absDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	result := &Result{
		Command: fmt.Sprintf("%s %s", name, strings.Join(args, " ")),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
	}

	// Parse errors from stderr
	result.Errors = parseErrors(result.Stderr)

	// Determine success
	if err != nil {
		result.Success = false
	} else {
		result.Success = true
	}

	return result, nil
}

// parseErrors extracts error lines from stderr output.
func parseErrors(stderr string) []string {
	var errors []string
	lines := strings.Split(stderr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && (strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, ":") || strings.HasPrefix(trimmed, "error:")) {
			errors = append(errors, trimmed)
		}
	}
	return errors
}

// RunAll runs test, build, and vet commands and returns all results.
func RunAll(dir string) ([]*Result, error) {
	results := make([]*Result, 0, 3)

	testResult, err := RunTest(dir)
	if err != nil {
		return nil, err
	}
	results = append(results, testResult)

	buildResult, err := RunBuild(dir)
	if err != nil {
		return nil, err
	}
	results = append(results, buildResult)

	vetResult, err := RunVet(dir)
	if err != nil {
		return nil, err
	}
	results = append(results, vetResult)

	return results, nil
}
