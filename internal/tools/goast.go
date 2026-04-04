package tools

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	tool "github.com/benaskins/axon-tool"

	"github.com/benaskins/axon-code/internal/sandbox"
)

// GoASTTools holds the AST-aware ToolDefs bound to a project directory.
type GoASTTools struct {
	InspectTool tool.ToolDef
	RewriteTool tool.ToolDef
}

// NewGoASTTools constructs GoASTTools bound to projectDir.
func NewGoASTTools(projectDir string) GoASTTools {
	return GoASTTools{
		InspectTool: makeInspectTool(projectDir),
		RewriteTool: makeRewriteTool(projectDir),
	}
}

func makeInspectTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "ast",
		Description: "Inspect the structure of a Go source file. Returns package name, imports, and all top-level declarations with signatures, line numbers, and function bodies. This is the primary way to understand Go code: call it once per file to see everything.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]tool.PropertySchema{
				"path": {Type: "string", Description: "Path to a .go file, relative to the project directory."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("ast: missing required arg 'path'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("ast: " + err.Error())
			}
			if filepath.Ext(abs) != ".go" {
				return errResult("ast: not a .go file: " + path)
			}

			src, err := os.ReadFile(abs)
			if err != nil {
				return errResult("ast: " + err.Error())
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, abs, src, parser.ParseComments)
			if err != nil {
				return errResult("ast: parse error: " + err.Error())
			}

			return tool.ToolResult{Content: formatAST(fset, f, src)}
		},
	}
}

func makeRewriteTool(projectDir string) tool.ToolDef {
	return tool.ToolDef{
		Name: "rewrite",
		Description: `Perform a structural rewrite on a single Go source file. Each call operates on one file only. To rename across multiple files, call rewrite once per file.

Operations:
- "rename": Rename a function, type, or variable and all its references within this file.
- "replace_body": Replace the entire body of a function with new code.
- "replace_return": Replace the return expression(s) in a function.
- "change_signature": Change a function's parameter list and/or return types. Provide the full signature like "(a, b int) (int, error)".

The file is reformatted with gofmt after rewriting.`,
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"path", "operation"},
			Properties: map[string]tool.PropertySchema{
				"path":      {Type: "string", Description: "Path to a .go file, relative to the project directory."},
				"operation": {Type: "string", Description: "One of: rename, replace_body, replace_return, change_signature."},
				"target":    {Type: "string", Description: "Name of the function, type, or variable to modify."},
				"name":      {Type: "string", Description: "New name (for rename operation)."},
				"code":      {Type: "string", Description: "New code (for replace_body: function body without braces; for replace_return: return expression; for change_signature: full signature like '(a, b int) (int, error)')."},
			},
		},
		Execute: func(ctx *tool.ToolContext, a map[string]any) tool.ToolResult {
			path, ok := stringArg(a, "path")
			if !ok {
				return errResult("rewrite: missing required arg 'path'")
			}
			abs, err := sandbox.Resolve(projectDir, path)
			if err != nil {
				return errResult("rewrite: " + err.Error())
			}
			if filepath.Ext(abs) != ".go" {
				return errResult("rewrite: not a .go file: " + path)
			}

			op, ok := stringArg(a, "operation")
			if !ok {
				return errResult("rewrite: missing required arg 'operation'")
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, abs, nil, parser.ParseComments)
			if err != nil {
				return errResult("rewrite: parse error: " + err.Error())
			}

			var rewriteErr error
			switch op {
			case "rename":
				rewriteErr = doRename(fset, f, a)
			case "replace_body":
				rewriteErr = doReplaceBody(fset, f, a)
			case "replace_return":
				rewriteErr = doReplaceReturn(fset, f, a)
			case "change_signature":
				rewriteErr = doChangeSignature(fset, f, a, abs)
			default:
				return errResult(fmt.Sprintf("rewrite: unknown operation %q", op))
			}

			if rewriteErr != nil {
				return errResult("rewrite: " + rewriteErr.Error())
			}

			// Write back formatted.
			out, err := os.Create(abs)
			if err != nil {
				return errResult("rewrite: " + err.Error())
			}
			defer out.Close()

			if err := format.Node(out, fset, f); err != nil {
				return errResult("rewrite: format error: " + err.Error())
			}

			return tool.ToolResult{Content: fmt.Sprintf("rewrote %s in %s", op, path)}
		},
	}
}

