FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o auth_server ./cmd/auth_server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/certs ./certs
COPY --from=builder /app/config ./config

EXPOSE 50051

CMD ["./auth_server"]