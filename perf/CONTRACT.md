# Контракт на реализацию: ТИОПО ЛР3 (микробенчмарк) и ЛР4 (нагрузочное тестирование)

Исполнитель: Gemini. Архитектор и ревьюер: Claude.
Исполнитель пишет код строго по этому документу. Архитектурных решений не принимает.
Если контракт расходится с реальным кодом репозитория или что-то не работает так, как здесь описано, исполнитель
не импровизирует, а останавливается и формулирует вопрос в ответе.

---

## 0. Правила для исполнителя (обязательны)

### 0.1. Границы изменений

Разрешено создавать и менять только:

- всё внутри `perf/` (кроме этого файла `perf/CONTRACT.md` — его не трогать);
- `Makefile` — только дописать в конец два блока целей (раздел 3.7 и 4.9), существующие цели не трогать;
- `.gitignore` — добавить одну строку `/perf/results/`.

Запрещено: менять `services/`, `shared/`, `tests/`, `tools/`, `scripts/`, `deploy/`, `docker-compose*.yml`,
`config/`, существующие Dockerfile. Запрещено делать `git commit`, `git push`, `git stash`, `git checkout`.

### 0.2. Стиль кода

- Никаких эмодзи нигде: ни в коде, ни в комментариях, ни в выводе скриптов, ни в markdown.
- Никаких декоративных линий в комментариях (`// ======`, `# ------`, `/* **** */` и т.п.).
- Комментарии — только если объясняют **почему**, а не **что**. Код без комментариев — нормальная ситуация.
  Комментарии вида `// Create client`, `# Parse args`, `// Loop over items`, `// Step 1: ...` запрещены.
  Язык комментариев — русский, коротко.
- Ранний возврат вместо вложенных if/else.
- Никаких «на всякий случай» абстракций, фабрик, интерфейсов с одной реализацией, конфиг-классов ради трёх полей.
  Ровно то, что описано в контракте. Лишние фичи (CLI-цвета, прогресс-бары, ретраи «для надёжности», логгеры-обёртки)
  не добавлять.
- Go: ошибки оборачивать в стиле проекта `fmt.Errorf("%s: %w", op, err)` или `fmt.Errorf("context: %w", err)`.
  `gofmt` обязателен. `panic` только в `main` при невалидных флагах.
- Bash: `#!/usr/bin/env bash` + `set -euo pipefail`, переменные в кавычках, `trap` для очистки контейнеров.
  Вывод скриптов — короткие строки `--> что делаю`, без баннеров из `=====`.
- Python: 3.12, без классов там, где хватает функций, type hints на сигнатурах, `argparse`, без `print` отладочного мусора.
- JS (k6): ES-модули, без TypeScript, без сборщиков.
- Не создавать README, CHANGELOG, NOTES, SUMMARY и прочие md-файлы, кроме явно указанных в контракте.
- Версии всех образов и библиотек — пиннить ровно как указано в контракте. `latest` запрещён.

### 0.3. Порядок работы

Работа идёт по этапам (раздел 5). После каждого этапа выполняется его проверка (Definition of Done).
Если проверка не проходит — чинить, а не пропускать. Длинные прогоны (soak на час, полный набор ЛР3) **не запускать**,
только smoke-режимы. В конце — короткий отчёт в ответе: список созданных файлов, какие команды проверки
запускались и их результат, какие вопросы остались.

---

## 1. Контекст проекта (то, что уже есть)

GoFlow — микросервисная система (Go 1.25, `go.work`):

- `gateway` (HTTP, gin, `:8080`) -> gRPC: `auth` (`:50051`), `accounts` (`:50052`), `transactions` (`:50053`).
- Хранилище: PostgreSQL 16 (у каждого сервиса своя БД), Redis (blacklist токенов в `auth`).
- Сервисы сами накатывают миграции при старте (`migrations.RunMigrations`).
- Конфиг сервиса: YAML по пути из `CONFIG_PATH`; пользователь/пароль БД из env `DATABASE_USER`/`DATABASE_PASSWORD`;
  `auth` требует `JWT_PRIVATE_KEY_PATH`; `gateway` требует `JWT_PUBLIC_KEY_PATH` и `BOOTSTRAP_TOKEN`.
- Образцы рабочих конфигов: генерируются в `scripts/test_stand_up.sh` (стенд ЛР2) — брать структуру оттуда.
- Dockerfile каждого сервиса: `services/<name>/Dockerfile`, контекст сборки — корень репозитория.
- HTTP API v1 (см. `services/gateway/internal/delivery/http/router/router.go`, DTO в `services/gateway/internal/dto/`,
  рабочий пример всей цепочки — `tests/e2e/mvp_transfer_e2e_test.go`):
  - `POST /api/v1/auth/register/company` `{login,email,password,company_name}` -> 201 `{user_id,company_id,email,role,token}`
  - `POST /api/v1/auth/login` `{email,password}` -> 200 `{..., token}`
  - `POST /api/v1/auth/logout` (Bearer)
  - `POST /api/v1/accounts` `{currency:"RUB"}` -> 201 `{account_id,...}` (роль company_admin)
  - `POST /api/v1/accounts/bank` `{account_id,name,bic,settlement_account,currency}` -> 201 `{bank_account_id,...}`
  - `POST /api/v1/accounts/bank/deposit` `{account_id,bank_account_id,amount,idempotency_key}` -> 200
  - `GET  /api/v1/companies/accounts`
  - `GET  /api/v1/accounts/:account_id/balance`
  - `POST /api/v1/accounts/:account_id/statement` `{period_from,period_to}` (RFC3339)
  - `POST /api/v1/transactions/transfer` `{from_account_id,to_account_id,amount,currency,idempotency_key}` -> 200
  - `GET  /api/v1/transactions/:account_id`
- Маппинг gRPC -> HTTP: `services/gateway/internal/delivery/http/handlers/helpers.go`
  (NotFound 404, AlreadyExists 409, InvalidArgument 400, Unauthenticated 401, PermissionDenied 403, иначе 500).
- Каждый защищённый запрос в gateway: проверка JWT + gRPC-вызов `auth.IsTokenValid` (Redis).
- Хост: 8 логических CPU (0-7), ~7 ГБ RAM, Docker 29, cgroup v2.

---

## 2. Общая структура

```
perf/
  CONTRACT.md                  (этот файл, не трогать)
  analysis/                    общий Python-анализ для ЛР3 и ЛР4
    Dockerfile
    requirements.txt
    plots.py                   общие функции: статистика, графики
    lr3_report.py
    lr4_report.py
  lr3-bench/                   ЛР3: Go-модуль lr3bench, только stdlib
    go.mod
    Dockerfile
    run.sh
    cmd/bench/main.go
    internal/harness/harness.go
    internal/harness/calibrate.go
    internal/scenario/scenario.go
  lr4-load/                    ЛР4
    compose.monitoring.yml
    compose.sut.yml
    profiles.env
    run.sh
    config/{auth,accounts,transactions,gateway}.yaml
    prometheus/prometheus.yml
    grafana/provisioning/datasources/prometheus.yml
    grafana/provisioning/dashboards/dashboards.yml
    grafana/dashboards/goflow-load.json
    k6/main.js
    k6/lib/api.js
    k6/lib/seed.js
    k6/lib/errors.js
  results/                     (gitignored) результаты прогонов
```

