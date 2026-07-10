# authStartGrowingUp

Репозиторий для auth-сервиса StartGrowingUp. Сейчас это минимальный Go bootstrap с базовой инфраструктурой качества: тесты, `go vet`, сборка и CI в GitHub Actions.

## Текущее состояние

- Кодовая база находится на раннем этапе и пока содержит только стартовый entrypoint.
- Основная точка входа расположена в `cmd/main.go`.
- При запуске сервис печатает стартовое сообщение.
- В репозитории уже настроены правила разработки и PR-процесс.

## Структура проекта

- `cmd` - текущий entrypoint приложения.
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
auth service bootstrap is running
```

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
