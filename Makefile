CONFIG_PATH ?= ./config/local.yaml

.PHONY: run build test fmt tidy
.PHONY: start-app stop-app
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

start-app:
	docker-compose up --build -d

stop-app:
	docker-compose down

logs-api:
	docker-compose logs -f api

logs-postgres:
	docker-compose logs -f postgres

health-live:
	curl http://localhost:8080/health/live

health-ready:
	curl http://localhost:8080/health/ready