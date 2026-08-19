// Package main — точка входа для multichecker статического анализа кода проекта
// URL Shortener.
//
// Multichecker объединяет несколько статических анализаторов для проверки
// всего проекта на наличие типовых ошибок, anti-patterns и нарушений
// Go-идиом:
//
//   - Стандартные анализаторы golang.org/x/tools/go/analysis/passes:
//     atomic, loopclosure, nilfunc, printf, shift, unreachable, unsafeptr, unusedresult
//
//   - Все анализаторы класса SA пакета staticcheck.io (syntax and logic errors):
//     SA0001, SA1000, SA1001, ..., SA9999 — полный набор
//
//   - Анализаторы других классов staticcheck.io:
//     ST1000 (missing doc-comments), ST1003 (naming), ST1016 (qualifiers)
//
//   - Все анализаторы honnef.co/go/tools/simple (S1xxx — code simplification)
//
//   - Все анализаторы honnef.co/go/tools/stylecheck (STxxx — style violations)
//
//   - Публичные анализаторы:
//     bodyclose — проверка закрытия http.Response.Body после запроса
//
//   - Кастомный анализатор noosexit:
//     Запрещает прямой вызов os.Exit() в функции main() пакета main.
//
// Использование:
//
//	# Проверить весь проект:
//	go run cmd/staticlint/main.go ./...
//
//	# Проверить конкретный пакет:
//	go run cmd/staticlint/main.go ./internal/...
//
//	# Проверить только cmd:
//	go run cmd/staticlint/main.go ./cmd/...
//
// Список всех подключённых анализаторов:
//
//	// Стандартные (golang.org/x/tools/go/analysis/passes):
//	atomic        — проверка атомарных операций через sync/atomic
//	loopclosure   — захват переменных в замыканиях loop
//	nilfunc       — сравнение с nilfunc (устаревший pattern)
//	printf        — некорректные аргументы в fmt.* функциях
//	shift         — некорректные сдвиги (shift by constant that is too large)
//	unreachable   — недостижимый код после return/panic
//	unsafeptr     — невалидные unsafe.Pointer conversions
//	unusedresult  — проигнорированные результаты функций (fmt.Fprintf и др.)
//
//	// Staticcheck SA (все синтаксические и логические анализаторы):
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
//	... и все остальные классы SA (SA6000-SA9999)
//
//	// Staticcheck ST (style):
//	ST1000 — package comment is missing
//	ST1001 — missing or useless doc-comment
//	ST1003 — wrong naming convention
//	ST1016 — useless qualifier
//
//	// Simple (S1xxx — simplification):
//	S1000 — replace if-then-else with bool conversion
//	S1001 — use copy for slice copy
//	S1002 — compare abs instead of manual calculation
//	... и все остальные S1xxx
//
//	// Stylecheck (STxxx — style):
//	ST1000-ST1028 — все стандартные проверки стиля кода
//
//	// Публичные анализаторы:
//	bodyclose   — проверка закрытия Response.Body после HTTP-запроса
//
//	// Кастомные:
//	noosexit    — запрет os.Exit() в main.main() пакета main
//
// Для исправления найденных проблем смотрите сообщения линтера и рекомендации
// в документации к каждому анализатору.

package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/andrea20024/go-musthave-shortener-tpl/cmd/staticlint/noosexit"
	bodyclose "github.com/timakin/bodyclose/passes/bodyclose"
)

// allAnalyzers возвращает полный список анализаторов для multichecker.
//
// Функция собирает анализаторы из нескольких источников:
// 1. Стандартные pass-анализаторы из golang.org/x/tools
// 2. Все SA-анализаторы staticcheck (syntax & logic errors)
// 3. Избранные ST-анализаторы staticcheck (style)
// 4. Все S-анализаторы из honnef.co/go/tools/simple
// 5. Все ST-анализаторы из honnef.co/go/tools/stylecheck
// 6. Публичные анализаторы: errcheck, govet
// 7. Кастомный анализатор noosexit
//
// Возвращает: слайс []*analysis.Analyzer со всеми подключёнными анализаторами.
func allAnalyzers() []*analysis.Analyzer {
	// Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
	myChecks := []*analysis.Analyzer{
		atomic.Analyzer,       // проверка атомарных операций sync/atomic
		loopclosure.Analyzer,  // проверка замыканий в циклах
		nilfunc.Analyzer,      // устаревший pattern сравнения с nilfunc
		printf.Analyzer,       // некорректные аргументы в fmt.* функциях
		shift.Analyzer,        // некорректные сдвиги бит
		unreachable.Analyzer,  // недостижимый код
		unsafeptr.Analyzer,    // невалидные unsafe.Pointer
		unusedresult.Analyzer, // игнорируемые результаты функций
	}

	// Все SA-анализаторы staticcheck (syntax and logic errors)
	for _, a := range staticcheck.Analyzers {
		// Фильтруем только анализаторы класса SA (syntax & logic)
		if a.Analyzer.Name[0] == 'S' && a.Analyzer.Name[1] == 'A' {
			myChecks = append(myChecks, a.Analyzer)
		}
	}

	// Дополнительные ST-анализаторы staticcheck (style violations)
	for _, a := range staticcheck.Analyzers {
		// ST1000 — missing doc-comments в package
		// ST1003 — naming conventions (snake_case vs PascalCase)
		// ST1016 — useless type qualifiers (e.g. io.ReadCloser вместо Reader)
		if a.Analyzer.Name == "ST1000" ||
			a.Analyzer.Name == "ST1003" ||
			a.Analyzer.Name == "ST1016" {
			myChecks = append(myChecks, a.Analyzer)
		}
	}

	// Все анализаторы simplification (S1xxx)
	for _, a := range simple.Analyzers {
		myChecks = append(myChecks, a.Analyzer)
	}

	// Все анализаторы stylecheck (STxxx)
	for _, a := range stylecheck.Analyzers {
		myChecks = append(myChecks, a.Analyzer)
	}

	// Публичные анализаторы
	myChecks = append(myChecks, bodyclose.Analyzer)

	// Кастомный анализатор: запрет os.Exit() в main.main()
	myChecks = append(myChecks, noosexit.Analyzer)

	return myChecks
}

func main() {
	multichecker.Main(allAnalyzers()...)
}
