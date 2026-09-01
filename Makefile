.PHONY: up down logs build migrate-up migrate-down migrate-create test race

# Load .env file if present
ifneq ( $(wildcard .env),)
	include .env
	export
endif

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f api

build:
	go build -o bin/api ./cmd/api

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

test:
	go test ./...

race:
	go test -race ./...