// Package main is a utility for generating Reset() methods for struct types
// tagged with // generate:reset.
//
// The generator scans all packages in the project and generates a Reset()
// method for each struct with the // generate:reset comment, resetting
// fields to their zero values:
//
//   - primitives (int, string, bool) → zero values
//   - slices → [:0] (truncate without nil)
//   - maps → clear()
//   - pointers to primitives → nil check + reset value
//   - nested structs with Reset() → call Reset()
//
// Generated methods are written to reset.gen.go in the source package.
//
// Usage:
//
//	go run ./cmd/reset/...
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

const resetTag = "generate:reset"

// structInfo holds information about a struct for Reset() generation.
type structInfo struct {
	name   string
	fields []fieldInfo
}

// fieldInfo holds information about a struct field.
type fieldInfo struct {
	name string
	typ  types.Type
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

func run() error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedTypesSizes,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	packages.PrintErrors(pkgs)

	for _, pkg := range pkgs {
		if pkg.Name == "main" {
			continue
		}

		structs := collectStructs(pkg)
		if len(structs) == 0 {
			continue
		}

		if err := writeResetFile(pkg, structs); err != nil {
			return fmt.Errorf("write reset file for %s: %w", pkg.PkgPath, err)
		}
		fmt.Printf("generated %s/reset.gen.go (%d structs)\n", pkg.PkgPath, len(structs))
	}

	return nil
}

// collectStructs finds all structs with the // generate:reset tag in a package.
func collectStructs(pkg *packages.Package) []structInfo {
	var result []structInfo

	// Map: filename -> tag line numbers
	fileTagLines := make(map[string]map[int]bool)
	for _, filename := range pkg.GoFiles {
		if strings.HasSuffix(filename, ".gen.go") {
			continue
		}
		tagLines := findTagLines(filename)
		if len(tagLines) > 0 {
			fileTagLines[filename] = tagLines
		}
	}

	// Map: filename -> *ast.File from pkg.Syntax
	fileToAST := make(map[string]*ast.File)
	for _, sf := range pkg.Syntax {
		if sf.Package.IsValid() {
			filename := pkg.Fset.Position(sf.Package).Filename
			fileToAST[filename] = sf
		}
	}

	// For each file with tags, find structs
	for filename, tagLines := range fileTagLines {
		astFile := fileToAST[filename]
		if astFile == nil {
			continue
		}

		for _, decl := range astFile.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Type == nil {
					continue
				}

				_, ok = typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				typeLine := pkg.Fset.Position(typeSpec.Name.Pos()).Line
				if !hasTagBeforeLine(typeLine, tagLines) {
					continue
				}

				fields := collectFieldsFromAST(typeSpec, pkg.TypesInfo)
				if len(fields) > 0 {
					result = append(result, structInfo{
						name:   typeSpec.Name.Name,
						fields: fields,
					})
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})

	return result
}

// findTagLines finds all lines with the // generate:reset tag in a file.
func findTagLines(filename string) map[int]bool {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	lines := make(map[int]bool)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		text := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(text, "//") {
			text = strings.TrimPrefix(text, "//")
			text = strings.TrimSpace(text)
			if text == resetTag {
				lines[lineNum] = true
			}
		}
	}
	return lines
}

// hasTagBeforeLine checks if there is a tag within N lines before the given line.
func hasTagBeforeLine(typeLine int, tagLines map[int]bool) bool {
	for tl := range tagLines {
		if tl < typeLine && typeLine-tl <= 3 {
			return true
		}
	}
	return false
}

// collectFieldsFromAST collects struct fields from AST + types.Info.
func collectFieldsFromAST(typeSpec *ast.TypeSpec, info *types.Info) []fieldInfo {
	var fields []fieldInfo
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	for _, field := range structType.Fields.List {
		if field.Names == nil {
			continue
		}
		for _, name := range field.Names {
			typ := info.TypeOf(name)
			fields = append(fields, fieldInfo{
				name: name.Name,
				typ:  typ,
			})
		}
	}
	return fields
}

// writeResetFile generates and writes reset.gen.go.
func writeResetFile(pkg *packages.Package, structs []structInfo) error {
	var sb strings.Builder

	sb.WriteString("// Code generated by cmd/reset; DO NOT EDIT.\n\n")
	sb.WriteString(fmt.Sprintf("package %s\n\n", pkg.Name))

	for _, s := range structs {
		sb.WriteString(generateResetMethod(s))
		sb.WriteString("\n")
	}

	genFile := filepath.Join(pkg.Dir, "reset.gen.go")
	return os.WriteFile(genFile, []byte(sb.String()), 0644)
}

// generateResetMethod generates the Reset() method for a struct.
func generateResetMethod(s structInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// Reset resets all fields of %s to zero values.\n", s.name))
	sb.WriteString(fmt.Sprintf("func (r *%s) Reset() {\n", s.name))

	for _, f := range s.fields {
		sb.WriteString(generateFieldReset(f))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// generateFieldReset generates the reset code for a single field.
func generateFieldReset(f fieldInfo) string {
	switch t := f.typ.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.String:
			return fmt.Sprintf("\tr.%s = \"\"\n", f.name)
		case types.Bool:
			return fmt.Sprintf("\tr.%s = false\n", f.name)
		default:
			return fmt.Sprintf("\tr.%s = 0\n", f.name)
		}
	case *types.Slice:
		return fmt.Sprintf("\tr.%s = r.%s[:0]\n", f.name, f.name)
	case *types.Map:
		return fmt.Sprintf("\tclear(r.%s)\n", f.name)
	case *types.Pointer:
		elem := t.Elem()
		switch elem.(type) {
		case *types.Basic:
			return fmt.Sprintf("\tif r.%s != nil {\n\t\t*r.%s = 0\n\t}\n", f.name, f.name)
		case *types.Struct:
			return fmt.Sprintf("\tif r.%s != nil {\n\t\tr.%s.Reset()\n\t}\n", f.name, f.name)
		default:
			return fmt.Sprintf("\tif r.%s != nil {\n\t\tr.%s = nil\n\t}\n", f.name, f.name)
		}
	default:
		return ""
	}
}
