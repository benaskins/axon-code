package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupGitRepo creates a temporary directory with a git repository
func setupGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user (required for commits)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git name: %v", err)
	}

	return tmpDir
}

func TestIsGitRepo(t *testing.T) {
	t.Run("valid git repo", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		if !IsGitRepo(tmpDir) {
			t.Error("Expected IsGitRepo to return true for a valid git repo")
		}
	})

	t.Run("invalid git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		if IsGitRepo(tmpDir) {
			t.Error("Expected IsGitRepo to return false for a non-git directory")
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("empty repo", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		status, err := Status(tmpDir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if status.Changed {
			t.Error("Expected no changes in empty repo")
		}
	})

	t.Run("with untracked file", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		status, err := Status(tmpDir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if !status.Changed {
			t.Error("Expected changes to be detected")
		}
		if len(status.Untracked) != 1 {
			t.Errorf("Expected 1 untracked file, got %d", len(status.Untracked))
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("add file", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		err := Add(tmpDir, "test.txt")
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		// Verify file is staged
		status, err := Status(tmpDir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if !status.Changed {
			t.Error("Expected file to be staged")
		}
	})

	t.Run("add all", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile1 := filepath.Join(tmpDir, "test1.txt")
		testFile2 := filepath.Join(tmpDir, "test2.txt")
		if err := os.WriteFile(testFile1, []byte("hello1"), 0644); err != nil {
			t.Fatalf("Failed to write test file 1: %v", err)
		}
		if err := os.WriteFile(testFile2, []byte("hello2"), 0644); err != nil {
			t.Fatalf("Failed to write test file 2: %v", err)
		}

		err := AddAll(tmpDir)
		if err != nil {
			t.Fatalf("AddAll failed: %v", err)
		}

		status, err := Status(tmpDir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if len(status.Files) != 2 {
			t.Errorf("Expected 2 staged files, got %d", len(status.Files))
		}
	})
}

func TestCommit(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Stage the file
		if err := AddAll(tmpDir); err != nil {
			t.Fatalf("AddAll failed: %v", err)
		}

		result, err := Commit(tmpDir, "Initial commit")
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if !result.Success {
			t.Error("Expected commit to succeed")
		}
		if result.Hash == "" {
			t.Error("Expected commit hash to be set")
		}
		if result.Message != "Initial commit" {
			t.Errorf("Expected message 'Initial commit', got '%s'", result.Message)
		}
	})

	t.Run("commit with nothing to commit", func(t *testing.T) {
		tmpDir := setupGitRepo(t)

		result, err := Commit(tmpDir, "Empty commit")
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if result.Success {
			t.Error("Expected commit to fail when nothing to commit")
		}
	})
}

func TestDiff(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		diff, err := Diff(tmpDir)
		if err != nil {
			t.Fatalf("Diff failed: %v", err)
		}
		if diff.Changed {
			t.Error("Expected no diff in empty repo")
		}
	})

	t.Run("with changes", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Stage and commit first
		if err := AddAll(tmpDir); err != nil {
			t.Fatalf("AddAll failed: %v", err)
		}
		if _, err := Commit(tmpDir, "Initial"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Modify file
		if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		diff, err := Diff(tmpDir)
		if err != nil {
			t.Fatalf("Diff failed: %v", err)
		}
		if !diff.Changed {
			t.Error("Expected diff to show changes")
		}
		if !strings.Contains(diff.Content, "hello world") {
			t.Error("Expected diff to contain new content")
		}
	})
}

func TestDiffStaged(t *testing.T) {
	t.Run("no staged changes", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		diff, err := DiffStaged(tmpDir)
		if err != nil {
			t.Fatalf("DiffStaged failed: %v", err)
		}
		if diff.Changed {
			t.Error("Expected no staged changes")
		}
	})

	t.Run("with staged changes", func(t *testing.T) {
		tmpDir := setupGitRepo(t)
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		if err := AddAll(tmpDir); err != nil {
			t.Fatalf("AddAll failed: %v", err)
		}

		diff, err := DiffStaged(tmpDir)
		if err != nil {
			t.Fatalf("DiffStaged failed: %v", err)
		}
		if !diff.Changed {
			t.Error("Expected staged changes to be detected")
		}
	})
}

func TestLog(t *testing.T) {
	tmpDir := setupGitRepo(t)
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := AddAll(tmpDir); err != nil {
		t.Fatalf("AddAll failed: %v", err)
	}
	if _, err := Commit(tmpDir, "Initial commit"); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	log, err := Log(tmpDir, 1)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if !strings.Contains(log, "Initial commit") {
		t.Error("Expected log to contain commit message")
	}
}
