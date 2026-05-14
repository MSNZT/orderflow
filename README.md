# OrderFlow

Backend service for processing marketplace orders.

## Run

make run

## Healthcheck

curl http://localhost:8080/health/live

## Configuration

HTTP_ADDR=:8080
HTTP_TIMEOUT=4s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=30s