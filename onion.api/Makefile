include .env
export

DB_URL := $(DATABASE_URL)

.PHONY: migrate-up migrate-down migrate-reset migrate-status sqlc-generate

migrate-up:
	goose -dir persistence/migrations postgres $(DB_URL) up

migrate-down:
	goose -dir persistence/migrations postgres $(DB_URL) down

migrate-reset:
	goose -dir persistence/migrations postgres $(DB_URL) reset

migrate-status:
	goose -dir persistence/migrations postgres $(DB_URL) status

sqlc-generate:
	cd persistence && sqlc generate
