FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/transactions/go.mod services/transactions/go.sum ./services/transactions/

WORKDIR /app/services/transactions
RUN go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/transactions/ ./services/transactions/

WORKDIR /app/services/transactions
RUN CGO_ENABLED=0 GOOS=linux go build -o transaction_service ./cmd/transaction_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/transactions/transaction_service .
COPY --from=builder /app/services/transactions/config ./config

EXPOSE 50051
CMD ["./transaction_service"]