# OrderFlow

Backend service for processing marketplace orders.

## Stack

- Go
- chi
- PostgreSQL
- pgxpool
- Docker Compose
- slog

## Setup

Create local environment file:

```bash
cp .env.example .env
```

## Run with Docker Compose

Start API and PostgreSQL:

```bash
make start-app
```

Stop services:

```bash
make stop-app
```

Check health:

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## Run locally

Start PostgreSQL only:

```bash
docker compose up -d postgres
```

Run API on host:

```bash
make run
```

Local run uses:

```txt
config/local.yaml
```

For local run, PostgreSQL host is `localhost`.

For Docker Compose run, PostgreSQL host is `postgres`.

## Build

```bash
make build
```

The binary is created at:

```txt
bin/orderflow-api
```

## PostgreSQL

Connect to database:

```bash
docker compose exec postgres psql -U orderflow -d orderflow
```

## Make commands

```bash
make run        # run API locally
make build      # build API binary
make test       # run tests
make fmt        # format Go code
make tidy       # clean Go modules
make start-app  # start Docker Compose services
make stop-app   # stop Docker Compose services
```

## Status

Implemented:

- HTTP server bootstrap
- graceful shutdown
- YAML/env config
- structured logging
- PostgreSQL connection pool
- Docker Compose
- `/health/live`
- `/health/ready`