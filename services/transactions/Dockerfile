FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/transactions/go.mod services/transactions/go.sum ./services/transactions/

WORKDIR /app/services/transactions
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/transactions/ ./services/transactions/

WORKDIR /app/services/transactions
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o transaction_service ./cmd/transaction_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/transactions/transaction_service .

EXPOSE 50051
CMD ["./transaction_service"]