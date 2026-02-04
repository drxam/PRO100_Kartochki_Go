# МозгоЁмка — Backend

REST API для приложения карточек «PRO100_Kartochki» на Go (Clean Architecture).

## Требования

- Go 1.21+
- PostgreSQL

## Структура проекта

```
/cmd/api          — точка входа
/internal
  /domain         — модели данных
  /handler        — HTTP-обработчики
  /service        — бизнес-логика
  /repository     — работа с БД
  /middleware     — auth, logging
  /config         — конфигурация
/pkg
  /jwt            — JWT (Access + Refresh)
  /validator      — валидация
/migrations       — SQL-миграции
/docs             — Swagger/OpenAPI
```

## Запуск

1. Создайте БД и примените миграции:

```bash
psql -U postgres -c "CREATE DATABASE PRO100_Kartochki;"
psql -U postgres -d PRO100_Kartochki -f migrations/001_init.up.sql
```

2. Переменные окружения (опционально):

```bash
export PORT=8080
export DATABASE_DSN="postgres://user:pass@localhost:5432/PRO100_Kartochki?sslmode=disable"
export JWT_ACCESS_SECRET="your-access-secret"
export JWT_REFRESH_SECRET="your-refresh-secret"
```

3. Запуск API:

```bash
go run ./cmd/api
```

4. Документация Swagger: http://localhost:8080/swagger/index.html

## Обновление Swagger

После изменения аннотаций в handlers:

```bash
swag init -g cmd/api/main.go --parseDependency --parseInternal
```

## API (кратко)

- **Auth:** `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`
- **Users:** `GET /api/v1/users/me`
- **Categories:** `GET/POST /api/v1/categories`
- **Tags:** `GET/POST /api/v1/tags`
- **Decks:** `GET/POST /api/v1/decks`, `GET/PUT/DELETE /api/v1/decks/:id`, `GET /api/v1/decks/public`
- **Cards:** `GET/POST /api/v1/decks/:deck_id/cards`, `GET/PUT/DELETE /api/v1/cards/:id`

Защищённые маршруты требуют заголовок: `Authorization: Bearer <access_token>`.
