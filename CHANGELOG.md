# Changelog

Все значимые изменения в проекте фиксируются в этом файле.

Формат основан на принципах Keep a Changelog.

## [Unreleased]

### Added

- Добавлен файл `CONTRIBUTING.md` с правилами по веткам, коммитам, PR, ревью и security disclosure.
- Добавлен файл `STYLE.md` с едиными стандартами по стилю кода, обработке ошибок, API-контрактам, безопасности, логированию и тестам.
- Добавлен workflow CI с шагами `go test`, `go vet` и сборкой entrypoint `./cmd`.
- Добавлен шаблон Pull Request.
- Добавлен минимальный тест для текущего entrypoint.
- Добавлен пакет `errs` с централизованной системой ошибок:
  - Базовый тип `AppError` с кодами, обёрткой и контекстными полями.
  - Тип `ValidationErrors` для агрегации ошибок валидации.
  - Конструкторы ошибок: `NewInternalError`, `NewValidationError`, `NewWrongPasswordError`, `NewInvalidTokenError`, `NewExpiredTokenError`.
  - Методы `WithField`, `WithValue`, `WithFieldValue` для иммутабельного добавления контекста.
- Добавлена доменная модель `User`:
  - Создание через `CreateUser` с валидацией и хешированием пароля.
  - Нормализация полей (trim, lowercase email).
  - Проверка пароля через `CheckPassword` с возвратом `WrongPasswordError` или `InternalError`.
- Добавлена доменная модель `RefreshToken`:
  - Создание через `NewRefreshToken` с UUIDv7 и валидацией.
  - Проверка срока с левериджем 5 секунд.
  - Метод `Check` для целостности и срока.
  - Метод `Verify` для безопасного сравнения токенов (константное время).
- Добавлены юнит-тесты для пакета `errs`, моделей `User` и `RefreshToken`.

### Changed

- `README.md` синхронизирован с текущим bootstrap-состоянием проекта и реальным entrypoint.
- `CONTRIBUTING.md` дополнен актуальными командами локальной проверки.

### Fixed

- Исправлена сериализация `RefreshToken` в JSON (поле `Token` теперь скрыто через `json:"-"`).
