FROM golang:1.20-alpine AS builder

WORKDIR /build
COPY src src
COPY main.go .
COPY go.mod .
COPY go.sum .

RUN go build

FROM alpine:3.17

WORKDIR /app
COPY --from=builder /build/verve-challenge-promotions /app/verve-challenge-promotions
COPY wait-for-it.sh /app/
COPY entrypoint.sh /app/

# Required for wait-for-it
RUN apk add bash

RUN chmod +x /app/verve-challenge-promotions /app/wait-for-it.sh /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
