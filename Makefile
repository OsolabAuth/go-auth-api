.PHONY: up down logs db-up dev migrate-up migrate-down test coverage build

up:
	docker compose up -d --build

down:
	docker compose --profile tools down

logs:
	docker compose logs -f api

db-up:
	docker compose up -d db

dev:
	@set -a; if [ -f .env ]; then . ./.env; fi; set +a; exec go run ./cmd/api

migrate-up:
	docker compose --profile tools run --rm migrate up

migrate-down:
	docker compose --profile tools run --rm migrate down 1

test:
	go test ./...

coverage:
	sh scripts/coverage.sh

build:
	docker build -f docker/Dockerfile.prod -t go-auth-api:local .
