CONFIG_PATH ?= ./config/local.yaml

.PHONY: run build test fmt tidy
.PHONY: postgres-up postgres-down psql
.PHONY: compose-up compose-down compose-ps
.PHONY: logs-api logs-postgres
.PHONY: health-live health-ready

run:
	CONFIG_PATH=$(CONFIG_PATH) go run ./cmd/api

build:
	go build -o bin/orderflow-api ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

postgres-up:
	docker compose up -d postgres

postgres-down:
	docker compose stop postgres

psql:
	docker compose exec postgres psql -U orderflow -d orderflow

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

compose-ps:
	docker compose ps

logs-api:
	docker compose logs -f api

logs-postgres:
	docker compose logs -f postgres

health-live:
	curl http://localhost:8080/health/live

health-ready:
	curl http://localhost:8080/health/ready