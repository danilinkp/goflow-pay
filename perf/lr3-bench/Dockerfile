ARG GO_IMAGE
FROM ${GO_IMAGE} AS builder

WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOTOOLCHAIN=local GOWORK=off go build -ldflags="-s -w" -o /bench ./cmd/bench

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /bench /app/bench

ENTRYPOINT ["/app/bench"]
