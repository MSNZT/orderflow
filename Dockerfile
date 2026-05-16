FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/orderflow-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bin/orderflow-api /app/orderflow-api

COPY --from=builder /app/config /app/config

EXPOSE 8080

CMD ["./orderflow-api"]