Образ анализа (`perf/analysis/Dockerfile`):

- `FROM python:3.12-slim`, `requirements.txt`: `numpy==2.0.1`, `pandas==2.2.2`, `matplotlib==3.9.1`, `requests==2.32.3`.
- Тег образа: `goflow-perf-analysis:local`. Запуск: `docker run --rm -v <results>:/data [--network ...] goflow-perf-analysis:local python <script> ...`.
- Скрипты копируются в образ в `/app`.

### 2.1. `perf/analysis/plots.py` — общие функции

**Устарело, заменено разделом 6.**

Ровно эти функции (имена и сигнатуры фиксированы):

```python
PERCENTILES = [50, 75, 90, 95, 99]

def percentiles(values: np.ndarray) -> dict[str, float]
    # ключи "p50","p75","p90","p95","p99" + "min","max","mean","median","std"; numpy.percentile, method="linear"

def plot_timeseries(ax, t_sec: np.ndarray, values: np.ndarray, window_sec: float, label: str) -> None
    # точки (scatter, alpha 0.2, s=2) + линии скользящих p50 и p99 по окнам window_sec

def plot_percentile_curve(ax, values: np.ndarray, label: str) -> None
    # распределение по перцентилям по числу запросов: X = перцентиль в шкале 1/(1-q) (как HdrHistogram),
    # подписи тиков 50%, 90%, 99%, 99.9%, 99.99%; Y = значение

def plot_histogram(ax, values: np.ndarray, label: str, bins: int = 100) -> None
    # гистограмма с вертикальными линиями p50/p95/p99; при разбросе > 2 порядков — логарифмические бины

def shade_phases(ax, phases: list[tuple[float, float, str]]) -> None
    # закрашивает интервалы фаз (warmup/teardown/stress...) полупрозрачным фоном с подписью

def write_markdown_table(path: str, header: list[str], rows: list[list]) -> None
```

Все графики: `matplotlib` backend `Agg`, PNG 150 dpi, подписи осей с единицами измерения, заголовки на русском.

---

## 3. ЛР3 — микробенчмарк: встроенный `map` в Go 1.23 против Go 1.25

### 3.1. Объект сравнения и обоснование

Сравниваются две версии языка (пункт 3 задания: «разные мажорные версии языка»):

- `go1.23` — классическая реализация `map` (бакеты по 8 элементов + overflow-цепочки);
- `go1.25` — реализация на Swiss Tables, появившаяся в Go 1.24 (группы по 8 слотов, control-байты, open addressing).

Код бенча один и тот же, собирается двумя тулчейнами в два образа. Проект GoFlow собран на Go 1.25.4,
то есть ЛР3 показывает, что проект получил от новой реализации map.

Единичное обращение к map занимает ~10–50 нс, это меньше нижней границы задания (100 нс). Поэтому
**одна операция бенча = пачка из 1000 обращений** (`BatchSize = 1000`) — это 10–100 мкс, в рамках задания.
В отчёте дополнительно выводится время на одно обращение (`latency / BatchSize`).

Тип map во всех сценариях: `map[uint64]uint64`.

### 3.2. Модуль

`perf/lr3-bench/go.mod`: `module lr3bench`, `go 1.23` (обе версии тулчейна должны собирать модуль),
без директивы `toolchain`. **Только стандартная библиотека**, ни одной внешней зависимости, `go.sum` не нужен.
Модуль в `go.work` не добавляется; локальная сборка — `GOWORK=off go build ./...` из `perf/lr3-bench`.

### 3.3. Данные и ключи

- Размеры map (`-size`): `1000`, `100000`, `1000000`. Маленькая map помещается в кеш CPU, большая — нет.
- Генератор: `math/rand/v2`, `rand.New(rand.NewPCG(seed, seed))`, seed из флага (по умолчанию 42).
- Подготовка вне замера (до старта `harness.Run`):
  - `present []uint64` — `size` уникальных случайных ключей, которыми заполнена map (без size hint в `make`,
    заполнение обычными вставками);
  - `absent []uint64` — `size` случайных ключей, гарантированно отсутствующих в map;
  - порядок обращений — заранее перемешанные копии этих слайсов; операция берёт следующие `BatchSize` ключей
    со сдвигом курсора, по кругу. Никакого rng внутри замеряемой операции.
- Результаты чтения складываются в переменную пакетного уровня `var sink uint64`, чтобы компилятор не выкинул поиск.

### 3.4. Код

#### `internal/harness/harness.go` — собственный инструмент бенчмаркинга

```go
type Phase uint8

const (
    PhaseWarmup Phase = iota
    PhaseMeasure
    PhaseTeardown
)

type Sample struct {
    StartNs   int64 // смещение от старта прогона, монотонные часы
    LatencyNs int64
    Op        uint8 // индекс в Config.Ops
    Phase     Phase
}

type Worker struct {
    Op uint8
    Fn func()
}

type Config struct {
    Warmup   time.Duration
    Measure  time.Duration
    Teardown time.Duration
    Ops      []string // уникальные имена операций, индекс = Sample.Op
    Workers  []Worker // каждый Worker — отдельная горутина, крутит свой Fn в цикле
}

type Result struct {
    Samples       []Sample // слияние всех воркеров, отсортировано по StartNs
    TotalDuration time.Duration
    GCCycles      uint32 // за фазу Measure
}

func Run(cfg Config) Result
```

Требования к `Run`:

- Перед стартом: `runtime.GC()`. Один общий `start := time.Now()`.
- Цикл воркера: `t0 := time.Now(); w.Fn(); t1 := time.Now()`. Фаза определяется по `t0.Sub(start)`
  относительно границ `Warmup`, `Warmup+Measure`, `Warmup+Measure+Teardown`. Выход из цикла, когда `t1` вышел
  за конец Teardown. Внутри цикла — никаких дополнительных вызовов часов, мьютексов, каналов и аллокаций.
- У каждого воркера свой заранее выделенный слайс сэмплов (`make([]Sample, 0, 1<<21)`), запись через `append`.
- Все воркеры стартуют одновременно через общий барьер (канал, который закрывается после запуска всех горутин).
- GC-счётчик: отдельная горутина спит до `start+Warmup` и до `start+Warmup+Measure` и в эти моменты снимает
  `runtime.ReadMemStats().NumGC`. В горячем цикле `ReadMemStats` не вызывать.

#### `internal/harness/calibrate.go` — оценка overhead инструмента (требование 7)

```go
type Overhead struct {
    TimerP50Ns       float64 // медиана t1.Sub(t0) для двух подряд time.Now(), 1_000_000 замеров
    TimerP99Ns       float64
    HarnessNoopP50Ns float64 // медиана LatencyNs при Run() с пустым Fn (1 воркер, 0.5s/2s/0.5s)
    HarnessNoopP99Ns float64
    LoopPerIterNs    float64 // время цикла из 10_000_000 вызовов пустого Fn без замеров / 10_000_000
}

func Calibrate() Overhead
```

