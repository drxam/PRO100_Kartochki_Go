.PHONY: run build migrate swagger tidy

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	psql -U postgres -d mozgoemka -f migrations/001_init.up.sql

migrate-down:
	psql -U postgres -d mozgoemka -f migrations/001_init.down.sql

swagger:
	swag init -g cmd/api/main.go --parseDependency --parseInternal

tidy:
	go mod tidy
