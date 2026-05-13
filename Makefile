CONFIG_PATH=./config/config.yaml

run:
	CONFIG_PATH=${CONFIG_PATH} go run ./cmd/api/main.go
