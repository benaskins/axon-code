package sandbox_test

import (
	"testing"

	"github.com/benaskins/axon-code/internal/sandbox"
)

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
