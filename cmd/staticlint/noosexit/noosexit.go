// Package noosexit implements a custom static analysis checker for Go that
// prohibits direct os.Exit() calls in main() functions.
//
// Purpose: prevents abrupt program termination without proper error handling.
// Direct os.Exit() in main() bypasses defer functions, leading to resource
// leaks and missing graceful shutdown.
//
// Recommendation: return an error code from main instead of os.Exit(code).
//
// The checker only triggers in the main package for the main() function.
package noosexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Doc is a detailed description of the analyzer for help output.
const Doc = `noosexit checks for direct os.Exit() calls in main.main(). Direct os.Exit() bypasses defer functions, which may lead to resource leaks and skipped graceful shutdown. Returning an error code from main is recommended instead of os.Exit.`

// Analyzer is the analyzer instance registered with multichecker.
//
// Analyzer configures the name, documentation, and run function,
// which performs AST traversal to find os.Exit() calls in main.main().
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  Doc,
	Run:  run,
}

// run is the main analyzer function.
//
// Algorithm:
// 1. Checks if the current package is main (skips all other packages).
// 2. Traverses all package files to find main() function declaration.
// 3. If main() is found, inspects its AST body for os.Exit() calls.
// 4. For each os.Exit() call, invokes pass.Reportf with:
//   - file and line number of the violation
//   - recommended replacement pattern
//
// Parameters: pass — analyzer context containing the package AST and file info.
// Returns: nil (the analyzer reports only messages, no return data).
func run(pass *analysis.Pass) (interface{}, error) {
	// Skip all packages except main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	var mainFunc *ast.FuncDecl

	// Find the main() function in package files
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				if funcDecl.Name.Name == "main" {
					mainFunc = funcDecl
					return false // main found, stop traversal
				}
			}
			return true
		})
	}

	// If main function is not found, do nothing
	if mainFunc == nil {
		return nil, nil
	}

	// Traverse the main() function body to find os.Exit() calls
	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		// Look for function calls only (CallExpr)
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if it is a selector expression (os.Exit)
		selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check that it is package "os"
		ident, ok := selectorExpr.X.(*ast.Ident)
		if !ok || ident.Name != "os" {
			return true
		}

		// Check that it is the function "Exit"
		if selectorExpr.Sel.Name != "Exit" {
			return true
		}

		// Found os.Exit() — report the error
		pos := pass.Fset.Position(callExpr.Pos())
		pass.Reportf(
			callExpr.Pos(),
			"direct call to os.Exit() is prohibited in main.main(); "+
				"return error code from main instead "+
				"(file: %s:%d)",
			pos.Filename,
			pos.Line,
		)

		return true
	})

	return nil, nil
}