Пустой `Fn` — функция с `//go:noinline`, вызывается через переменную пакетного уровня типа `func()`, чтобы
вызов не заинлайнился. `Calibrate` выполняется в каждом прогоне до подготовки данных.

#### `internal/scenario/scenario.go`

```go
const BatchSize = 1000

func Build(name string, size int, seed uint64) (harness.Config, error)
```

`Build` готовит map и ключи (3.3) и возвращает конфиг с воркерами. Сценарии:

| Сценарий      | Воркеры                           | Одна операция (`Fn`) |
|---------------|-----------------------------------|----------------------|
| `lookup_hit`  | 1 x `lookup_hit`                  | 1000 поисков существующих ключей, `v, ok := m[k]`, результат в `sink` |
| `lookup_miss` | 1 x `lookup_miss`                 | 1000 поисков отсутствующих ключей |
| `update`      | 1 x `update`                      | 1000 присваиваний `m[k] = v` по существующим ключам (запись без роста) |
| `churn`       | 1 x `churn`                       | 500 `delete` существующих + 500 вставок новых ключей; размер map остаётся `size` |
| `grow`        | 1 x `grow`                        | `m := make(map[uint64]uint64)` без hint + 1000 вставок; `-size` игнорируется, в meta пишется 1000 |
| `mixed`       | 3 x `mixed_read` + 1 x `mixed_write` | одновременные чтение и запись над одной map под `sync.RWMutex`: читатель — `RLock`, 1000 поисков, `RUnlock`; писатель — `Lock`, 100 `delete` + 100 вставок, `Unlock` |

Для `churn` и `mixed_write`: удалённые ключи переходят в пул «отсутствующих», новые берутся из заранее
сгенерированного пула (не rng в цикле); пулы — кольцевые буферы, подготовленные в `Build`. Размер map должен
оставаться равным `size` на протяжении всего прогона — проверить в конце прогона и выйти с ошибкой, если нет.

Блокировка в `mixed` берётся на всю пачку, а не на каждое обращение: иначе бенч мерил бы `RWMutex`, а не map.

#### `cmd/bench/main.go`

Флаги: `-scenario`, `-size` (int), `-out` (директория), `-seed` (42), `-warmup` (2s), `-duration` (10s),
`-teardown` (1s), `-run` (номер прогона, int).

Последовательность: `Calibrate` -> `scenario.Build` -> `harness.Run` -> запись файлов.

Выходные файлы в `-out`:

- `samples.csv`: заголовок `start_ns,latency_ns,op,phase`; `op` — имя операции, `phase` — `warmup|measure|teardown`.
  Запись через `bufio.Writer` + `strconv.AppendInt`, без `encoding/csv`.
- `meta.json`:

```json
{
  "go_version": "go1.25.4", "scenario": "lookup_hit", "size": 100000, "batch_size": 1000, "run": 1, "seed": 42,
  "warmup_ns": 0, "measure_ns": 0, "teardown_ns": 0, "total_duration_ns": 0,
  "counts": {"warmup": {"lookup_hit": 0}, "measure": {"lookup_hit": 0}, "teardown": {"lookup_hit": 0}},
  "gc_cycles_measure": 0,
  "overhead": {"timer_p50_ns": 0, "timer_p99_ns": 0, "harness_noop_p50_ns": 0, "harness_noop_p99_ns": 0, "loop_per_iter_ns": 0},
  "env": {"gomaxprocs": 1, "num_cpu": 1, "started_at": "RFC3339"}
}
```

`go_version` — `runtime.Version()`.

### 3.5. Docker и изоляция

`perf/lr3-bench/Dockerfile`: multi-stage, `ARG GO_IMAGE`, `FROM ${GO_IMAGE} AS builder` -> `FROM alpine:3.19`,
контекст — `perf/lr3-bench`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, `GOWORK=off`. Два образа:

| Тег образа                    | `GO_IMAGE`             |
|-------------------------------|------------------------|
| `goflow-lr3-bench:go1.23`     | `golang:1.23.4-alpine` |
| `goflow-lr3-bench:go1.25`     | `golang:1.25.4-alpine` |

`perf/lr3-bench/run.sh`. Переменные окружения с дефолтами:
`RUNS=5`, `WARMUP=2s`, `DURATION=10s`, `TEARDOWN=1s`, `BENCH_CPU=2`, `VERSIONS="go1.23 go1.25"`,
`SCENARIOS="lookup_hit lookup_miss update churn mixed grow"`, `SIZES="1000 100000 1000000"`.

Алгоритм:

1. `docker build` обоих образов бенча и образа анализа.
2. `OUT=perf/results/lr3/$(date +%Y%m%d-%H%M%S)`.
3. Цикл `for run in 1..RUNS; for scenario; for size; for version`: версии чередуются внутри одной комбинации,
   чтобы дрейф машины не попадал систематически в одну из них. Для `grow` цикл по размерам не делается (один размер).
4. На каждую итерацию — **свежий контейнер** из того же образа:
   `docker run --rm --cpuset-cpus "$BENCH_CPU" --memory 512m -e GOMAXPROCS=1 -v "$OUT:/out" goflow-lr3-bench:$version
   -scenario ... -size ... -run ... -out /out/$version/$scenario/$size/run-$run`.
   Сеть контейнеру не нужна: `--network none`.
5. В конце — `lr3_report.py` в контейнере анализа на `$OUT`.

При дефолтах полный прогон — около 2 x (5 x 3 + 1) x 5 = 160 контейнеров по ~14 с, примерно 40 минут.

### 3.6. Отчёт ЛР3 (`perf/analysis/lr3_report.py`)

**Устарело, заменено разделом 6.**

Вход: директория прогона (`/data`). Статистика считается **только по фазе measure**, по всем прогонам вместе (pooled).

Для каждой тройки (сценарий, размер, операция):

- `figures/<scenario>/<size>/<op>_timeseries.png` — латентность во времени для run-1 обеих версий (2 подграфика),
  все фазы, warmup/teardown закрашены `shade_phases`, окно скользящих перцентилей 0.5 с;
- `figures/<scenario>/<size>/<op>_percentiles.png` — кривые перцентилей go1.23 и go1.25 на одном графике;
- `figures/<scenario>/<size>/<op>_histogram.png` — две гистограммы с общей осью X.

Сводные:

- `figures/<scenario>_by_size.png` — p50 и p99 (нс на одно обращение) в зависимости от размера map,
  две версии, логарифмическая ось X;
- `figures/overhead.png` — столбцы `timer_p50`, `harness_noop_p50`, `loop_per_iter` (медиана по всем прогонам).

`summary.md`:

- таблица на каждую тройку: версия, p50, p75, p90, p95, p99, mean, std, min, max (мкс на операцию, 2 знака),
  p50 и p99 в нс на одно обращение, throughput (операций/с в фазе measure), отношение `go1.25/go1.23` по p50 и p99;
