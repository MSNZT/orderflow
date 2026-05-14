# OrderFlow

Backend service for processing marketplace orders.

## Run

make run

## Healthcheck

curl http://localhost:8080/health/live

## Configuration

HTTP_ADDR=:8080
HTTP_READ_TIMEOUT=10s
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=30s

## PostgreSQL

```bash
docker compose up -d postgres
docker compose exec postgres psql -U orderflow -d orderflow