// formatAST produces a structural summary of a Go file including function bodies.
func formatAST(fset *token.FileSet, f *ast.File, src []byte) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "package %s\n", f.Name.Name)

	// Imports.
	for _, imp := range f.Imports {
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name + " "
		}
		fmt.Fprintf(&sb, "  import %s%s\n", name, imp.Path.Value)
	}

	sb.WriteString("\n")

	// Top-level declarations.
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			start := fset.Position(d.Pos()).Line
			end := fset.Position(d.End()).Line
			// Extract the full function source from the file.
			funcSrc := extractSource(fset, d, src)
			fmt.Fprintf(&sb, "[%d:%d]\n%s\n\n", start, end, funcSrc)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					start := fset.Position(s.Pos()).Line
					end := fset.Position(s.End()).Line
					fmt.Fprintf(&sb, "  type %s %T [%d:%d]\n", s.Name.Name, s.Type, start, end)
				case *ast.ValueSpec:
					start := fset.Position(s.Pos()).Line
					for _, n := range s.Names {
						fmt.Fprintf(&sb, "  var %s [%d]\n", n.Name, start)
					}
				}
			}
		}
	}

	return sb.String()
}

// extractSource returns the source text for an AST node.
func extractSource(fset *token.FileSet, node ast.Node, src []byte) string {
	start := fset.Position(node.Pos()).Offset
	end := fset.Position(node.End()).Offset
	if start < 0 || end > len(src) || start >= end {
		return "// source unavailable"
	}
	return string(src[start:end])
}

func formatFuncType(ft *ast.FuncType) string {
	var sb strings.Builder
	sb.WriteString("(")
	sb.WriteString(formatFieldList(ft.Params))
	sb.WriteString(")")
	if ft.Results != nil && len(ft.Results.List) > 0 {
		results := formatFieldList(ft.Results)
		if len(ft.Results.List) == 1 && len(ft.Results.List[0].Names) == 0 {
			sb.WriteString(" " + results)
		} else {
			sb.WriteString(" (" + results + ")")
		}
	}
	return sb.String()
}

func formatFieldList(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, field := range fl.List {
		typeName := exprString(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeName)
		} else {
			var names []string
			for _, n := range field.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeName)
		}
	}
	return strings.Join(parts, ", ")
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(e.Elt)
	case *ast.MapType:
		return "map[" + exprString(e.Key) + "]" + exprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprString(e.Elt)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// doRename renames all occurrences of target to name within the file.
func doRename(fset *token.FileSet, f *ast.File, a map[string]any) error {
	target, ok := stringArg(a, "target")
	if !ok {
		return fmt.Errorf("rename requires 'target' arg")
	}
	newName, ok := stringArg(a, "name")
	if !ok {
		return fmt.Errorf("rename requires 'name' arg")
	}

	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			if node.Name == target {
				node.Name = newName
				found = true
			}
		}
		return true
	})

	if !found {
		return fmt.Errorf("identifier %q not found", target)
	}
	return nil
}

// doReplaceBody replaces the body of a named function.
func doReplaceBody(fset *token.FileSet, f *ast.File, a map[string]any) error {
	target, ok := stringArg(a, "target")
	if !ok {
		return fmt.Errorf("replace_body requires 'target' arg")
	}
	code, ok := stringArg(a, "code")
	if !ok {
		return fmt.Errorf("replace_body requires 'code' arg")
	}

	funcDecl := findFunc(f, target)
	if funcDecl == nil {
		return fmt.Errorf("function %q not found", target)
	}

	// Parse the new body as a function body.
	newBody, err := parseBody(code, fset)
	if err != nil {
		return fmt.Errorf("parse body: %w", err)
	}

	// Preserve original brace positions.
	newBody.Lbrace = funcDecl.Body.Lbrace
	newBody.Rbrace = funcDecl.Body.Rbrace
	funcDecl.Body = newBody
	return nil
}

// doReplaceReturn replaces the return expression in a function.
// Finds the last return statement and replaces its results.
func doReplaceReturn(fset *token.FileSet, f *ast.File, a map[string]any) error {
	target, ok := stringArg(a, "target")
	if !ok {
		return fmt.Errorf("replace_return requires 'target' arg")
	}
	code, ok := stringArg(a, "code")
	if !ok {
		return fmt.Errorf("replace_return requires 'code' arg")
	}

	funcDecl := findFunc(f, target)
	if funcDecl == nil {
		return fmt.Errorf("function %q not found", target)
	}

	// Parse the return expression(s).
	exprs, err := parseExprs(code)
	if err != nil {
		return fmt.Errorf("parse return expression: %w", err)
	}

	// Replace all return statements in the function.
	found := false
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if ret, ok := n.(*ast.ReturnStmt); ok {
			ret.Results = exprs
			found = true
		}
		return true
	})

	if !found {
		return fmt.Errorf("no return statement found in %q", target)
	}
	return nil
}