- таблица воспроизводимости: для каждой (сценарий, размер, версия) — p50 каждого прогона, разброс `(max-min)/median`, %;
- таблица объёмов: длительность теста, число операций в warmup / measure / teardown, GC-циклы за measure;
- таблица overhead инструмента по версиям.

`summary.json` — те же числа в иерархии `scenario -> size -> op -> version -> {...}`.

### 3.7. Makefile (дописать в конец)

```make
# ЛР3 ТИОПО: микробенчмарк map в Go 1.23 vs Go 1.25
.PHONY: lr3-bench lr3-bench-smoke lr3-report
lr3-bench:
	./perf/lr3-bench/run.sh

lr3-bench-smoke:
	RUNS=1 WARMUP=1s DURATION=2s TEARDOWN=1s SIZES="1000 100000" ./perf/lr3-bench/run.sh

lr3-report:
	docker run --rm -v $(abspath $(DIR)):/data goflow-perf-analysis:local python /app/lr3_report.py /data
```

---

## 4. ЛР4 — нагрузочное тестирование GoFlow

### 4.1. Инструменты (зафиксировано)

| Компонент        | Образ                                              |
|------------------|----------------------------------------------------|
| Генератор        | `grafana/k6:0.52.0`                                |
| Метрики          | `prom/prometheus:v2.53.0`                          |
| Контейнеры       | `gcr.io/cadvisor/cadvisor:v0.49.1`                 |
| Хост             | `prom/node-exporter:v1.8.1`                        |
| PostgreSQL       | `prometheuscommunity/postgres-exporter:v0.15.0`    |
| Визуализация     | `grafana/grafana:11.1.0`                           |
| БД / кеш         | `postgres:16-alpine`, `redis:7-alpine`             |

### 4.2. Разделение CPU (требование 12)

| Группа                   | Сервисы                                          | cpuset |
|--------------------------|--------------------------------------------------|--------|
| SUT: БД                  | postgres                                         | `0,1`  |
| SUT: приложение          | gateway, auth, accounts, transactions, redis     | `2,3`  |
| Генератор нагрузки       | k6                                               | `4,5`  |
| Мониторинг               | prometheus, grafana, cadvisor, node-exporter, postgres-exporter | `6,7` |

Задаётся через `cpuset:` в compose, значения из переменных с дефолтами (`${CPUS_DB:-0,1}` и т.д.).
Лимиты памяти (`mem_limit`): postgres 1g, каждый Go-сервис 256m, redis 128m, k6 1g, prometheus 512m, grafana 256m,
cadvisor 256m, node-exporter 64m, postgres-exporter 64m.

### 4.3. Docker-окружение

Общая внешняя сеть `goflow-load` (создаёт `run.sh`, если нет). Два compose-проекта:

**`compose.monitoring.yml`** (project `goflow-mon`, живёт между прогонами — чтобы история была в одном Prometheus):

- prometheus: `--config.file=/etc/prometheus/prometheus.yml --web.enable-remote-write-receiver
  --storage.tsdb.retention.time=7d`, порт `9090`, том `prom-data`.
- cadvisor: `privileged: true`, монтирования `/:/rootfs:ro`, `/var/run:/var/run:ro`, `/sys:/sys:ro`,
  `/var/lib/docker/:/var/lib/docker:ro`, `/dev/disk/:/dev/disk:ro`, `devices: [/dev/kmsg]`,
  аргументы `--docker_only=true --housekeeping_interval=5s --store_container_labels=true`.
- node-exporter: `pid: host`, `/:/host:ro,rslave`, `--path.rootfs=/host`.
- grafana: порт `3000`, анонимный доступ Admin (`GF_AUTH_ANONYMOUS_ENABLED=true`, `GF_AUTH_ANONYMOUS_ORG_ROLE=Admin`),
  provisioning из `grafana/`.

`prometheus.yml`: `scrape_interval: 5s`; jobs `cadvisor:8080`, `node-exporter:9100`, `postgres-exporter:9187`.

**`compose.sut.yml`** (project `goflow-load`, пересоздаётся на каждый прогон с `down -v`):

- postgres: `postgres:16-alpine`, `POSTGRES_PASSWORD=postgres`, init-скрипт `scripts/init-test-dbs.sql`
  монтируется в `/docker-entrypoint-initdb.d/` (БД `auth_test`, `account_test`, `transaction_test`), данные в
  именованном томе (не tmpfs — нужны реальные дисковые операции), флаги `-c shared_buffers=256MB -c max_connections=200`,
  healthcheck `pg_isready`.
- redis: `redis:7-alpine`, healthcheck `redis-cli ping`.
- auth, accounts, transactions, gateway: `build` из существующих `services/<name>/Dockerfile` (context `../..`),
  `image: goflow-load-<name>:local`, конфиги из `perf/lr4-load/config/*.yaml` монтируются read-only,
  `certs/` монтируется read-only, env как в `scripts/test_stand_up.sh`. `depends_on` с `condition: service_healthy`
  на postgres/redis. Наружу публикуется только gateway `8080`.
- postgres-exporter: `DATA_SOURCE_NAME=postgresql://postgres:postgres@postgres:5432/postgres?sslmode=disable`,
  сеть `goflow-load`, `container_name: postgres-exporter` (чтобы Prometheus находил его по имени).

Конфиги сервисов (`perf/lr4-load/config/`): структура как в `scripts/test_stand_up.sh`, но хосты — имена сервисов compose
(`postgres:5432`, `redis:6379`, `accounts:50052` и т.д.), `log.level: warn`, `log.output: stdout`, у accounts и transactions
`kafka.brokers: "kafka:9092"` (Kafka в стенде нет намеренно), `outbox.interval: 1h`, `recover.interval: 1h`,
`bank_gateway.failure_rate: 0.0`, `hash_cost: 10`. Если какой-то сервис не стартует без Kafka — остановиться и спросить.

### 4.4. Пользовательский сценарий k6 (требование 2)

`k6/lib/api.js` — по функции на эндпоинт: `register`, `login`, `logout`, `createAccount`, `linkBank`, `deposit`,
`listAccounts`, `balance`, `history`, `transfer`, `statement`. Каждая:

- принимает `baseUrl`, `token` (где нужно) и параметры;
- ставит тег `name` вида `'GET /accounts/:id/balance'` (чтобы URL с ID группировались) и тег `kind: 'normal'`;
- там, где тело ответа не нужно, — `responseType: 'none'`;
- делает `check` на ожидаемый статус; возвращает распарсенное тело или `null`.

`k6/lib/seed.js` — `seedCompanies(baseUrl, n)`: для каждой компании `register` -> 2 x `createAccount` -> `linkBank`
(к первому счёту) -> `deposit` 10 000 000 на каждый счёт. Все запросы сида с тегом `phase: 'setup'`.
Логин `load_<i>_<runId>`, пароль один на всех. Возвращает `[{email, password, accounts: [id1, id2], bankAccountId}]`.

`k6/main.js`:

- `setup()` — `seedCompanies(BASE_URL, SEED_COMPANIES)`, `setupTimeout: '15m'`.
- `default function(data)` — одна пользовательская сессия, компания `data[(__VU - 1) % data.length]`:

