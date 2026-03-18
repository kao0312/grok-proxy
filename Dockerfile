FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o grok-proxy .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/grok-proxy .

# proxies.txt 可通过 docker run -v ./proxies.txt:/app/proxies.txt 挂载
VOLUME ["/app/proxies.txt"]

EXPOSE 8080

CMD ["./grok-proxy"]
