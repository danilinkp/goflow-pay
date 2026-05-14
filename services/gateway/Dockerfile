FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/

WORKDIR /app/services/gateway
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY certs/ ./certs/
COPY services/gateway/ ./services/gateway/

WORKDIR /app/services/gateway
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o gateway_server ./cmd/gateway

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/gateway/gateway_server .
COPY --from=builder /app/certs ./certs

EXPOSE 8080

CMD ["./gateway_server"]