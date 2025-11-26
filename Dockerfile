# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Install goose for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.11.0

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o brandishbot ./cmd/app

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata bash postgresql-client

# Copy binaries from builder
COPY --from=builder /app/brandishbot .
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy entrypoint script
COPY scripts/docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Expose port
EXPOSE 8080

# Command to run
ENTRYPOINT ["./docker-entrypoint.sh"]
