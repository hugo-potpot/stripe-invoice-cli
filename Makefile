-include .env
export

MIGRATIONS_DIR := internal/store/migrations
GOOSE          := goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)"

.PHONY: migrate-up migrate-down migrate-status migrate-reset migrate-create

migrate-up:
	$(GOOSE) up

migrate-down:
	$(GOOSE) down

migrate-status:
	$(GOOSE) status

migrate-reset:
	$(GOOSE) reset

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=<nom>"; exit 1)
	$(GOOSE) create $(name) sql