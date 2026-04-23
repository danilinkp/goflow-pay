FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

WORKDIR /app/services/auth
RUN go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY certs/ ./certs/
COPY services/auth/ ./services/auth/

WORKDIR /app/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -o auth_service ./cmd/auth_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/auth/auth_service .
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/services/auth/config ./config

EXPOSE 50051
CMD ["./auth_service"]