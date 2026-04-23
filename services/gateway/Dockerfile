FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/

WORKDIR /app/services/gateway
RUN go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY certs/ ./certs/
COPY services/gateway/ ./services/gateway/

WORKDIR /app/services/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway_server ./cmd/gateway

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/gateway/gateway_server .
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/services/gateway/config ./config

EXPOSE 8080

CMD ["./gateway_server"]