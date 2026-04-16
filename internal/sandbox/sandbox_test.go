package sandbox_test

import (
	"testing"

	"github.com/benaskins/axon-code/internal/sandbox"
)

func TestIsScratchPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"tmp/explore.go", true},
		{"tmp/main.go", true},
		{"tmp/nested/file.go", true},
		{"internal/pkg/foo.go", false},
		{"main.go", false},
		{"cmd/app/main.go", false},
		{"tmp_cost_discover.go", true},       // tmp_*.go scratch file at root
		{"tmp_explore.go", true},             // tmp_*.go scratch file
		{"internal/tmp_helper.go", true},     // tmp_*.go in subdirectory
		{"tmpfile.go", false},                // "tmp" prefix without underscore
		{"tmp_notes.txt", false},             // tmp_ but not .go
		{"internal/tmp/ok.go", false},        // tmp nested under another dir is fine
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := sandbox.IsScratchPath(tc.path)
			if got != tc.want {
				t.Errorf("IsScratchPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	root := "/projects/myapp"

	tests := []struct {
		name    string
		relPath string
		wantErr bool
		want    string
	}{
		{
			name:    "normal relative path",
			relPath: "main.go",
			want:    "/projects/myapp/main.go",
		},
		{
			name:    "nested relative path",
			relPath: "internal/pkg/foo.go",
			want:    "/projects/myapp/internal/pkg/foo.go",
		},
		{
			name:    "path with dot segments that stay inside root",
			relPath: "internal/../main.go",
			want:    "/projects/myapp/main.go",
		},
		{
			name:    "traversal escaping root",
			relPath: "../secret/file.go",
			wantErr: true,
		},
		{
			name:    "deep traversal escaping root",
			relPath: "internal/../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "absolute path input",
			relPath: "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "empty path",
			relPath: "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sandbox.Resolve(root, tc.relPath)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Resolve(%q, %q) = %q, want error", root, tc.relPath, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q, %q) unexpected error: %v", root, tc.relPath, err)
			}
			if got != tc.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", root, tc.relPath, got, tc.want)
			}
		})
	}
}
