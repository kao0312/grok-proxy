FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o grok-proxy .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/grok-proxy .

EXPOSE 8080

CMD ["./grok-proxy"]
