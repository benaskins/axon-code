package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunBuild(t *testing.T) {
	// Create a temporary directory with a simple Go file
	tmpDir := t.TempDir()

	// Create a simple Go module
	goMod := `module testmodule

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create a simple Go file
	goFile := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	result, err := RunBuild(tmpDir)
	if err != nil {
		t.Fatalf("RunBuild failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected build to succeed, but it failed. Stderr: %s", result.Stderr)
	}
}

func TestRunVet(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module testmodule

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	goFile := `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	result, err := RunVet(tmpDir)
	if err != nil {
		t.Fatalf("RunVet failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected vet to succeed, but it failed. Stderr: %s", result.Stderr)
	}
}

func TestRunTest(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module testmodule

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	goFile := `package main

import "testing"

func TestExample(t *testing.T) {
	if true != true {
		t.Error("This should not fail")
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(goFile), 0644); err != nil {
		t.Fatalf("Failed to write main_test.go: %v", err)
	}

	result, err := RunTest(tmpDir)
	if err != nil {
		t.Fatalf("RunTest failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected test to succeed, but it failed. Stderr: %s", result.Stderr)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		expected int
	}{
		{
			name:     "empty",
			stderr:   "",
			expected: 0,
		},
		{
			name:     "single error",
			stderr:   "main.go:10:5: undefined: foo\n",
			expected: 1,
		},
		{
			name:     "multiple errors",
			stderr:   "main.go:10:5: undefined: foo\nutil.go:20:3: undeclared variable\n",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseErrors(tt.stderr)
			if len(errors) != tt.expected {
				t.Errorf("Expected %d errors, got %d", tt.expected, len(errors))
			}
		})
	}
}
