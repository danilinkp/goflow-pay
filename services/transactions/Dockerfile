FROM golang:1.25.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o transaction_service ./cmd/transaction_server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/transaction_service .
COPY --from=builder /app/config ./config

EXPOSE 50051

CMD ["./transaction_service"]