# PRO100_Kartochki — Backend

REST API для образовательной платформы «PRO100_Карточки» на Go (Clean Architecture).
Этот репозиторий содержит модуль **«Пользователи и доступ»** (ТЗ §4.1):
регистрация, JWT (access + refresh), роли и RBAC, восстановление пароля,
блокировка/удаление учёток, rate limiting, audit-лог, опциональный TLS.

## Требования

- **Go 1.25+** (зафиксировано в `go.mod`)
- **PostgreSQL 14+** (рекомендуется 16-alpine)
- Для запуска одной командой — Docker + docker compose

## Быстрый старт (Docker compose)

Проще всего проверяющему — `docker compose up --build`:

```bash
docker compose up --build
```

Это поднимет:

1. **postgres** — БД с healthcheck;
2. **migrate** — одноразово применит все `migrations/*.up.sql` (идемпотентно);
3. **api** — соберётся из `Dockerfile`, дождётся миграций и стартует на `:8080`.

Сразу после запуска доступен админ-аккаунт `admin@example.com / admin1234`
(см. переменные `BOOTSTRAP_ADMIN_*` в `docker-compose.yml`). Health-check:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Swagger UI: <http://localhost:8080/swagger/index.html>

## Запуск без docker

1. Поднять PostgreSQL и создать БД `mozgoemka`:

   ```bash
   psql -U postgres -c "CREATE DATABASE mozgoemka;"
   ```

2. Применить миграции (по порядку):

   ```bash
   make migrate-up
   ```

3. Скопировать `.env.example` → `.env`, при необходимости поправить значения,
   и запустить API:

   ```bash
   make run
   ```

## Структура проекта

```
/cmd/api               — точка входа, bootstrap admin, фоновая чистка токенов
/internal
  /domain              — модели и DTO
  /handler             — HTTP-обработчики, audit-лог, response-хелперы
  /service             — бизнес-логика (AuthService, AdminService, UserService)
  /repository          — pgx-репозитории, фильтрация мягко удалённых
  /middleware          — Auth (с RBAC и token_version), CORS, rate limit, RequestID, logging
  /config              — чтение env, дефолты
/pkg
  /jwt                 — JWT (HS256, jti, token_version)
  /validator           — кастомные теги email/password
/migrations            — SQL-миграции (001…004)
/docs                  — Swagger/OpenAPI (генерируется `make swagger`)
```

