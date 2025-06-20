# ───── Stage 1: Build ─────
FROM golang:1.24.3-alpine3.21 as builder

WORKDIR /app
RUN apk add --no-cache git

COPY . .

RUN go build -o order-service ./main.go

# ───── Stage 2: Minimal Runtime ─────
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/order-service .

COPY --from=builder /app/config ./config

CMD ["./order-service"]
