.PHONY: run test fmt tidy

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy