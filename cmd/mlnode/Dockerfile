FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/cmd/mlnode
RUN go build -o /mlnode

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /mlnode /app/mlnode

# El puerto interno del nodo
EXPOSE 9001

CMD ["/app/mlnode"]
