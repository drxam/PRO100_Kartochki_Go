# PRO100_Kartochki — Backend

REST API для образовательной платформы «PRO100_Карточки» на Go (Clean Architecture).
Реализованы два серверных модуля из ТЗ:

- **«Пользователи и доступ»** (ТЗ §4.1) — регистрация, JWT, роли, восстановление пароля, блокировка, audit-лог, rate limiting, TLS.
- **«Учебный контент»** — полный CRUD карточек, наборов, категорий и тегов; публичные/приватные наборы; прогресс обучения (SM-2); сессии изучения; избранное; копирование наборов.

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

1. Поднять PostgreSQL и создать БД `pro100_kartochki`:

   ```bash
   psql -U postgres -c "CREATE DATABASE pro100_kartochki;"
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
/migrations            — SQL-миграции (001…006)
/pkg
  /sm2                 — алгоритм интервального повторения SuperMemo SM-2
/docs                  — Swagger/OpenAPI (генерируется `make swagger`)
```

## Конфигурация (env)

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `SERVER_PORT` | `8080` | Порт HTTP/HTTPS-сервера |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE` | `localhost / 5432 / postgres / postgres / pro100_kartochki / disable` | Реквизиты Postgres (или используйте `DATABASE_DSN` целиком) |
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

### Учебный контент (публичные, без авторизации)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/categories` | Список категорий |
| `GET` | `/api/categories/:id` | Категория по ID |
| `GET` | `/api/tags?search=` | Список тегов (с поиском) |
| `GET` | `/api/tags/:id` | Тег по ID |
| `GET` | `/api/public/decks?page=&limit=&category_id=&search=&sort_by=` | Публичные наборы |
| `GET` | `/api/public/decks/:id` | Публичный набор с карточками |

### Учебный контент (с Bearer-токеном)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/decks?page=&limit=&category_id=&search=` | Мои наборы |
| `POST` | `/api/decks` | Создать набор |
| `GET` | `/api/decks/:id` | Набор по ID |
| `PUT` | `/api/decks/:id` | Обновить набор |
| `DELETE` | `/api/decks/:id` | Удалить набор |
| `POST` | `/api/public/decks/:id/copy` | Скопировать публичный набор себе |
| `GET` | `/api/decks/:id/cards` | Карточки набора |
| `GET` | `/api/cards?page=&limit=&category_id=&tag_id=&search=` | Все мои карточки |
| `POST` | `/api/cards` | Создать карточку (`deck_id` в body) |
| `POST` | `/api/decks/:id/cards` | Создать карточку в наборе |
| `GET` | `/api/cards/:id` | Карточка по ID |
| `PUT` | `/api/cards/:id` | Обновить карточку |
| `DELETE` | `/api/cards/:id` | Удалить карточку |
| `POST` | `/api/categories` | Создать категорию |
| `POST` | `/api/tags` | Создать тег |

### Прогресс и обучение (с Bearer-токеном)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/decks/:id/progress` | Статистика прогресса по набору |
| `POST` | `/api/decks/:id/study/start` | Начать сессию обучения |
| `GET` | `/api/study/sessions` | История сессий |
| `GET` | `/api/study/sessions/:id` | Сессия по ID |
| `POST` | `/api/study/sessions/:id/review` | Ответить на карточку (quality 0–5, SM-2) |
| `POST` | `/api/study/sessions/:id/finish` | Завершить сессию досрочно |

#### Алгоритм SM-2

Оценки (`quality`):

| 0 | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| Полный провал | Неверно, трудно | Неверно, легко | Верно с трудом | Верно с паузой | Идеальный ответ |

При `quality ≥ 3` интервал растёт (1 → 6 → N×EF дней). При `< 3` — сброс.
Карточка переходит в `mastered` после ≥ 3 успешных повторений с интервалом ≥ 21 дня.

### Избранное (с Bearer-токеном)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/favorites?page=&limit=` | Избранные наборы |
| `POST` | `/api/decks/:id/favorite` | Добавить в избранное |
| `DELETE` | `/api/decks/:id/favorite` | Убрать из избранного |

### Admin (требует роль `admin`)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/api/admin/users?page=&limit=&include_deleted=` | Список пользователей |
| `GET` | `/api/admin/users/:id` | Получить пользователя |
| `PATCH` | `/api/admin/users/:id/block` | `{"blocked": true\|false}` |
| `PATCH` | `/api/admin/users/:id/role` | `{"role": "user\|moderator\|admin"}` |
| `DELETE` | `/api/admin/users/:id` | Мягкое удаление |
| `DELETE` | `/api/admin/decks/:id` | Удалить любой набор (модерация) |
| `PUT` | `/api/admin/categories/:id` | Переименовать категорию |
| `DELETE` | `/api/admin/categories/:id` | Удалить категорию |
| `PUT` | `/api/admin/tags/:id` | Переименовать тег |
| `DELETE` | `/api/admin/tags/:id` | Удалить тег |

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

~130 unit-тестов: AuthService, AdminService, CardService, DeckService, CategoryService,
TagService, validator, RateLimiter, RequestID, CORS, Auth+RequireRole middleware, handler-слой через `httptest`.

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

## Примеры запросов

### Начать сессию обучения

```bash
curl -X POST http://localhost:8080/api/decks/1/study/start \
  -H "Authorization: Bearer <token>"

# Response:
# { "session_id": 42, "total_cards": 10, "card": {"id": 5, "question": "Что такое горутина?", "answer": "..."} }
```

### Ответить на карточку

```bash
curl -X POST http://localhost:8080/api/study/sessions/42/review \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"card_id": 5, "quality": 4}'

# Response: следующая карточка или итоги сессии
```

### Скопировать публичный набор

```bash
curl -X POST http://localhost:8080/api/public/decks/7/copy \
  -H "Authorization: Bearer <token>"
# → 201, копия набора в вашей коллекции
```

### Добавить в избранное

```bash
curl -X POST http://localhost:8080/api/decks/7/favorite \
  -H "Authorization: Bearer <token>"
# → { "deck_id": 7, "is_favorite": true }
```