```
login                                          -> token
think()
listAccounts
think()
repeat ACTIONS_PER_SESSION (по умолчанию 3):
    action = weighted: balance 40, history 25, transfer 25, statement 5, deposit 5
    if Math.random() < ERROR_RATE: выполнить ошибочный вариант (4.5) вместо action
    else: выполнить action
    think()
logout
sleep(uniform(SESSION_PAUSE_MIN, SESSION_PAUSE_MAX))   // по умолчанию 2..5 с
```

`think()` = `sleep(uniform(THINK_MIN, THINK_MAX))`, по умолчанию 1..3 с. `transfer`: со своего случайного счёта
на случайный счёт другой компании из `data`, сумма 1..100, валюта RUB, `idempotency_key` — случайный UUID v4
(своя функция на `crypto.randomUUID` если доступен в k6 0.52, иначе на `Math.random`). `statement`: период — последние сутки.
`deposit`: 1000 на свой счёт через свой `bankAccountId`.

### 4.5. Ошибочные операции (требование 4)

`k6/lib/errors.js` — функция `doErroneous(ctx)` выбирает равновероятно один из вариантов:

| Вариант             | Что делаем                                                  | Ожидаемый статус |
|---------------------|-------------------------------------------------------------|------------------|
| `wrong_password`    | login с неверным паролем                                    | 401              |
| `foreign_balance`   | balance случайного несуществующего UUID                     | 404              |
| `bad_token`         | balance своего счёта с токеном `Bearer garbage`             | 401              |
| `invalid_body`      | transfer с `from_account_id: "not-a-uuid"`                  | 400              |
| `overdraft`         | transfer на сумму 10^15                                     | см. ниже         |

Ожидаемые статусы в таблице — гипотеза. Исполнитель обязан проверить каждый вариант smoke-прогоном против стенда,
зафиксировать **фактический** статус в константе `EXPECTED` в `errors.js` и перечислить расхождения в итоговом отчёте.
Если `overdraft` возвращает 200 с `TRANSACTION_STATUS_FAILED` — фиксировать как есть (200 + check на статус транзакции).

Каждый ошибочный запрос: тег `kind: 'error'`, `tags.variant`, параметр
`responseCallback: http.expectedStatuses(EXPECTED[variant])` — чтобы ожидаемый отказ не считался провалом
в `http_req_failed`. Реальные отказы = неожиданные статусы в любых запросах.

### 4.6. Профили нагрузки (требование 3)

`profiles.env` — все числа здесь, `run.sh` передаёт их в k6 через `-e`:

```
SEED_COMPANIES=200
THINK_MIN=1
THINK_MAX=3
ACTIONS_PER_SESSION=3

RAMP_START_VUS=10
RAMP_STEP_VUS=20
RAMP_STEPS=15
RAMP_STEP_DURATION=60s

NORMAL_VUS=50
STRESS_VUS=300
STRESS_BASELINE=5m
STRESS_SPIKE_RAMP=30s
STRESS_HOLD=3m
STRESS_RECOVERY=10m

SOAK_VUS=50
SOAK_DURATION=60m
```

`NORMAL_VUS`, `STRESS_VUS`, `SOAK_VUS` — заглушки; реальные значения пользователь выставит по результатам `ramp`.

`main.js` строит `options.scenarios` по `__ENV.SCENARIO`, executor `ramping-vus`, `gracefulRampDown: '30s'`:

- `ramp` — ступени: с `RAMP_START_VUS`, `RAMP_STEPS` ступеней по `+RAMP_STEP_VUS`; каждая ступень = подъём за 10 с
  + полка `RAMP_STEP_DURATION`. Threshold `http_req_failed: [{threshold: 'rate<0.5', abortOnFail: true, delayAbortEval: '1m'}]`
  — система заведомо уронена, дальше гнать смысла нет.
- `stress` — `NORMAL_VUS` в течение `STRESS_BASELINE` -> подъём до `STRESS_VUS` за `STRESS_SPIKE_RAMP` -> полка `STRESS_HOLD`
  -> спуск до `NORMAL_VUS` за 10 с -> `NORMAL_VUS` в течение `STRESS_RECOVERY`.
- `soak` — `SOAK_VUS` постоянно `SOAK_DURATION` (подъём 30 с).
- `smoke` — 5 VU, 1 мин, `SEED_COMPANIES` переопределяется в 10. Только для проверки.

Переменная `ERROR_RATE` (0 / 0.1 / 0.2) — независима от сценария. Требуемые запуски для отчёта:
`ramp`, `stress`, `soak` x `ERROR_RATE` = 0, 0.1, 0.2.

`handleSummary(data)` — пишет `/results/summary.json` (стандартный `JSON.stringify(data)`) и текстовую сводку в stdout
(`textSummary` из `https://jslib.k6.io/k6-summary/0.0.2/index.js`). Тренды: `summaryTrendStats: ['min','avg','med','max','p(50)','p(75)','p(90)','p(95)','p(99)']`.

### 4.7. `run.sh`

Использование: `./perf/lr4-load/run.sh <ramp|stress|soak|smoke> [ERROR_RATE]`.

1. `source profiles.env`. Сгенерировать `certs/private.pem`/`public.pem`, если их нет (как в `scripts/test_stand_up.sh`).
2. Создать сеть `goflow-load`, если нет. Поднять мониторинг: `docker compose -p goflow-mon -f compose.monitoring.yml up -d`.
3. Свежий стенд: `docker compose -p goflow-load -f compose.sut.yml down -v`, затем `up -d --build --wait`.
   Дождаться `200` от `http://127.0.0.1:8080/swagger` (до 60 попыток по 1 с).
4. `OUT=perf/results/lr4/<scenario>-err<rate>-<YYYYmmdd-HHMMSS>`. Пауза 15 с (фон до нагрузки).
5. Записать `start` (unix-секунды). Запустить k6:

```
docker run --rm --network goflow-load --cpuset-cpus "${CPUS_LOADGEN:-4,5}" --memory 1g
  -v <repo>/perf/lr4-load/k6:/scripts:ro -v <OUT>:/results
  -e K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write
  -e K6_PROMETHEUS_RW_TREND_STATS="p(50),p(75),p(90),p(95),p(99),min,max,avg"
  -e K6_PROMETHEUS_RW_PUSH_INTERVAL=5s
  -e K6_CSV_TIME_FORMAT=unix_milli
  -e BASE_URL=http://gateway:8080 -e SCENARIO=... -e ERROR_RATE=... (+ все из profiles.env)
  grafana/k6:0.52.0 run --tag testid=<имя OUT>
    --out experimental-prometheus-rw --out csv=/results/k6.csv.gz /scripts/main.js
```

   Если `K6_CSV_TIME_FORMAT=unix_milli` в 0.52.0 не поддерживается — проверить по документации k6 и выбрать формат
   с миллисекундами, отметить в отчёте.
6. Записать `end`. Пауза 30 с (видно восстановление ресурсов после нагрузки).
7. `run.json`: `{scenario, error_rate, start, end, profile: {все переменные profiles.env}, git_commit}`.
8. Логи SUT: `docker compose -p goflow-load -f compose.sut.yml logs > $OUT/sut.log`. Стенд погасить `down -v`.
9. Запустить `lr4_report.py` в контейнере анализа с `--network goflow-load`, `--prometheus http://prometheus:9090`.

