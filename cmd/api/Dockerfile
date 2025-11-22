FROM golang:1.22-alpine AS builder

WORKDIR /app

# Módulos
COPY go.mod go.sum ./
RUN go mod download

# Código
COPY . .

# Construir binario de la API
WORKDIR /app/cmd/api
RUN go build -o /api

# Imagen final
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /api /app/api

# Variables típicas (puedes sobreescribirlas en docker-compose)
ENV HTTP_PORT=8080

EXPOSE 8080

CMD ["/app/api"]
