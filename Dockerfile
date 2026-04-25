# --- сборка ---------------------------------------------------------------
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Кэшируем зависимости отдельным слоем.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

# --- финальный образ -----------------------------------------------------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 app
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /src/migrations /app/migrations
COPY --from=builder /src/docs /app/docs

USER app
EXPOSE 8080
ENTRYPOINT ["/app/api"]