## Конфигурация (env)

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `SERVER_PORT` | `8080` | Порт HTTP/HTTPS-сервера |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE` | `localhost / 5432 / postgres / postgres / mozgoemka / disable` | Реквизиты Postgres (или используйте `DATABASE_DSN` целиком) |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | — | Секреты подписи (обязательно поменять в проде) |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | `15m` / `720h` | Время жизни токенов |
| `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD` | пусто | Если заданы — при старте создаётся/повышается админ |
| `PASSWORD_RESET_RETURN_TOKEN` | `false` | dev-режим: возвращать reset-токен в JSON-ответе `forgot-password` |
| `RATE_LIMIT_GLOBAL_RPS` / `RATE_LIMIT_GLOBAL_BURST` | `100 / 200` | Глобальный лимит на `/api/*` |
| `RATE_LIMIT_AUTH_PER_MIN` / `RATE_LIMIT_AUTH_BURST` | `20 / 5` | Жёсткий лимит на `/api/auth/*` (антибрутфорс) |
| `CORS_ORIGINS` | `*` | Через запятую: `https://app.example.com,https://admin.example.com` |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | пусто | Если оба заданы — сервер слушает HTTPS (TLS ≥ 1.2) |
| `UPLOAD_PATH` / `SERVER_BASE_URL` | `./uploads` / `http://localhost:8080` | Путь и базовый URL для аватаров |

## API

Все ответы — JSON. Стандартный конверт ошибок:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "details": {...} } }
```

Любой запрос возвращает заголовок `X-Request-ID` (UUID или присланный клиентом).

### Auth (публичные)

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/auth/register` | Регистрация (email + password) |
| `POST` | `/api/auth/login` | Вход, возвращает access + refresh |
| `POST` | `/api/auth/refresh` | Обновление пары токенов (refresh ротируется) |
| `POST` | `/api/auth/forgot-password` | Запрос ссылки на сброс пароля |
| `POST` | `/api/auth/reset-password` | Применение токена сброса |

`/auth/*` под жёстким rate-limiter (защита от брутфорса).

### Auth (с Bearer-токеном)

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/api/auth/logout` | Инвалидация refresh-токена |
| `GET` | `/api/users/me` | Профиль текущего пользователя |
| `PUT` | `/api/users/me` | Обновление профиля |
| `POST` | `/api/users/me/avatar` | Загрузка аватара (multipart, JPG/PNG ≤ 5 MB) |

### Admin (требует роль `admin`)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/admin/users?page=&limit=&include_deleted=` | Список пользователей |
| `GET` | `/api/admin/users/:id` | Получить пользователя |
| `PATCH` | `/api/admin/users/:id/block` | `{"blocked": true\|false}` |
| `PATCH` | `/api/admin/users/:id/role` | `{"role": "user\|moderator\|admin"}` |
| `DELETE` | `/api/admin/users/:id` | Мягкое удаление |

При блокировке, смене роли и сбросе пароля сервер инкрементирует
`token_version` пользователя, что **мгновенно** инвалидирует все его ранее
выпущенные access-токены (без ожидания истечения exp).

Полный спецификации — в Swagger UI: `/swagger/index.html`.

## Безопасность

- **bcrypt** для хеша пароля (DefaultCost).
- **JWT HS256** с уникальным `jti` и `token_version` в claims.
- **Refresh-токены** хранятся в БД, ротируются при `/auth/refresh`, удаляются
  при logout, блокировке, смене пароля или роли.
- **Auth middleware** при каждом запросе проверяет: подпись JWT, что
  пользователь не удалён, не заблокирован, и `token_version` актуален.
- **RBAC** через `RequireRole(roles...)` middleware.
- **Rate limiting** per-IP (token bucket из `golang.org/x/time/rate`).
- **HTTPS**: при заданных `TLS_CERT_FILE` + `TLS_KEY_FILE` — TLS ≥ 1.2,
  legacy 1.0/1.1 отбиваются на handshake.
- **Audit-лог** через zap: все чувствительные операции (login success/failure,
  password reset, block/unblock, role change, delete) пишутся со структурой
  `{audit:true, event, request_id, client_ip, ...}`.
- Защита от **account enumeration**: `/auth/forgot-password` отвечает
  одинаково для существующих и несуществующих email.
- Фоновая горутина раз в час чистит истёкшие refresh- и password-reset-токены.

## Тестирование

```bash
go test ./...               # все тесты
go test -cover ./...        # с покрытием
go test -run TestHandler_   ./internal/handler/...   # один пакет/префикс
```

78 unit-тестов: AuthService, AdminService, validator, RateLimiter, RequestID,
CORS, Auth+RequireRole middleware, handler-слой через `httptest`.

## Полезные make-таргеты

```bash
make run                    # go run ./cmd/api
make build                  # bin/api
make tidy                   # go mod tidy
make migrate-up             # все *.up.sql на локальный psql
make migrate-down           # все *.down.sql в обратном порядке
make swagger                # перегенерация docs/ (требуется CLI swag)
make compose-up             # docker compose up -d (только postgres)
make compose-migrate-up     # миграции внутри контейнера postgres
make compose-down           # остановить compose
make compose-psql           # psql внутри контейнера
```

## Реализованные требования ТЗ §4.1

| Требование | Статус |
|---|---|
| Регистрация пользователей | ✅ `POST /api/auth/register` |
| Аутентификация и авторизация (access + refresh) | ✅ `/auth/login`, `/auth/refresh`, JWT с `jti` и `token_version` |
| Управление ролями и правами доступа | ✅ `RequireRole`, `PATCH /admin/users/:id/role` |
| Восстановление пароля | ✅ `/auth/forgot-password` + `/auth/reset-password`, одноразовый токен, TTL 1 час |
| Управление профилями | ✅ `GET/PUT /users/me`, `POST /users/me/avatar` |
| Блокировка и удаление учётных записей | ✅ `/admin/users/:id/block`, `DELETE /admin/users/:id` |
| Защита API от несанкционированного доступа | ✅ Auth middleware + token_version + опц. TLS 1.2+ |
| Rate limiting | ✅ token-bucket per IP, отдельно жёсткий на `/auth/*` |
| Unit-тесты | ✅ 78 кейсов |
