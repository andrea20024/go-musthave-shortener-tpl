// Package noosexit реализует кастомный анализатор статического анализа кода Go,
// который запрещает прямой вызов os.Exit() в функции main() пакета main.
//
// Назначение: предотвращает аварийное завершение программы без корректной
// обработки ошибок. Прямой вызов os.Exit() в main() обходит defer-функции,
// что приводит к утечке ресурсов и пропуску graceful shutdown.
//
// Рекомендация: вместо os.Exit(code) возвращайте код ошибки из main.
//
// Анализатор срабатывает только в пакете main для функции main().
package noosexit

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Doc — подробное описание анализатора для вывода в help-сообщениях.
const Doc = `noosexit проверяет отсутствие прямого вызова os.Exit() в функции main пакета main. Прямой вызов os.Exit обходит defer-функции, что может привести к утечке ресурсов и пропуску graceful shutdown. Рекомендуется возвращать код ошибки из main вместо os.Exit.`

// Analyzer — экземпляр анализатора, регистрируемый в multichecker.
//
// Analyzer настраивает имя, документацию и функцию run,
// которая выполняет AST-обход для поиска вызовов os.Exit в main.main().
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  Doc,
	Run:  run,
}

// run — основная функция анализатора.
//
// Выполняет следующий алгоритм:
//
// 1. Проверяет, что текущий пакет — main (пропускает все другие пакеты).
//
// 2. Обходит все файлы пакета и ищет объявление функции main().
//
// 3. Если main() найдена, обходит её тело AST для поиска вызовов os.Exit().
//
// 4. Для каждого вызова os.Exit() вызывает pass.Reportf с указанием:
//   - файла и номера строки нарушения
//   - рекомендуемого паттерна замены
//
// Параметры: pass — контекст анализатора, содержащий AST пакета и информацию о файлах.
// Возвращает: nil (анализатор не возвращает данных, только сообщения).
func run(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что текущий пакет — main
	// Анализатор игнорирует все другие пакеты (internal, cmd/profiler и т.д.)
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	var mainFunc *ast.FuncDecl

	// Находим функцию main() в файлах пакета
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				if funcDecl.Name.Name == "main" {
					mainFunc = funcDecl
					return false // main найден, прекращаем обход
				}
			}
			return true
		})
	}

	// Если функция main не найдена — ничего не делать
	if mainFunc == nil {
		return nil, nil
	}

	// Игнорируем временные файлы из go-build кэша
	// multichecker.Main() сканирует временные файлы компиляции,
	// которые тоже попадают в pass.Files
	for _, f := range pass.Files {
		pos := pass.Fset.Position(f.Pos())
		if strings.Contains(pos.Filename, "go-build") {
			return nil, nil
		}
	}

	// Обходим тело функции main() для поиска вызовов os.Exit()
	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		// Ищем только вызовы функций (CallExpr)
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Проверяем, что это селекторный вызов (os.Exit)
		selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Проверяем, что это package "os"
		ident, ok := selectorExpr.X.(*ast.Ident)
		if !ok || ident.Name != "os" {
			return true
		}

		// Проверяем, что это функция "Exit"
		if selectorExpr.Sel.Name != "Exit" {
			return true
		}

		// Нашли os.Exit() — сообщаем об ошибке
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
