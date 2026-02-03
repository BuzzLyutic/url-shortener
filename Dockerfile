# Stage 1 - Сборка
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Установка сертификатов для https запросов
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /shortener ./cmd/shortener

# Stage 2 - Runtime

FROM alpine:3.19

WORKDIR /app

# Установка сертификатов для https и данные о данные о часовом поясе
RUN apk add --no-cache ca-certificates tzdata

# Создание нового пользователя (не root)
RUN adduser -D -g '' appuser

COPY --from=builder /shortener /app/shortener

COPY migrations /app/migrations

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Запуск
ENTRYPOINT [ "/app/shortener" ]
