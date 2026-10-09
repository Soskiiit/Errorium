# Build stage
FROM golang:1.27.1-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./cmd/main.go

# Run stage
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata curl

COPY --from=builder /app/gateway /app/gateway

EXPOSE 8000

ENTRYPOINT ["/app/gateway"]
