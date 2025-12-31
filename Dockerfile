# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

WORKDIR /app

# Install build dependencies (git needed for some Go modules)
RUN apk add --no-cache git

# Install goose for migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.11.0

# Copy go mod and sum files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (cached separately from source code)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy source code
COPY . .

# Build the application with optimizations and version information
# -ldflags="-w -s" strips debug info and symbol table
# Embed version, build time, and git commit in the binary
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s \
      -X github.com/osse101/BrandishBot_Go/internal/handler.Version=${VERSION} \
      -X github.com/osse101/BrandishBot_Go/internal/handler.BuildTime=${BUILD_TIME} \
      -X github.com/osse101/BrandishBot_Go/internal/handler.GitCommit=${GIT_COMMIT}" \
    -o brandishbot ./cmd/app

# Runtime stage - minimal image
FROM alpine:3.19

# Build arguments (passed from build stage)
ARG VERSION=dev

# Add image metadata
LABEL org.opencontainers.image.title="BrandishBot"
LABEL org.opencontainers.image.description="Discord bot for Brandish game progression and economy"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="BrandishBot Project"

WORKDIR /app

# Install only essential runtime dependencies
# ca-certificates: for HTTPS connections
# tzdata: for timezone support
# postgresql-client: for pg_isready in entrypoint
RUN apk add --no-cache ca-certificates tzdata postgresql-client && \
    # Create non-root user for security
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    # Create directories with proper permissions
    mkdir -p /app/migrations && \
    chown -R appuser:appuser /app

# Copy binaries from builder
COPY --from=builder --chown=appuser:appuser /app/brandishbot .
COPY --from=builder --chown=appuser:appuser /go/bin/goose /usr/local/bin/goose

# Copy migrations and entrypoint
COPY --chown=appuser:appuser migrations ./migrations
COPY --chown=appuser:appuser configs ./configs
COPY --chown=appuser:appuser scripts/docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Command to run
ENTRYPOINT ["./docker-entrypoint.sh"]
