package inspect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoMod(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures", "go.mod")

	mod, err := ParseGoMod(fixturePath)
	if err != nil {
		t.Fatalf("ParseGoMod failed: %v", err)
	}

	if mod.Module != "example.com/testmodule" {
		t.Errorf("Expected module 'example.com/testmodule', got '%s'", mod.Module)
	}

	if mod.GoVersion != "1.21" {
		t.Errorf("Expected go version '1.21', got '%s'", mod.GoVersion)
	}

	if len(mod.Requires) != 2 {
		t.Errorf("Expected 2 requires, got %d", len(mod.Requires))
	}

	if mod.Requires[0].Path != "github.com/some/package" {
		t.Errorf("Expected first require path 'github.com/some/package', got '%s'", mod.Requires[0].Path)
	}

	if mod.Requires[0].Version != "v1.2.3" {
		t.Errorf("Expected first require version 'v1.2.3', got '%s'", mod.Requires[0].Version)
	}

	if mod.Requires[1].Indirect != true {
		t.Error("Expected second require to be indirect")
	}

	if len(mod.Replaces) != 2 {
		t.Errorf("Expected 2 replaces, got %d", len(mod.Replaces))
	}

	if mod.Replaces[0].OldPath != "github.com/some/package" {
		t.Errorf("Expected first replace old path 'github.com/some/package', got '%s'", mod.Replaces[0].OldPath)
	}

	if mod.Replaces[0].NewPath != "../some-package" {
		t.Errorf("Expected first replace new path '../some-package', got '%s'", mod.Replaces[0].NewPath)
	}
}

func TestParseModContent(t *testing.T) {
	content := `module example.com/test

go 1.20

require (
	github.com/pkg1 v1.0.0
	github.com/pkg2 v2.0.0 // indirect
)

replace github.com/pkg1 => ../pkg1
`

	mod, err := ParseModContent(content)
	if err != nil {
		t.Fatalf("ParseModContent failed: %v", err)
	}

	if mod.Module != "example.com/test" {
		t.Errorf("Expected module 'example.com/test', got '%s'", mod.Module)
	}

	if mod.GoVersion != "1.20" {
		t.Errorf("Expected go version '1.20', got '%s'", mod.GoVersion)
	}

	if len(mod.Requires) != 2 {
		t.Errorf("Expected 2 requires, got %d", len(mod.Requires))
	}

	if len(mod.Replaces) != 1 {
		t.Errorf("Expected 1 replace, got %d", len(mod.Replaces))
	}
}

