# authStartGrowingUp

Репозиторий для auth-сервиса StartGrowingUp. Сейчас это минимальный Go bootstrap с базовой инфраструктурой качества: structured logging, тесты, `go vet`, сборка и CI в GitHub Actions.

## Текущее состояние

- Кодовая база находится на раннем этапе и пока содержит только стартовый entrypoint.
- Основная точка входа расположена в `cmd/main.go`.
- При запуске сервис пишет структурированный стартовый лог через production-ready `zap`-обертку.
- В репозитории уже настроены правила разработки и PR-процесс.

## Структура проекта

- `cmd` - текущий entrypoint приложения.
- `pkg/logger` - единая production-ready обертка над `zap` с typed fields и безопасными defaults.
- `internal/config` - сборка runtime-конфига из env-переменных.
- `.github/workflows/ci.yaml` - CI-пайплайн для `test`, `vet` и `build`.
- `.github/PULL_REQUEST_TEMPLATE.md` - шаблон описания pull request.
- `CONTRIBUTING.md` - правила работы с репозиторием.
- `STYLE.md` - соглашения по стилю кода и качеству.

## Требования

- Go `1.26.4`

## Локальный запуск

```bash
go run ./cmd
```

Ожидаемый вывод:

```text
2026-07-14T12:00:00.000+0300    info    cmd/main.go:27  auth service bootstrap is running {"service":"authStartGrowingUp","environment":"development","action":"service_start","result":"success"}
```

## Логирование

Сервис использует единый логгер в `pkg/logger`:

- structured fields для `service`, `environment`, `request_id`, `actor_id`, `action`, `result`;
- безопасные defaults для `development` и `production`;
- потокобезопасную запись для конкурентной нагрузки;
- конфигурацию через `.env` и переменные окружения.

Основные env-переменные:

- `APP_NAME`
- `APP_ENV`
- `LOG_LEVEL`
- `LOG_FORMAT`
- `LOG_OUTPUT_PATHS`
- `LOG_ERROR_OUTPUT_PATHS`
- `LOG_DISABLE_CALLER`
- `LOG_DISABLE_STACKTRACE`
- `LOG_SAMPLING_INITIAL`
- `LOG_SAMPLING_THEREAFTER`

## Проверки качества

Локально перед PR достаточно прогнать:

```bash
go test ./...
go vet ./...
mkdir -p .tmp && go build -o .tmp/authStartGrowingUp ./cmd
```

## CI

Workflow в `.github/workflows/ci.yaml` запускается на `push` в `main` и `master`, а также на каждый `pull_request`.

Пайплайн выполняет:

- `go test ./...`
- `go vet ./...`
- `mkdir -p .tmp && go build -o .tmp/authStartGrowingUp ./cmd`

## Roadmap

Дальше сюда можно наращивать реальную auth-логику:

- регистрация и вход,
- работа с токенами и сессиями,
- верификация контактов,
- аудит и security-события,
- интеграции с БД и внешними провайдерами.

## Лицензия

Проект распространяется на условиях, описанных в [LICENSE](LICENSE).