// doChangeSignature replaces a function's signature.
// code should be the full signature like "(a, b int) (int, error)"
func doChangeSignature(fset *token.FileSet, f *ast.File, a map[string]any, filePath string) error {
	target, ok := stringArg(a, "target")
	if !ok {
		return fmt.Errorf("change_signature requires 'target' arg")
	}
	code, ok := stringArg(a, "code")
	if !ok {
		return fmt.Errorf("change_signature requires 'code' arg")
	}

	funcDecl := findFunc(f, target)
	if funcDecl == nil {
		return fmt.Errorf("function %q not found", target)
	}

	// Parse the new signature by wrapping in a func declaration.
	newType, err := parseFuncType(code, fset)
	if err != nil {
		return fmt.Errorf("parse signature: %w", err)
	}

	// Preserve the original Func keyword position to avoid formatting breakage.
	newType.Func = funcDecl.Type.Func
	funcDecl.Type = newType
	return nil
}

func findFunc(f *ast.File, name string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

func parseBody(code string, fset *token.FileSet) (*ast.BlockStmt, error) {
	// Wrap in a function so we can parse it.
	src := "package p\nfunc _() {\n" + code + "\n}"
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			clearPositions(fd.Body)
			return fd.Body, nil
		}
	}
	return nil, fmt.Errorf("no function body parsed")
}

func parseExprs(code string) ([]ast.Expr, error) {
	// Wrap in a return statement inside a function.
	src := "package p\nfunc _() { return " + code + " }"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			for _, stmt := range fd.Body.List {
				if ret, ok := stmt.(*ast.ReturnStmt); ok {
					for _, expr := range ret.Results {
						clearPositions(expr)
					}
					return ret.Results, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no expressions parsed")
}

func parseFuncType(sig string, fset *token.FileSet) (*ast.FuncType, error) {
	src := "package p\nfunc _" + sig + " {}"
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			clearPositions(fd.Type)
			return fd.Type, nil
		}
	}
	return nil, fmt.Errorf("no function type parsed")
}

// clearPositions zeros out all token.Pos values in an AST subtree so that
// format.Node does not insert spurious whitespace from mismatched positions.
func clearPositions(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		switch v := n.(type) {
		case *ast.Ident:
			v.NamePos = token.NoPos
		case *ast.BasicLit:
			v.ValuePos = token.NoPos
		case *ast.BinaryExpr:
			v.OpPos = token.NoPos
		case *ast.UnaryExpr:
			v.OpPos = token.NoPos
		case *ast.ParenExpr:
			v.Lparen = token.NoPos
			v.Rparen = token.NoPos
		case *ast.CallExpr:
			v.Lparen = token.NoPos
			v.Rparen = token.NoPos
		case *ast.IndexExpr:
			v.Lbrack = token.NoPos
			v.Rbrack = token.NoPos
		case *ast.BlockStmt:
			v.Lbrace = token.NoPos
			v.Rbrace = token.NoPos
		case *ast.ReturnStmt:
			v.Return = token.NoPos
		case *ast.AssignStmt:
			v.TokPos = token.NoPos
		case *ast.IfStmt:
			v.If = token.NoPos
		case *ast.ForStmt:
			v.For = token.NoPos
		case *ast.RangeStmt:
			v.For = token.NoPos
			v.TokPos = token.NoPos
		case *ast.BranchStmt:
			v.TokPos = token.NoPos
		case *ast.ExprStmt:
			// no position field
		case *ast.DeclStmt:
			// no position field
		case *ast.FieldList:
			v.Opening = token.NoPos
			v.Closing = token.NoPos
		case *ast.Field:
			// positions are on child nodes
		case *ast.FuncType:
			v.Func = token.NoPos
		case *ast.SelectorExpr:
			// positions are on child nodes
		case *ast.StarExpr:
			v.Star = token.NoPos
		case *ast.CompositeLit:
			v.Lbrace = token.NoPos
			v.Rbrace = token.NoPos
		case *ast.KeyValueExpr:
			v.Colon = token.NoPos
		case *ast.GenDecl:
			v.TokPos = token.NoPos
			v.Lparen = token.NoPos
			v.Rparen = token.NoPos
		case *ast.ValueSpec:
			// positions on child nodes
		case *ast.TypeSpec:
			v.Assign = token.NoPos
		}
		return true
	})
}
