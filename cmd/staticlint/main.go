// Package main is the entry point for the multichecker static analysis tool
// for the URL Shortener project.
//
// Multichecker combines several static analyzers to check the entire project
// for common errors, anti-patterns, and Go idiom violations:
//
//   - Standard golang.org/x/tools/go/analysis/passes analyzers:
//     atomic, printf, sqlrowserr, shift, unreachable, unsafeptr, unusedresult
//
//   - All SA-class analyzers from staticcheck.io (syntax and logic errors):
//     SA0001, SA1000, SA1001, ..., SA9999 — full set
//
//   - Other staticcheck.io analyzer classes:
//     ST1000 (missing doc-comments), ST1003 (naming), ST1016 (qualifiers)
//
//   - All analyzers from honnef.co/go/tools/simple (S1xxx — code simplification)
//
//   - All analyzers from honnef.co/go/tools/stylecheck (STxxx — style violations)
//
//   - Public analyzers:
//     bodyclose — checks http.Response.Body is closed after requests
//
//   - Custom noosexit analyzer:
//     Prohibits direct os.Exit() calls in the main() function of the main package.
//
// Usage:
//
//	# Check the entire project:
//	go run cmd/staticlint/main.go ./...
//
//	# Check a specific package:
//	go run cmd/staticlint/main.go ./internal/...
//
//	# Check only cmd:
//	go run cmd/staticlint/main.go ./cmd/...
//
// List of all connected analyzers:
//
//	// Standard (golang.org/x/tools/go/analysis/passes):
//	atomic        — checks atomic operations via sync/atomic
//	printf        — incorrect arguments in fmt.* functions
//	sqlrowserr    — SQL Rows.Err() unchecked
//	shift         — incorrect bit shifts (shift by constant that is too large)
//	unreachable   — unreachable code after return/panic
//	unsafeptr     — invalid unsafe.Pointer conversions
//	unusedresult  — ignored function results (fmt.Fprintf, etc.)
//
//	// Staticcheck SA (all syntax and logic analyzers):
//	SA1000 — syntax errors, deprecated calls, invalid regex, etc.
//	SA2000 — unused or nil Go routine
//	SA3000 — broken close on receive
//	SA4000 — single-sided or double-sided literal
//	SA4002 — one or more useless comparisons
//	SA4003 — LeftHandSide is not a boolean
//	SA4004 — surrounding loop statement is not a loop
//	SA4005 — Loop closure (loop variable address taken)
//	SA4006 — Loop variable and loop variable pointers (unused value)
//	SA4010 — unreachable GoCode
//	SA4011 — missing error in return value
//	SA5000 — missing nil check of map
//	SA5001 — uninitialized struct
//	SA5002 — Unused field
//	SA5003 — Struct field tag is not well formed
//	... and all other SA classes (SA6000-SA9999)
//
//	// Staticcheck ST (style):
//	ST1000 — package comment is missing
//	ST1001 — missing or useless doc-comment
//	ST1003 — wrong naming convention
//	ST1016 — useless type qualifier
//
//	// Simple (S1xxx — simplification):
//	S1000 — replace if-then-else with bool conversion
//	S1001 — use copy for slice copy
//	S1002 — compare abs instead of manual calculation
//	... and all other S1xxx
//
//	// Stylecheck (STxxx — style):
//	ST1000-ST1028 — all standard style checks
//
//	// Public analyzers:
//	bodyclose   — checks Response.Body is closed after HTTP requests
//
//	// Custom:
//	noosexit    — prohibits os.Exit() in main.main() of the main package
//
// To fix found issues, review the linter messages and recommendations
// in each analyzer's documentation.

package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sqlrowserr"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/andrea20024/go-musthave-shortener-tpl/cmd/staticlint/noosexit"
	bodyclose "github.com/timakin/bodyclose/passes/bodyclose"
)

// allAnalyzers returns the full list of analyzers for multichecker.
//
// The function collects analyzers from several sources:
// 1. Standard pass analyzers from golang.org/x/tools
// 2. All SA analyzers from staticcheck (syntax & logic errors)
// 3. Selected ST analyzers from staticcheck (style)
// 4. All S analyzers from honnef.co/go/tools/simple
// 5. All ST analyzers from honnef.co/go/tools/stylecheck
// 6. Public analyzers: bodyclose
// 7. Custom noosexit analyzer
//
// Returns: a slice []*analysis.Analyzer with all connected analyzers.
func allAnalyzers() []*analysis.Analyzer {
	// Standard analyzers from golang.org/x/tools/go/analysis/passes
	myChecks := []*analysis.Analyzer{
		atomic.Analyzer,       // checks atomic operations via sync/atomic
		printf.Analyzer,       // incorrect arguments in fmt.* functions
		sqlrowserr.Analyzer,   // SQL Rows.Err() unchecked
		shift.Analyzer,        // incorrect bit shifts
		unreachable.Analyzer,  // unreachable code
		unsafeptr.Analyzer,    // invalid unsafe.Pointer
		unusedresult.Analyzer, // ignored function results
	}

	// All SA analyzers from staticcheck (syntax and logic errors)
	for _, a := range staticcheck.Analyzers {
		// Filter only SA-class analyzers (syntax & logic)
		if a.Analyzer.Name[0] == 'S' && a.Analyzer.Name[1] == 'A' {
			myChecks = append(myChecks, a.Analyzer)
		}
	}

	// Additional ST analyzers from staticcheck (style violations)
	for _, a := range staticcheck.Analyzers {
		// ST1000 — missing doc-comments in package
		// ST1003 — naming conventions (snake_case vs PascalCase)
		// ST1016 — useless type qualifiers (e.g. io.ReadCloser instead of Reader)
		if a.Analyzer.Name == "ST1000" ||
			a.Analyzer.Name == "ST1003" ||
			a.Analyzer.Name == "ST1016" {
			myChecks = append(myChecks, a.Analyzer)
		}
	}

	// All simplification analyzers (S1xxx)
	for _, a := range simple.Analyzers {
		myChecks = append(myChecks, a.Analyzer)
	}

	// All stylecheck analyzers (STxxx)
	for _, a := range stylecheck.Analyzers {
		myChecks = append(myChecks, a.Analyzer)
	}

	// Public analyzers
	myChecks = append(myChecks, bodyclose.Analyzer)

	// Custom analyzer: prohibit os.Exit() in main.main()
	myChecks = append(myChecks, noosexit.Analyzer)

	return myChecks
}

func main() {
	multichecker.Main(allAnalyzers()...)
}
