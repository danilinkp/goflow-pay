FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/accounts/go.mod services/accounts/go.sum ./services/accounts/

WORKDIR /app/services/accounts
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/accounts/ ./services/accounts/

WORKDIR /app/services/accounts
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o account_service ./cmd/account_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/accounts/account_service .

EXPOSE 50051
CMD ["./account_service"]