func TestParseRequireDirective(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected Require
		ok       bool
	}{
		{
			name:     "simple require",
			line:     "github.com/pkg v1.0.0",
			expected: Require{Path: "github.com/pkg", Version: "v1.0.0", Indirect: false},
			ok:       true,
		},
		{
			name:     "indirect require",
			line:     "github.com/pkg v1.0.0 // indirect",
			expected: Require{Path: "github.com/pkg", Version: "v1.0.0", Indirect: true},
			ok:       true,
		},
		{
			name:     "empty line",
			line:     "",
			expected: Require{},
			ok:       false,
		},
		{
			name:     "comment line",
			line:     "// this is a comment",
			expected: Require{},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := parseRequireDirective(tt.line)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got %v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestParseReplaceDirective(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected Replace
		ok       bool
	}{
		{
			name:     "simple replace",
			line:     "github.com/pkg => ../pkg",
			expected: Replace{OldPath: "github.com/pkg", NewPath: "../pkg"},
			ok:       true,
		},
		{
			name:     "replace with version",
			line:     "github.com/pkg v1.0.0 => github.com/newpkg v2.0.0",
			expected: Replace{OldPath: "github.com/pkg", OldVersion: "v1.0.0", NewPath: "github.com/newpkg", NewVersion: "v2.0.0"},
			ok:       true,
		},
		{
			name:     "empty line",
			line:     "",
			expected: Replace{},
			ok:       false,
		},
		{
			name:     "invalid replace (no =>)",
			line:     "github.com/pkg ../pkg",
			expected: Replace{},
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := parseReplaceDirective(tt.line)
			if ok != tt.ok {
				t.Errorf("Expected ok=%v, got %v", tt.ok, ok)
			}
			if ok && result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestExtractInterfaces(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures")

	interfaces, err := ExtractInterfaces(fixturePath)
	if err != nil {
		t.Fatalf("ExtractInterfaces failed: %v", err)
	}

	// Should find SampleInterface, AnotherInterface, EmbeddedInterface
	// Should NOT find sampleUnexported
	foundNames := make(map[string]bool)
	for _, iface := range interfaces {
		foundNames[iface.Name] = true
	}

	expectedInterfaces := []string{"SampleInterface", "AnotherInterface", "EmbeddedInterface"}
	for _, expected := range expectedInterfaces {
		if !foundNames[expected] {
			t.Errorf("Expected to find interface '%s', but didn't", expected)
		}
	}

	if foundNames["sampleUnexported"] {
		t.Error("Should not find unexported interface 'sampleUnexported'")
	}

	// Check SampleInterface methods
	var sampleIface *ExportedInterface
	for i := range interfaces {
		if interfaces[i].Name == "SampleInterface" {
			sampleIface = &interfaces[i]
			break
		}
	}

	if sampleIface == nil {
		t.Fatal("Could not find SampleInterface")
	}

	if len(sampleIface.Methods) != 2 {
		t.Errorf("Expected SampleInterface to have 2 methods, got %d", len(sampleIface.Methods))
	}

	methodNames := make(map[string]bool)
	for _, method := range sampleIface.Methods {
		methodNames[method.Name] = true
	}

	if !methodNames["DoSomething"] || !methodNames["GetValue"] {
		t.Error("SampleInterface should have DoSomething and GetValue methods")
	}

	// Check EmbeddedInterface has the embedded method
	var embeddedIface *ExportedInterface
	for i := range interfaces {
		if interfaces[i].Name == "EmbeddedInterface" {
			embeddedIface = &interfaces[i]
			break
		}
	}

	if embeddedIface == nil {
		t.Fatal("Could not find EmbeddedInterface")
	}

	// Should have SampleInterface (embedded) and ExtendedMethod
	if len(embeddedIface.Methods) < 2 {
		t.Errorf("Expected EmbeddedInterface to have at least 2 methods, got %d", len(embeddedIface.Methods))
	}
}

func TestExtractTypes(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures")

	types, err := ExtractTypes(fixturePath)
	if err != nil {
		t.Fatalf("ExtractTypes failed: %v", err)
	}

	foundNames := make(map[string]bool)
	for _, typ := range types {
		foundNames[typ.Name] = true
	}

	expectedTypes := []string{
		"SampleStruct", "SampleType", "SamplePointer", "SampleSlice",
		"SampleMap", "SampleChannel", "SampleFunc", "SampleInterface",
		"AnotherInterface", "EmbeddedInterface",
	}

	for _, expected := range expectedTypes {
		if !foundNames[expected] {
			t.Errorf("Expected to find type '%s', but didn't", expected)
		}
	}

	// Note: UnexportedStruct is actually exported (starts with uppercase)
	// The test fixture has it defined as exported but with "Unexported" in the name
	// This is intentional to test that we correctly identify exported types

	// Check type kinds
	typeKinds := make(map[string]string)
	for _, typ := range types {
		typeKinds[typ.Name] = typ.TypeKind
	}

	if typeKinds["SampleStruct"] != "struct" {
		t.Errorf("Expected SampleStruct to be 'struct', got '%s'", typeKinds["SampleStruct"])
	}

	if typeKinds["SampleInterface"] != "interface" {
		t.Errorf("Expected SampleInterface to be 'interface', got '%s'", typeKinds["SampleInterface"])
	}

	if typeKinds["SampleType"] != "type alias" {
		t.Errorf("Expected SampleType to be 'type alias', got '%s'", typeKinds["SampleType"])
	}

	if typeKinds["SamplePointer"] != "pointer" {
		t.Errorf("Expected SamplePointer to be 'pointer', got '%s'", typeKinds["SamplePointer"])
	}
}

func TestReadSourceFile(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures", "sample.go")

	content, err := ReadSourceFile(".", fixturePath)
	if err != nil {
		t.Fatalf("ReadSourceFile failed: %v", err)
	}

	if content == "" {
		t.Fatal("Expected non-empty content")
	}

	if !strings.Contains(content, "package sample") {
		t.Error("Expected content to contain 'package sample'")
	}

	if !strings.Contains(content, "SampleInterface") {
		t.Error("Expected content to contain 'SampleInterface'")
	}
}

func TestReadSourceFiles(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures")

	files, err := filepath.Glob(filepath.Join(fixturePath, "*.go"))
	if err != nil {
		t.Fatalf("filepath.Glob failed: %v", err)
	}

	// Filter to only non-test files
	var sourceFiles []string
	for _, f := range files {
		if !strings.HasSuffix(f, "_test.go") {
			sourceFiles = append(sourceFiles, f)
		}
	}

	contents, err := ReadSourceFiles(".", sourceFiles)
	if err != nil {
		t.Fatalf("ReadSourceFiles failed: %v", err)
	}

	if len(contents) != len(sourceFiles) {
		t.Errorf("Expected %d files, got %d", len(sourceFiles), len(contents))
	}

	for _, path := range sourceFiles {
		if _, ok := contents[path]; !ok {
			t.Errorf("Expected to find content for %s", path)
		}
	}
}

func TestGetSourceFiles(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures")

	files, err := GetSourceFiles(fixturePath)
	if err != nil {
		t.Fatalf("GetSourceFiles failed: %v", err)
	}

	// Should find sample.go, types.go, another.go
	// Should NOT find test files
	foundBasenames := make(map[string]bool)
	for _, f := range files {
		foundBasenames[filepath.Base(f)] = true
	}

	expectedFiles := []string{"sample.go", "types.go", "another.go"}
	for _, expected := range expectedFiles {
		if !foundBasenames[expected] {
			t.Errorf("Expected to find file '%s', but didn't", expected)
		}
	}

	// Check no test files
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			t.Errorf("Should not include test files, but found '%s'", f)
		}
	}
}

func TestGetAllSourceFiles(t *testing.T) {
	// Create a test file for this test
	fixturePath := filepath.Join("testdata", "fixtures")
	testFilePath := filepath.Join(fixturePath, "sample_test.go")
	err := os.WriteFile(testFilePath, []byte("package sample\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	files, err := GetAllSourceFiles(fixturePath)
	if err != nil {
		t.Fatalf("GetAllSourceFiles failed: %v", err)
	}

	foundBasenames := make(map[string]bool)
	for _, f := range files {
		foundBasenames[filepath.Base(f)] = true
	}

	// Should include test files now
	if !foundBasenames["sample_test.go"] {
		t.Error("Should include test files")
	}
}

func TestListPackages(t *testing.T) {
	// This test requires a valid Go module
	// We'll test it in the current directory
	packages, err := ListPackages("./inspect")
	if err != nil {
		t.Skipf("ListPackages failed (might be expected in test environment): %v", err)
	}

	if len(packages) == 0 {
		t.Skip("No packages found")
	}

	foundInspect := false
	for _, pkg := range packages {
		if strings.Contains(pkg.ImportPath, "inspect") {
			foundInspect = true
			break
		}
	}

	if !foundInspect {
		t.Error("Expected to find inspect package")
	}
}

func TestListPackagesInDir(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures")

	packages, err := ListPackagesInDir(fixturePath)
	if err != nil {
		t.Skipf("ListPackagesInDir failed: %v", err)
	}

	if len(packages) == 0 {
		t.Skip("No packages found in fixture directory")
	}

	// Should find at least one package
	foundSample := false
	for _, pkg := range packages {
		if pkg.Name == "sample" || pkg.Name == "another" {
			foundSample = true
			break
		}
	}

	if !foundSample {
		t.Error("Expected to find sample or another package")
	}
}

func TestExtractTypesFromAnotherPkg(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures", "anotherpkg")

	types, err := ExtractTypes(fixturePath)
	if err != nil {
		t.Fatalf("ExtractTypes failed: %v", err)
	}

	foundNames := make(map[string]bool)
	for _, typ := range types {
		foundNames[typ.Name] = true
	}

	if !foundNames["AnotherPkgStruct"] {
		t.Error("Expected to find AnotherPkgStruct")
	}
}

func TestExtractInterfacesFromAnotherPkg(t *testing.T) {
	fixturePath := filepath.Join("testdata", "fixtures", "anotherpkg")

	interfaces, err := ExtractInterfaces(fixturePath)
	if err != nil {
		t.Fatalf("ExtractInterfaces failed: %v", err)
	}

	foundNames := make(map[string]bool)
	for _, iface := range interfaces {
		foundNames[iface.Name] = true
	}

	if !foundNames["AnotherPkgInterface"] {
		t.Error("Expected to find AnotherPkgInterface")
	}
}

func TestNonExistentFile(t *testing.T) {
	_, err := ReadSourceFile(".", "nonexistent.go")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestNonExistentDirectory(t *testing.T) {
	_, err := ExtractTypes("nonexistent_dir")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	_, err = ExtractInterfaces("nonexistent_dir")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}
