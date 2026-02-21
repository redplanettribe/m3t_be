# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy module manifests and download dependencies (cached when unchanged)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build static binary for Linux
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/api

# Stage 2: Runtime (Alpine + bash for Sevalla Web Terminal)
FROM alpine:3.21

RUN apk --no-cache add ca-certificates bash

WORKDIR /app

# Copy the compiled binary from builder (fix: binary is named 'server')
COPY --from=builder /app/server /app/server

# Sevalla sets PORT at runtime; app uses os.Getenv("PORT") and defaults to 8080
EXPOSE 8080

ENTRYPOINT ["/app/server"]
