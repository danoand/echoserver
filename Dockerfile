# Build stage
FROM golang:latest AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
# -a: force rebuild of packages (ensure static linking)
# -installsuffix cgo: ensure static binary
# -ldflags: strip debug symbols to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -o echoserver .

# Runtime stage
FROM alpine:latest

# Install minimal dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder --chown=appuser:appuser /app/echoserver .

# Switch to non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 8999

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8999/health || exit 1

# Run the application
CMD ["./echoserver"]
