// Package inspect provides deterministic codebase inspection functionality.
package inspect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ExportedInterface represents an exported interface in a package.
type ExportedInterface struct {
	Name    string
	Doc     string
	Methods []InterfaceMethod
}

// InterfaceMethod represents a method in an interface.
type InterfaceMethod struct {
	Name      string
	Doc       string
	Signature string
}

// ExportedType represents an exported type in a package.
type ExportedType struct {
	Name     string
	Doc      string
	TypeKind string // "struct", "interface", "type alias", etc.
}

// ExtractInterfaces extracts all exported interfaces from a package directory using go/packages.
func ExtractInterfaces(dir string) ([]ExportedInterface, error) {
	// Use go/packages to load the package
	cfg := &packages.Config{
		Mode:  packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedName,
		Fset:  token.NewFileSet(),
		Tests: false, // Don't include test files
	}

	pkgs, err := packages.Load(cfg, dir)
	if err != nil {
		// If go/packages fails (e.g., testdata directory), fall back to parser.ParseDir
		// This is a common case for test fixtures
		return extractInterfacesFallback(dir)
	}

	var interfaces []ExportedInterface

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			// If there are errors, try fallback
			return extractInterfacesFallback(dir)
		}

		// Skip test files (already filtered by Tests: false, but double-check)
		var files []*ast.File
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if !strings.HasSuffix(filename, "_test.go") {
				files = append(files, file)
			}
		}

		if len(files) == 0 {
			continue
		}

		// Extract interfaces using type info from go/packages
		for _, file := range files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if !ast.IsExported(typeSpec.Name.Name) {
						continue
					}

					// Check if it's an interface by looking at the AST
					if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						// Get type info from type checker
						obj := pkg.TypesInfo.Defs[typeSpec.Name]
						var methods []InterfaceMethod

						if obj != nil {
							// For named types, we need to get the underlying type
							if named, ok := obj.Type().(*types.Named); ok {
								if iface, ok := named.Underlying().(*types.Interface); ok {
									for i := 0; i < iface.NumMethods(); i++ {
										method := iface.Method(i)
										if method.Exported() {
											methods = append(methods, InterfaceMethod{
												Name:      method.Name(),
												Signature: method.Type().String(),
											})
										}
									}
								}
							}
						}

						interfaceDoc := ""
						if genDecl.Doc != nil {
							interfaceDoc = genDecl.Doc.Text()
						}

						interfaces = append(interfaces, ExportedInterface{
							Name:    typeSpec.Name.Name,
							Doc:     interfaceDoc,
							Methods: methods,
						})
					}
				}
			}
		}
	}

	return interfaces, nil
}

// extractInterfacesFallback parses Go files directly as a fallback when go/packages fails.
// This is needed for testdata directories and other edge cases.
func extractInterfacesFallback(dir string) ([]ExportedInterface, error) {
	fset := token.NewFileSet()
	files, err := parseGoFiles(fset, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", dir, err)
	}

	var interfaces []ExportedInterface

	for pkgName, fileGroup := range files {
		// Skip test files
		var filtered []*ast.File
		for _, file := range fileGroup {
			if !strings.HasSuffix(fset.Position(file.Pos()).Filename, "_test.go") {
				filtered = append(filtered, file)
			}
		}

		if len(filtered) == 0 {
			continue
		}

		// Type check the package
		conf := types.Config{
			Sizes: types.SizesFor("gc", "amd64"),
		}

		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		}

		_, err = conf.Check(pkgName, fset, filtered, info)
		if err != nil {
			_ = err // Type checking errors are common in partial code, continue anyway
		}

		// Extract interfaces using go/types
		for _, file := range filtered {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if !ast.IsExported(typeSpec.Name.Name) {
						continue
					}

					// Check if it's an interface by looking at the AST
					if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						// Get type info from type checker
						obj := info.Defs[typeSpec.Name]
						var methods []InterfaceMethod

						if obj != nil {
							// For named types, we need to get the underlying type
							if named, ok := obj.Type().(*types.Named); ok {
								if iface, ok := named.Underlying().(*types.Interface); ok {
									for i := 0; i < iface.NumMethods(); i++ {
										method := iface.Method(i)
										if method.Exported() {
											methods = append(methods, InterfaceMethod{
												Name:      method.Name(),
												Signature: method.Type().String(),
											})
										}
									}
								}
							}
						}

						interfaceDoc := ""
						if genDecl.Doc != nil {
							interfaceDoc = genDecl.Doc.Text()
						}

						interfaces = append(interfaces, ExportedInterface{
							Name:    typeSpec.Name.Name,
							Doc:     interfaceDoc,
							Methods: methods,
						})
					}
				}
			}
		}
	}

	return interfaces, nil
}

