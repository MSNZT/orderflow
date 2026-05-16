.PHONY: run test fmt tidy start-app stop-app

CONFIG_PATH=./config/local.yaml

run:
	CONFIG_PATH=${CONFIG_PATH} go run ./cmd/api

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