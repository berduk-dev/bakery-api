# --- Stage 1: Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o bot ./cmd/bot

# --- Stage 2: Runtime ---
FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/bot .

CMD ["./bot"]