// ExtractTypes extracts all exported types from a package directory using go/packages.
func ExtractTypes(dir string) ([]ExportedType, error) {
	// Use go/packages to load the package
	cfg := &packages.Config{
		Mode:  packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedName,
		Fset:  token.NewFileSet(),
		Tests: false, // Don't include test files
	}

	pkgs, err := packages.Load(cfg, dir)
	if err != nil {
		// If go/packages fails (e.g., testdata directory), fall back to parser.ParseDir
		return extractTypesFallback(dir)
	}

	var typesList []ExportedType

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			// If there are errors, try fallback
			return extractTypesFallback(dir)
		}

		// Skip test files (already filtered by Tests: false, but double-check)
		var files []*ast.File
		for _, file := range pkg.Syntax {
			filename := pkg.Fset.Position(file.Pos()).Filename
			if !strings.HasSuffix(filename, "_test.go") {
				files = append(files, file)
			}
		}

		if len(files) == 0 {
			continue
		}

		// Extract types using type info from go/packages
		for _, file := range files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if !ast.IsExported(typeSpec.Name.Name) {
						continue
					}

					// Get type info from type checker
					obj := pkg.TypesInfo.Defs[typeSpec.Name]
					var typeKind string

					if obj != nil {
						typeKind = getTypeKindFromType(obj.Type())
					} else {
						typeKind = getTypeKind(typeSpec.Type)
					}

					typeDoc := ""
					if genDecl.Doc != nil {
						typeDoc = genDecl.Doc.Text()
					}

					typesList = append(typesList, ExportedType{
						Name:     typeSpec.Name.Name,
						Doc:      typeDoc,
						TypeKind: typeKind,
					})
				}
			}
		}
	}

	return typesList, nil
}

// extractTypesFallback parses Go files directly as a fallback when go/packages fails.
// This is needed for testdata directories and other edge cases.
func extractTypesFallback(dir string) ([]ExportedType, error) {
	fset := token.NewFileSet()
	files, err := parseGoFiles(fset, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", dir, err)
	}

	var typesList []ExportedType

	for pkgName, fileGroup := range files {
		// Skip test files
		var filtered []*ast.File
		for _, file := range fileGroup {
			if !strings.HasSuffix(fset.Position(file.Pos()).Filename, "_test.go") {
				filtered = append(filtered, file)
			}
		}

		if len(filtered) == 0 {
			continue
		}

		// Type check the package
		conf := types.Config{
			Sizes: types.SizesFor("gc", "amd64"),
		}

		info := &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		}

		_, err = conf.Check(pkgName, fset, filtered, info)
		if err != nil {
			_ = err // Type checking errors are common in partial code, continue anyway
		}

		// Extract types using go/types
		for _, file := range filtered {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok != token.TYPE {
					continue
				}

				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if !ast.IsExported(typeSpec.Name.Name) {
						continue
					}

					// Get type info from type checker
					obj := info.Defs[typeSpec.Name]
					var typeKind string

					if obj != nil {
						typeKind = getTypeKindFromType(obj.Type())
					} else {
						typeKind = getTypeKind(typeSpec.Type)
					}

					typeDoc := ""
					if genDecl.Doc != nil {
						typeDoc = genDecl.Doc.Text()
					}

					typesList = append(typesList, ExportedType{
						Name:     typeSpec.Name.Name,
						Doc:      typeDoc,
						TypeKind: typeKind,
					})
				}
			}
		}
	}

	return typesList, nil
}

// getTypeKind determines the kind of type from an AST expression.
func getTypeKind(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	case *ast.Ident:
		return "type alias"
	case *ast.SelectorExpr:
		return "qualified type"
	case *ast.ArrayType:
		return "array"
	case *ast.MapType:
		return "map"
	case *ast.ChanType:
		return "channel"
	case *ast.FuncType:
		return "function type"
	case *ast.StarExpr:
		return "pointer"
	default:
		return "unknown"
	}
}

// getTypeKindFromType determines the kind of type from a types.Type.
// For named types, it checks the underlying type to determine the kind.
func getTypeKindFromType(typ types.Type) string {
	switch t := typ.(type) {
	case *types.Struct:
		return "struct"
	case *types.Interface:
		return "interface"
	case *types.Named:
		// For named types, check the underlying type
		underlying := t.Underlying()
		switch underlying.(type) {
		case *types.Interface:
			return "interface"
		case *types.Struct:
			return "struct"
		case *types.Basic:
			return "type alias"
		case *types.Pointer:
			return "pointer"
		case *types.Slice:
			return "slice"
		case *types.Array:
			return "array"
		case *types.Map:
			return "map"
		case *types.Chan:
			return "channel"
		case *types.Signature:
			return "function type"
		default:
			return "named type"
		}
	case *types.Basic:
		return "basic type"
	case *types.Pointer:
		return "pointer"
	case *types.Slice:
		return "slice"
	case *types.Array:
		return "array"
	case *types.Map:
		return "map"
	case *types.Chan:
		return "channel"
	case *types.Signature:
		return "function type"
	default:
		return "unknown"
	}
}

// parseGoFiles parses all .go files in dir, grouped by package name.
func parseGoFiles(fset *token.FileSet, dir string) (map[string][]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	pkgs := make(map[string][]*ast.File)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			continue
		}
		name := f.Name.Name
		pkgs[name] = append(pkgs[name], f)
	}
	return pkgs, nil
}
