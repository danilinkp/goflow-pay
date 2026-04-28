FROM golang:1.25.4-alpine AS builder

WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/notifications/go.mod services/notifications/go.sum ./services/notifications/

WORKDIR /app/services/notifications
RUN go mod download

WORKDIR /app
COPY shared/ ./shared/
COPY services/notifications/ ./services/notifications/

WORKDIR /app/services/notifications
RUN CGO_ENABLED=0 GOOS=linux go build -o notification_service ./cmd/notification_server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/services/notifications/notification_service .

EXPOSE 50051
CMD ["./notification_service"]