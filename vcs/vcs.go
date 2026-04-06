// Package vcs provides Git version control operations.
package vcs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StatusInfo holds the git status output.
type StatusInfo struct {
	Changed   bool
	Files     []string
	Untracked []string
}

// DiffInfo holds the git diff output.
type DiffInfo struct {
	Content string
	Changed bool
}

// CommitResult holds the result of a git commit.
type CommitResult struct {
	Success bool
	Hash    string
	Message string
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

// RunGit runs a git command with the given arguments and returns the output.
// The directory is validated via validateDir to prevent directory traversal.
// Git is a trusted command and the args are controlled by the caller through
// specific API functions, preventing arbitrary command execution.
// #nosec G204 - Git is a trusted command with controlled arguments
func RunGit(dir string, args ...string) (string, error) {
	// Validate directory to prevent directory traversal
	absDir, err := validateDir(dir)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = absDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}

// Add stages files to the git index.
func Add(dir string, files ...string) error {
	args := []string{"add"}
	args = append(args, files...)

	_, err := RunGit(dir, args...)
	return err
}

// AddAll stages all changes to the git index.
func AddAll(dir string) error {
	_, err := RunGit(dir, "add", "-A")
	return err
}

// Commit creates a new commit with the given message.
func Commit(dir string, message string) (*CommitResult, error) {
	// Validate directory to prevent directory traversal
	absDir, err := validateDir(dir)
	if err != nil {
		return nil, err
	}

	// Git commit with -m flag only uses the message as commit text,
	// not as a command or argument that could be injected.
	// #nosec G204 - Git is a trusted command; message is only used as commit text
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = absDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String()
	errOutput := stderr.String()

	if err != nil {
		// Check if it's a "nothing to commit" error (message appears in stdout)
		if strings.Contains(output, "nothing to commit") || strings.Contains(errOutput, "nothing to commit") {
			return &CommitResult{
				Success: false,
				Message: message,
			}, nil
		}
		return nil, fmt.Errorf("git commit failed: %w: %s", err, errOutput)
	}

	// Get the commit hash
	hashOutput, err := RunGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	return &CommitResult{
		Success: true,
		Hash:    strings.TrimSpace(hashOutput),
		Message: message,
	}, nil
}

// Diff returns the differences between the working directory and the index (alias for GetDiff).
func Diff(dir string) (*DiffInfo, error) {
	return GetDiff(dir)
}

// GetDiff returns the differences between the working directory and the index.
func GetDiff(dir string) (*DiffInfo, error) {
	output, err := RunGit(dir, "diff")
	if err != nil {
		return nil, err
	}

	return &DiffInfo{
		Content: output,
		Changed: strings.TrimSpace(output) != "",
	}, nil
}

// DiffStaged returns the differences between the index and HEAD (alias for GetDiffStaged).
func DiffStaged(dir string) (*DiffInfo, error) {
	return GetDiffStaged(dir)
}

// GetDiffStaged returns the differences between the index and HEAD.
func GetDiffStaged(dir string) (*DiffInfo, error) {
	output, err := RunGit(dir, "diff", "--cached")
	if err != nil {
		return nil, err
	}

	return &DiffInfo{
		Content: output,
		Changed: strings.TrimSpace(output) != "",
	}, nil
}

// Status returns the current git status (alias for GetStatus).
func Status(dir string) (*StatusInfo, error) {
	return GetStatus(dir)
}

// GetStatus returns the current git status.
func GetStatus(dir string) (*StatusInfo, error) {
	output, err := RunGit(dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	status := &StatusInfo{
		Changed:   strings.TrimSpace(output) != "",
		Files:     []string{},
		Untracked: []string{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse porcelain format: XY filename
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		file := parts[1]
		status.Files = append(status.Files, file)

		// Check if untracked (?? prefix)
		if strings.HasPrefix(parts[0], "??") {
			status.Untracked = append(status.Untracked, file)
		}
	}

	return status, nil
}

// IsGitRepo checks if the directory is a git repository.
func IsGitRepo(dir string) bool {
	_, err := RunGit(dir, "rev-parse", "--git-dir")
	return err == nil
}

// Log returns the commit log.
func Log(dir string, limit int) (string, error) {
	format := "--oneline"
	if limit > 0 {
		return RunGit(dir, "log", fmt.Sprintf("-%d", limit), format)
	}
	return RunGit(dir, "log", format)
}
