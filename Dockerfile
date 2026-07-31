# Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build standard statically linked binary (pure Go, CGO disabled)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o auth-cli ./cmd/app/main.go

# Production Stage
FROM alpine:latest

WORKDIR /app

# Create directory for persistent SQLite storage
RUN mkdir -p /app/data

# Copy binary from builder
COPY --from=builder /app/auth-cli .

ENV DB_PATH=/app/data/app.db

ENTRYPOINT ["./auth-cli"]