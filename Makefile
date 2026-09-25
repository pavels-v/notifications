.PHONY: build swagger lint fmt hooks test test-integration up down db-up db-down migrate seed logs tidy

# --project-directory . keeps the build context and .env at the repository root.
COMPOSE := docker compose -f deployments/docker-compose.yml --project-directory .
TOOL := go tool -modfile=tools/go.mod

# bin/ because a binary named migrations would otherwise be written into the migrations/ directory.
build: swagger
	go build -o bin/api ./cmd/api
	go build -o bin/migrations ./cmd/migrations

swagger:
	$(TOOL) swag init -g cmd/api/main.go -o api/v1 --instanceName v1 --parseInternal --outputTypes go,json,yaml

lint:
	$(TOOL) golangci-lint run

fmt:
	$(TOOL) golangci-lint fmt

hooks:
	git config core.hooksPath githooks

test:
	go test ./...

test-integration: db-up
	go test ./... -tags=integration -count=1

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f

db-up:
	$(COMPOSE) up -d --wait db

db-down:
	$(COMPOSE) down -v

migrate:
	$(COMPOSE) run --rm migrations

seed:
	$(COMPOSE) exec -T db psql -v ON_ERROR_STOP=1 -U notifications -d notifications < migrations/seed/seed.sql

tidy:
	go mod tidy
	go -C tools mod tidy