Мониторинг между прогонами не гасится. Для остановки — цель `lr4-mon-down`.

### 4.8. Отчёт ЛР4 (`perf/analysis/lr4_report.py`)

**Устарело, заменено разделом 6.**

Аргументы: `<run_dir>`, `--prometheus URL`. Читает `run.json`, `k6.csv.gz` (чанками через `pandas.read_csv(chunksize=...)`,
оставляя только нужные метрики; строки с тегом `phase=setup` отбрасываются).

Латентность (`http_req_duration`), отдельно для `kind=normal` и `kind=error`, плюс по каждому `name`:

- `figures/latency_timeseries.png` — точки + скользящие p50/p99 (окно 10 с); для `stress` закрашены фазы baseline/spike/recovery;
- `figures/latency_percentiles.png` — кривая перцентилей (normal и error на одном графике);
- `figures/latency_histogram.png`;
- `figures/latency_by_endpoint.png` — p50/p95/p99 по каждому `name` (горизонтальные столбцы).

Пропускная способность:

- RPS по секундам (все запросы) и бизнес-TPS (успешные `transfer` в секунду), график во времени
  вместе с числом VU (`vus`) на второй оси — `figures/throughput.png`;
- распределение посекундного RPS: p50/p75/p90/p95/p99, min, max, mean — не только среднее (требование 7);
- error rate (доля неожиданных статусов) по секундам — `figures/errors.png`.

Ресурсы (Prometheus `query_range`, шаг 5 с, окно `[start-15s, end+30s]`), компонент = лейбл
`container_label_com_docker_compose_service` (для postgres-exporter и мониторинга — тоже, чтобы видеть их затраты):

| Метрика           | PromQL                                                                                 |
|-------------------|----------------------------------------------------------------------------------------|
| CPU, ядер         | `sum by (svc) (rate(container_cpu_usage_seconds_total{...}[15s]))`                    |
| RAM, байт         | `sum by (svc) (container_memory_working_set_bytes{...})`                              |
| Диск чтение, B/s  | `sum by (svc) (rate(container_blkio_device_usage_total{operation="Read"}[15s]))`      |
| Диск запись, B/s  | `sum by (svc) (rate(container_blkio_device_usage_total{operation="Write"}[15s]))`     |
| Сеть rx/tx, B/s   | `rate(container_network_receive_bytes_total[15s])` / `..._transmit_...`               |
| PostgreSQL TPS    | `sum(rate(pg_stat_database_xact_commit[15s]))`                                         |

`svc` получить через `label_replace` из `container_label_com_docker_compose_service`. Если blkio-метрики
в cgroup v2 пустые — проверить в UI Prometheus, какие дисковые метрики отдаёт cAdvisor, использовать их и указать в отчёте.

Выход:

- `resources.json` — `{ "<component>": { "cpu_cores": {min,max,mean,median,p95}, "ram_bytes": {...},
  "disk_read_Bps": {...}, "disk_write_Bps": {...}, "net_rx_Bps": {...}, "net_tx_Bps": {...} } }` (требование 8);
- `figures/resources_cpu.png`, `resources_ram.png`, `resources_disk.png`, `resources_net.png`, `pg_tps.png` —
  линия на компонент, ось времени общая с латентностью;
- `summary.md`: параметры прогона; таблица перцентилей латентности (общая, normal, error, по эндпоинтам);
  таблица RPS/TPS; число запросов всего / неожиданных отказов / ожидаемых ошибок; таблица ресурсов
  в формате «компонент — min/max/mean/median CPU, RAM, диск»;
  - для `ramp`: таблица по ступеням (ступень = интервал постоянного числа VU): VU, медианный RPS, p95, p99, error rate,
    CPU каждой группы. Строка «стабильная работа до N VU» = последняя ступень с p95 < 500 мс и error rate < 1%;
    «отказ с N VU» = первая ступень с error rate > 5% или RPS ниже предыдущей ступени;
  - для `stress`: базовые p95 и error rate (последние 2 мин baseline); **время восстановления** = время от окончания
    спуска до первого момента, после которого 30 с подряд скользящий (10 с) p95 <= 1.2 x базового и error rate <= базовый + 1%.
    Если не восстановилась до конца прогона — так и написать.
- `summary.json` — те же числа.

Grafana-дашборд `goflow-load.json` — для живого наблюдения на защите, 8 панелей: VUs, RPS, p95/p99 латентности
(из k6 remote write, фильтр по `testid`), error rate, CPU по `svc`, RAM по `svc`, диск по `svc`, PG TPS.
Переменная дашборда `testid`.

### 4.9. Makefile (дописать в конец)

```make
# ЛР4 ТИОПО: нагрузочное тестирование
.PHONY: lr4-smoke lr4-ramp lr4-stress lr4-soak lr4-soak-all lr4-mon-down lr4-report
lr4-smoke:
	./perf/lr4-load/run.sh smoke 0.1

lr4-ramp:
	./perf/lr4-load/run.sh ramp 0

lr4-stress:
	./perf/lr4-load/run.sh stress 0

lr4-soak:
	./perf/lr4-load/run.sh soak $(or $(ERROR_RATE),0)

lr4-soak-all:
	./perf/lr4-load/run.sh soak 0
	./perf/lr4-load/run.sh soak 0.1
	./perf/lr4-load/run.sh soak 0.2

lr4-mon-down:
	docker compose -p goflow-mon -f perf/lr4-load/compose.monitoring.yml down

lr4-report:
	docker run --rm --network goflow-load -v $(abspath $(DIR)):/data goflow-perf-analysis:local \
		python /app/lr4_report.py /data --prometheus http://prometheus:9090
```

---

## 5. Этапы и Definition of Done

| # | Этап | Проверка (должна проходить) |
|---|------|-----------------------------|
| 1 | `perf/analysis/` (Dockerfile, plots.py) | `docker build -t goflow-perf-analysis:local perf/analysis` |
| 2 | ЛР3: модуль, harness, calibrate, scenario, main | `cd perf/lr3-bench && GOWORK=off go vet ./... && GOWORK=off go build ./...` |
| 3 | ЛР3: Dockerfile, run.sh, lr3_report.py, Makefile | `make lr3-bench-smoke` завершается с кодом 0; в результатах есть `samples.csv`, `meta.json` на все 22 комбинации (2 версии x (5 сценариев x 2 размера + grow)); в `meta.json` у образа go1.23 `go_version` начинается с `go1.23`, у go1.25 — с `go1.25`, `summary.md` и все PNG; `loop_per_iter_ns` < 5 |
| 4 | ЛР4: compose-файлы, конфиги, prometheus, grafana | `docker compose -f ... config` без ошибок для обоих файлов; стенд поднимается, `curl 127.0.0.1:8080/swagger` -> 200; в Prometheus targets все `up` |
| 5 | ЛР4: k6-скрипты | `make lr4-smoke` проходит; фактические статусы ошибочных операций зафиксированы в `EXPECTED` |
| 6 | ЛР4: lr4_report.py, Makefile | после smoke есть `summary.md`, `summary.json`, `resources.json` с ненулевыми CPU/RAM для postgres и gateway, все PNG |

