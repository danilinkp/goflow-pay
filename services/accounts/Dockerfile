FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/accounts/go.mod services/accounts/go.sum ./services/accounts/

WORKDIR /app/services/accounts
RUN go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/accounts/ ./services/accounts/

WORKDIR /app/services/accounts
RUN CGO_ENABLED=0 GOOS=linux go build -o account_service ./cmd/account_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/accounts/account_service .
COPY --from=builder /app/services/accounts/config ./config

EXPOSE 50051
CMD ["./account_service"]