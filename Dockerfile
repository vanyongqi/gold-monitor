FROM --platform=linux/amd64 golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /out/gold-monitor ./cmd/gold-monitor

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/gold-monitor /app/gold-monitor

ENV GOLD_MONITOR_LISTEN=:8080
ENV GOLD_MONITOR_DB_PATH=/app/storage/gold_monitor.db

VOLUME ["/app/storage"]
EXPOSE 8080

ENTRYPOINT ["./gold-monitor", "-http", "-listen", ":8080", "-db", "/app/storage/gold_monitor.db"]
