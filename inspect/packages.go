// Package inspect provides deterministic codebase inspection functionality.
package inspect

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PackageInfo represents information about a Go package.
type PackageInfo struct {
	ImportPath string
	Dir        string
	Name       string
	Files      []string
}

// ListPackages runs `go list` and returns information about packages.
func ListPackages(patterns ...string) ([]PackageInfo, error) {
	args := []string{"list", "-json"}
	if len(patterns) > 0 {
		args = append(args, patterns...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	return parseGoListOutput(output)
}

// ListPackagesInDir runs `go list` in a specific directory.
func ListPackagesInDir(dir string, patterns ...string) ([]PackageInfo, error) {
	args := []string{"list", "-json"}
	if len(patterns) > 0 {
		args = append(args, patterns...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	return parseGoListOutput(output)
}

func parseGoListOutput(output []byte) ([]PackageInfo, error) {
	var packages []PackageInfo

	// go list -json outputs multi-line JSON objects, one per package
	// We need to decode them as a stream of JSON objects
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var pkg PackageInfo
		if err := decoder.Decode(&pkg); err != nil {
			// Stop when we reach the end of input
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error parsing go list output: %w", err)
		}
		// Only add packages that have an ImportPath (skip any empty results)
		if pkg.ImportPath != "" {
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}
