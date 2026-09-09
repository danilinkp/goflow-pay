FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

WORKDIR /app/services/auth
RUN --mount=type=cache,target=/go/pkg/mod go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/auth/ ./services/auth/

WORKDIR /app/services/auth
RUN  --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o auth_service ./cmd/auth_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/auth/auth_service .

EXPOSE 50051
CMD ["./auth_service"]