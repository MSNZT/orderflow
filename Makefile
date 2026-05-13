.PHONY: run, test, fmt, tidy

run:
	go run ./cmd/api/main.go

test:
	go test ./...

fmt:
	go fmt ./..

tidy:
	go mod tidy