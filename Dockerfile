# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/bot ./cmd/bot

# Final stage
FROM alpine:3.20

LABEL maintainer="set-night"

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/bot .
COPY --from=builder /build/assets ./assets
COPY --from=builder /build/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

ENV DOCKER=true

EXPOSE 3000

CMD ["./bot"]
