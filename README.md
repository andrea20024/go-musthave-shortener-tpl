# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки и оптимизация производительности

### Бенчмарки

В проекте добавлены бенчмарки для ключевых компонентов системы:

| Benchmark | Время | Аллокации |
|-----------|-------|-----------|
| GenerateShortURL | ~207 ns/op | 32 B/op, 2 allocs/op |
| JSONHandler | ~7102 ns/op | 7554 B/op, 39 allocs/op |
| BatchHandler (5 items) | ~15138 ns/op | 8984 B/op, 71 allocs/op |
| BatchHandler (50 items) | ~94574 ns/op | 37714 B/op, 259 allocs/op |
| GetUserURLs (100 URLs) | ~2101 ns/op | 2144 B/op, 6 allocs/op |
| PostHandler | ~6530 ns/op | 7589 B/op, 33 allocs/op |

Запуск: `go test -bench=. -benchmem ./internal/handler/`

### Анализ памяти с pprof

Профилирование проводилось с использованием `go tool pprof` на реальном HTTP-сервере:

```bash
# Сбор профиля после оптимизации
curl http://localhost:9999/debug/pprof/allocs -o profiles/result.pprof

# Анализ
go tool pprof -top profiles/result.pprof
go tool pprof -list=JSONHandler profiles/result.pprof
```

**Ключевые findings до оптимизации:**
- `compress/flate.NewWriter`: 757MB (51.8%) — каждая HTTP-запрос создавала новый gzip.Writer
- `JSONHandler`: 1151MB cum (78.75%) — основной hotpath
- `fmt.Fprintln`: 236MB (16.16%) — логирование через fmt
- `BatchHandler`: 247MB (16.92%)

### Выполненные оптимизации

1. **Gzip Writer Pool** — `sync.Pool` для `gzip.Writer`:
   - Было: `gzip.NewWriter(w)` на каждый запрос
   - Стало: `pool.Get() → Reset() → Close() → pool.Put()`
   - **Результат**: 757MB → ~9MB (-98.8%)

2. **JSONHandler stream parsing** — замена `io.ReadAll` + `json.Unmarshal` на `json.NewDecoder`:
   - Избегание промежуточного буфера для тела запроса
   - 41 → 39 аллокаций (-4.9%)

3. **String concatenation** — замена `fmt.Sprintf("%s/%s", a, b)` на `a + "/" + b`:
   - Избегание форматирования и парсинга вербатов
   - Эффективно для простых конкатенаций

4. **sync.Pool для Input/Output** — пулы для JSON структур

5. **BatchHandler stream parsing** — замена `json.Unmarshal` на `json.NewDecoder`:
   - 311 → 259 аллокаций для large batch (-16.7%)

### Результаты дифференциального профилирования

```
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof

      flat  flat%   sum%        cum   cum%
 -748.34MB 51.20% 51.20% -1370.21MB 93.75%  compress/flate.NewWriter
 -379.67MB 25.98% 77.18%  -621.87MB 42.55%  compress/flate.(*compressor).init
 -239.70MB 16.40% 93.58%  -239.70MB 16.40%  compress/flate.newDeflateFast
  -13.52MB  0.93% 94.50%   -13.52MB  0.93%  sync.(*Pool).pinSlow
```

**Итого:**
- Gzip allocations: **-98.8%** (757MB → 9MB)
- JSONHandler: **~1%** improvement (pool overhead vs decoder savings)
- BatchHandler Large: **-16.7%** аллокаций (311 → 259)
