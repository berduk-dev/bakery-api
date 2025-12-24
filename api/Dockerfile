# --- Stage 1: Build ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

# goose для миграций
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# зависимости (кэшируются отдельно)
COPY go.mod go.sum ./
RUN go mod download

# код
COPY . .

# CGO не требуется для PostgreSQL драйвера (pure Go)
ENV CGO_ENABLED=0

# собираем бинарник API
RUN go build -o server .

# --- Stage 2: Runtime ---
FROM alpine:3.20

WORKDIR /app

# копируем бинарники
COPY --from=builder /go/bin/goose .
COPY --from=builder /app/server .

# копируем миграции
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations

# порт приложения
EXPOSE 8080

# Запуск сервера
CMD ["./server"]
