package calc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestVerifyIndependenceArchitecture statically verifies that dual.go and verification helper paths
// do not call or import solver rewrite routines (e.g. differentiate, expandNode, simplifyCore).
func TestVerifyIndependenceArchitecture(t *testing.T) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "dual.go", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse dual.go: %v", err)
	}

	// Forbidden identifier names inside the independent dual evaluator
	forbiddenFuncs := map[string]bool{
		"differentiate":           true,
		"expandNode":              true,
		"simplify":                true,
		"simplifyCore":            true,
		"checkIsZeroAlgebraically": true,
	}

	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok {
				if forbiddenFuncs[ident.Name] {
					t.Errorf("Architecture violation: dual.go directly calls solver function %s()", ident.Name)
				}
			}
		}
		return true
	})
}

// TestForbiddenImportsInVerify checks that verify packages/files don't import unwanted subsystems.
func TestForbiddenImportsInVerify(t *testing.T) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "dual.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("Failed to parse imports of dual.go: %v", err)
	}

	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "unsafe") {
			t.Errorf("Forbidden import of unsafe in dual.go: %s", path)
		}
	}
}