Для этапа 6 разрешено временно уменьшить длительности в `profiles.env` через переменные окружения, но не менять дефолты.

Итоговый ответ исполнителя: список файлов, вывод проверок по этапам (кратко), фактические статусы из `EXPECTED`,
открытые вопросы. Без пересказа контракта и без маркетинга.

---

## 6. Изменение: результаты обеих ЛР в Grafana

Результаты показываются на защите в Grafana: два дашборда, «ЛР3» и «ЛР4».
PNG-графики, `summary.md` и `summary.json` больше не нужны. Python-анализ (`perf/analysis/`) удаляется целиком.
Разделы 2.1, 3.6 и 4.8 не действуют. Где раздел 6 противоречит разделам 2–5, прав раздел 6.

Prometheus не годится для сырых выборок ЛР3 (миллионы точек с прошлыми метками времени) и для точных
перцентилей ЛР4 за весь тест (из k6 remote write приходят перцентили по окнам 5 с, и усреднять их нельзя).
Поэтому сырые замеры обеих ЛР загружаются в отдельный PostgreSQL `results-db`, а Grafana читает его
встроенным источником PostgreSQL. Prometheus остаётся источником живых графиков ЛР4 и ресурсов.

### 6.1. Новая структура

```
perf/
  monitoring/                  было lr4-load/compose.monitoring.yml и lr4-load/{prometheus,grafana}
    compose.yml                проект goflow-mon, сеть goflow-load (external)
    prometheus/prometheus.yml
    results-db/schema.sql
    grafana/provisioning/datasources/datasources.yml   Prometheus + ResultsDB
    grafana/provisioning/dashboards/dashboards.yml
    grafana/dashboards/lr3-maps.json
    grafana/dashboards/lr4-load.json                   было goflow-load.json
  lr3-bench/
    load.sh                    загрузка одного прогона (samples.csv + meta.json) в results-db
  lr4-load/
    load.sh                    загрузка k6.csv.gz + run.json в results-db
    resources.py               resources.json из Prometheus, только stdlib
```

Удалить: `perf/analysis/` целиком, `lr4-load/compose.monitoring.yml`, `lr4-load/prometheus/`, `lr4-load/grafana/`.
Ссылки на старые пути в `run.sh` и Makefile поправить.

### 6.2. `results-db`

Сервис в `monitoring/compose.yml`:

- `postgres:16.4-alpine`, `POSTGRES_DB=results`, `POSTGRES_USER=results`, `POSTGRES_PASSWORD=results`;
- `cpuset: ${CPUS_MON:-6,7}`, `mem_limit: 1g`, именованный том `results-data` (данные переживают `down`, не `down -v`);
- `schema.sql` монтируется в `/docker-entrypoint-initdb.d/`; все `CREATE` с `IF NOT EXISTS`;
- порт наружу не публиковать: загрузка идёт через `docker compose exec -T results-db psql ...`.

Retention Prometheus поднять до `30d`: защита может быть позже чем через неделю после прогонов.

Схема (имена фиксированы, типы на усмотрение, если не указаны):

```sql
lr3_runs(
  id serial primary key,
  bench_id text,           -- имя директории прогона ЛР3, например 20261022-222341
  version text,            -- go1.23 / go1.25 (тег образа, не meta.go_version)
  scenario text, size int, run int,
  meta jsonb,              -- meta.json целиком
  unique (bench_id, version, scenario, size, run)
)
lr3_samples(
  run_id int references lr3_runs on delete cascade,
  t_ms double precision,   -- start_ns / 1e6, отсчёт от начала прогона
  latency_ns bigint, op text, phase text
)
index on lr3_samples (run_id, phase, op)

lr4_runs(
  testid text primary key, -- имя директории прогона ЛР4
  scenario text, error_rate real,
  started_at timestamptz, finished_at timestamptz,
  meta jsonb               -- run.json целиком
)
lr4_samples(
  testid text references lr4_runs on delete cascade,
  ts timestamptz,
  metric text,             -- только http_req_duration и vus
  value double precision,  -- мс для латентности, число VU для vus
  name text, kind text, phase text, status int, expected boolean
)
index on lr4_samples (testid, metric, ts)
```

Повторная загрузка того же прогона заменяет его (`delete ... where`, затем вставка) в одной транзакции.

### 6.3. Загрузка ЛР3 (`lr3-bench/load.sh <bench_dir>`)

- Обходит `<bench_dir>/<version>/<scenario>/<size>/run-<n>/` и для каждого грузит `meta.json` и `samples.csv`.
- Только `psql` внутри `results-db`: `\copy ... from stdin` из `samples.csv` во временную таблицу, затем
  `insert ... select` с `run_id`. Файл подаётся через stdin `docker compose exec -T`. Python и прочие образы не нужны.
- Вызывается из `lr3-bench/run.sh` один раз после всех прогонов вместо `lr3_report.py`. В начале `run.sh`
  поднимает мониторинг (`docker compose -p goflow-mon -f perf/monitoring/compose.yml up -d --wait`),
  потому что бенч-контейнеры работают с `--network none` и мониторинг им не мешает.
- В конце печатает ссылку: `http://localhost:3000/d/lr3-maps?var-bench=<bench_id>`.
- Отдельная цель `make lr3-load DIR=perf/results/lr3/<...>` для повторной загрузки.

### 6.4. Загрузка ЛР4 (`lr4-load/load.sh <run_dir>`)

- Из `k6.csv.gz` брать только строки `metric_name in ('http_req_duration','vus')` и не из фазы `setup`.
  Пользовательские теги (`kind`, `phase`, `variant`, `testid`) k6 пишет в колонку `extra_tags`
  в виде `k=v&k=v`. Перед реализацией посмотреть заголовок и пару строк реального файла после `make lr4-smoke`,
  разбор делать по факту.
- Распаковка `gunzip -c` на хосте, поток в stdin `psql` (`\copy ... from stdin` в staging-таблицу со всеми колонками CSV,
  затем `insert ... select` с разбором `extra_tags`).
- `expected` = `expected_response` из CSV (k6 ставит его по `responseCallback`).
- `lr4_runs`: из `run.json` (`start`/`end` -> `started_at`/`finished_at`, всё целиком -> `meta`).
- Вызывается из `lr4-load/run.sh` вместо `lr4_report.py`, затем `resources.py`.
- В конце `run.sh` печатает ссылку
  `http://localhost:3000/d/lr4-load?var-testid=<testid>&from=<(start-15)*1000>&to=<(end+30)*1000>`.
- Отдельная цель `make lr4-load DIR=...`.

### 6.5. `lr4-load/resources.py <run_dir>`

