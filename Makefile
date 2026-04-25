.PHONY: run build migrate-up migrate-down swagger tidy compose-up compose-down compose-logs compose-migrate-up compose-migrate-down compose-psql

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	psql -U postgres -d pro100_kartochki -f migrations/001_init.up.sql
	psql -U postgres -d pro100_kartochki -f migrations/002_users_access.up.sql
	psql -U postgres -d pro100_kartochki -f migrations/003_password_reset.up.sql
	psql -U postgres -d pro100_kartochki -f migrations/004_token_version.up.sql

migrate-down:
	psql -U postgres -d pro100_kartochki -f migrations/004_token_version.down.sql
	psql -U postgres -d pro100_kartochki -f migrations/003_password_reset.down.sql
	psql -U postgres -d pro100_kartochki -f migrations/002_users_access.down.sql
	psql -U postgres -d pro100_kartochki -f migrations/001_init.down.sql

swagger:
	swag init -g cmd/api/main.go --parseDependency --parseInternal

tidy:
	go mod tidy

# --- Docker compose (локальный Postgres) -----------------------------------

compose-up:
	docker compose up -d

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f postgres

# Применение миграций внутри контейнера postgres.
compose-migrate-up:
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/001_init.up.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/002_users_access.up.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/003_password_reset.up.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/004_token_version.up.sql

compose-migrate-down:
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/004_token_version.down.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/003_password_reset.down.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/002_users_access.down.sql
	docker compose exec -T postgres psql -U postgres -d pro100_kartochki -f /migrations/001_init.down.sql

compose-psql:
	docker compose exec postgres psql -U postgres -d pro100_kartochki
