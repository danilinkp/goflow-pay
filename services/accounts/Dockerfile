FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o account_service ./cmd/account_server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/account_service .
COPY --from=builder /app/config ./config

EXPOSE 50051

CMD ["./account_service"]