Требование 8 ЛР4 просит JSON «компонент — min/max/среднее». Скрипт на stdlib (`urllib`, `json`),
запускается на хосте (`python3`, как `run.json` в `run.sh`), Prometheus на `http://127.0.0.1:9090`.
Окно `[start-15s, end+30s]` из `run.json`, шаг 5 с. Запросы — таблица из раздела 4.8 (CPU, RAM, диск, сеть по `svc`).
Выход `resources.json` в формате из 4.8: `{component: {cpu_cores: {min,max,mean,median,p95}, ram_bytes: ..., ...}}`.
Скрипт до 100 строк.

### 6.6. Grafana

- `datasources.yml`: Prometheus (uid `prometheus`, как сейчас) и PostgreSQL (uid `resultsdb`, `results-db:5432`, база `results`).
- Оба дашборда в одной папке, `editable: true`, время по умолчанию не важно для ЛР3 (все панели ЛР3 игнорируют
  диапазон времени дашборда, ось X — секунды от начала прогона).
- Заголовки панелей на русском, единицы заданы через `unit` (ns, µs, ms, reqps, percentunit, bytes, Bps).
- SQL в панелях без лишних CTE. Перцентили — `percentile_cont(array[...]) within group (order by ...)`.
- Число точек на графиках времени ограничивать агрегацией в SQL (бакеты), не отдавать в Grafana сотни тысяч строк.

#### Дашборд ЛР3 `lr3-maps` («ЛР3: map Go 1.23 vs Go 1.25»)

Переменные: `bench` (последний по умолчанию), `scenario`, `size`, `op` (зависит от scenario), `run` (All или 1..N).
Обе версии всегда на одном графике, разными цветами (go1.23 и go1.25 — фиксированные цвета через overrides).
Латентность в мкс на операцию (одна операция = батч 1000 обращений, раздел 3), в сводке ещё и нс на обращение.

Ряд «Сводка» (переменные scenario/size/op не учитываются):

1. Таблица по всем (scenario, size, op): p50 и p99 для go1.23 и go1.25, отношение go1.25/go1.23 по p50 и p99
   (цвет ячейки: < 1 зелёный, > 1 красный), нс на обращение. Только фаза measure, все run вместе.
2. Столбцы: p50 на обращение по сценариям для size=$size, две версии рядом.

Ряд «Сценарий» (для выбранных scenario/size/op/run):

3. Латентность во времени: все три фазы, X — секунды от начала, бакет 50 мс, линии p50 и max по бакету
   для каждой версии; границы фаз — пороги (thresholds) или отдельные серии по `meta.warmup_ns` и т. д.
   Панель `trend` (числовая ось X) или `xychart`.
4. Распределение по перцентилям: X — перцентиль (0, 10, 20, ..., 90, 95, 99, 99.9, 99.99), Y — латентность, две версии.
5. Гистограмма: 100 бинов `width_bucket` между общим p0.1 и p99.9 обеих версий, панель `barchart`, две версии.
6. Таблица перцентилей: версия, run, p50, p75, p90, p95, p99, min, max, mean, std, число операций.
7. Таблица объёмов: версия, run, длительность теста (`meta.total_duration_ns`), число операций в warmup / measure /
   teardown (по `meta.counts`), GC-циклы за measure.
8. Таблица overhead инструмента: версия, `timer_p50_ns`, `harness_noop_p50_ns`, `loop_per_iter_ns` (медиана по run).

Сводная таблица (1) на полном прогоне считает перцентили по десяткам миллионов строк. Если панель грузится
дольше 5 с, `load.sh` после загрузки заполняет таблицу `lr3_stats` (перцентили фазы measure на каждый run_id),
и панель 1 читает её. Сделать это, если замер покажет необходимость; время загрузки панели указать в ответе.

#### Дашборд ЛР4 `lr4-load` («ЛР4: нагрузочное тестирование GoFlow»)

Переменная `testid` из `lr4_runs` (последний по умолчанию). Панели из Prometheus фильтруются по `testid`
(метрики k6) и по времени дашборда (ресурсы). Панели из results-db фильтруются только по `testid`.

Ряд «Нагрузка» (Prometheus, живое наблюдение — существующие панели, переделать):

1. VU; 2. RPS; 3. латентность во времени p50/p75/p90/p95/p99 (все пять трендов из remote write);
4. error rate (доля `expected="false"`).

Ряд «Латентность за весь тест» (results-db, точные значения):

5. Распределение по перцентилям, normal и error отдельно (как панель 4 ЛР3).
6. Гистограмма латентности normal и error (как панель 5 ЛР3, бины по ms).
7. Таблица перцентилей: срез (все / normal / error / каждый `name`), p50, p75, p90, p95, p99, min, max, mean, число запросов.
8. Таблица по фазам `phase` (для stress — baseline/spike/hold/recovery, для ramp — ступени по числу VU из `vus`):
   длительность, VU, медианный RPS, p95, p99, error rate.

Ряд «Пропускная способность» (results-db):

9. RPS по секундам во времени вместе с VU на второй оси.
10. Распределение посекундного RPS: p50, p75, p90, p95, p99, min, max, mean (требование «не только среднее»).
11. Число запросов: всего / неожиданных отказов / ожидаемых ошибок; число 500 на overdraft отдельно (раздел 4.5).

Ряд «Ресурсы» (Prometheus, `svc` через `label_replace`, как в 4.8):

12. CPU по компонентам; 13. RAM; 14. диск чтение/запись; 15. сеть rx/tx; 16. PostgreSQL TPS.
17. Таблица «компонент — min / max / среднее» CPU, RAM, диск за диапазон дашборда
    (`min_over_time`/`max_over_time`/`avg_over_time` по `[$__range]`, instant-запросы, transform «Join by field»).

### 6.7. Makefile

Удалить цели `lr3-report`, `lr4-report`. `lr4-mon-down` переименовать в `mon-down` с новым путём.
Добавить `lr3-load`, `lr4-load` (аргумент `DIR`), `mon-up`. Остальные цели не трогать.
Чужие цели пользователя выше блока ЛР3 не трогать.

### 6.8. Definition of Done раздела 6

| # | Проверка |
|---|----------|
| 1 | `docker compose -f perf/monitoring/compose.yml config` без ошибок; `make mon-up` поднимает всё, в Grafana два дашборда и два источника, оба источника проходят «Save & test» |
| 2 | `make lr3-bench-smoke` завершается с кодом 0, печатает ссылку; на дашборде ЛР3 все 8 панелей с данными для любого scenario/size; `make lr3-load DIR=<тот же прогон>` не дублирует строки (число строк до и после совпадает) |
| 3 | `make lr4-smoke` завершается с кодом 0, печатает ссылку; на дашборде ЛР4 все 17 панелей с данными; `resources.json` с ненулевыми CPU/RAM у postgres и gateway |
| 4 | `grep -rn "analysis\|lr3_report\|lr4_report\|compose.monitoring" perf Makefile` ничего не находит (кроме CONTRACT.md) |
| 5 | Время загрузки дашбордов: ни одна панель не дольше 5 с на smoke-данных (указать в ответе) |

Итоговый ответ: список изменённых, созданных и удалённых файлов, вывод проверок, фактический формат `extra_tags`,
время загрузки панелей, открытые вопросы.
